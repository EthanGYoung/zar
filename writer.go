package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"encoding/binary"
	"encoding/gob"
	"io/ioutil"
	"log"
	"path"
	"os"
	"strings"
	"syscall"
)

type fileType int

const(
	pageBoundary = 4096	// The boundary that needs to be upheld for page alignment
	RegularFile fileType = iota
	Directory
	Symlink
 )

// TODO
// Add new struct for walking directories
// Can have one that will do a breadth first search based on current implementation
// Can have another that will walk based on a yaml file uploaded
// TODO

// fileWriter struct writes to a file
type fileWriter struct {
	// zarw is used as the writer to the file f
	zarw *bufio.Writer

	// Count is the cumulative bytes written to the file f
	count int64

	// f is the file object that the writer will write to
	f *os.File
}


// Initializes a writer by creating the image file and attaching a writer to it\
//
// Parameter (fn): Name of image file
func (w *fileWriter) Init(fn string) error {
	if w.count != 0 {
		err := "unknown error, writer counter is not 0 when initializing"
		log.Fatalf(err)
		return errors.New(err)
	}

	// Create image file
	f, err := os.Create(fn)
	if err != nil {
		log.Fatalf("can't open zar output file %v, err: %v", fn, err)
		return err
	}

	// Initiaize the buffer writer
	w.f = f
	w.zarw = bufio.NewWriter(f)
	return nil
}

// NOTE: in the new version fileWriter.Writer return the "real" end
// Write writes the data to the zar file. The caller can specify whether or not
// to keep the file page aligned
//
// parameter (data)	: the data to be written
// parameter (pageAlign): whether to page align the data
func (w *fileWriter) Write(data []byte, pageAlign bool) (int64, error) {
	// Writes to fileWriter
	n, err := w.zarw.Write(data)
	if err != nil {
		return int64(n), err
	}

	// Adds padding if last page is not page aligned
	n2 := 0
	if pageAlign {
		pad := (pageBoundary - n % pageBoundary) % pageBoundary
		fmt.Printf("current write size: %v, padding size: %v\n", n, pad)
		if pad > 0 {
			s := make([]byte, pad)
			n2, err = w.zarw.Write(s)
		}
	}

	// Updates offsets
	realEnd := w.count + int64(n)
	w.count += int64(n + n2)

	return realEnd, err
}

// WriteInt64 writes a int64 to the fileWriter
//
// parameter (v): the value to be written
func (w *fileWriter) WriteInt64(v int64) (int64, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, v)
	n, err := w.Write(buf, false)
	return n, err
}

// Close closes the filewriter by flushing any buffer
func (w *fileWriter) Close() error {
	fmt.Println("Written Bytes: ", w.count, "+ metadata size")
	w.zarw.Flush()
	return w.f.Close()
}

// Manager is an interface for creating the image file.
// This interface allows for multiple implementations of its creation.
type Manager interface {
	// WalkDir recursively traverses each directory below the root director and processes files
	// by creating metadata.
	//
	// Parameter (dir) 		: name of path relative to root dir
	// parameter (foldername) 	: name of current folder
	// parameter (root)		: whether or not dir is the root dir
	WalkDir(dir string, foldername string, root bool)

	// IncludeFolderBegin initializes metadata for the beginning of a file
	//
	// parameter (name)	: name of the file beginning
	IncludeFolderBegin(name string)

	// IncludeFolderEnd initializes metadata for the end of a file
	IncludeFolderEnd()

	// IncludeFile reads the given file, adds it to the file, and creates the metadata.
	//
	// parameter (fn)	: name of the file to be read
	// paramter (basedir)	: name of the current directory relative to root
	// return		: new offset into the image file
	IncludeFile(fn string, basedir string) (int64, error)

	// WriterHeader writes the metadata for the imagefile to the end of the image file.
	// The location of the beginning of the header is written at the very end as an int64
	WriteHeader() error
}

// zarManager is the main driver of creating the image file. It writes the data and stores metadata.
type zarManager struct {
	// pageAlign indicates whether files will be aligned at page boundaries
	pageAlign bool

	// The fileWriter for this zar image
	writer fileWriter

	// metadata is a list of fileMetadata structs indicating start and end of directories and files
	metadata []fileMetadata
}

// fileMetadata holds information for the location of a file in the image file
type fileMetadata struct {
	// Begin indicates the beginning of a file (pointer) in the file
	Begin int64

	// End indicates the ending of a file (pointer) in the file
	End int64

	// Name indicates the name of a specific file in the file
	Name string

	// If the file is a symlink, this entry is used for link info
	Link string

	// Type indicated the type of a specific file (dir, symlink or regular file)
	Type fileType
}


// WalkDir implemented Manager.WalkDir
func (z *zarManager) WalkDir(dir string, foldername string, root bool) {
	// root dir not marked as directory
	if !root {
		fmt.Printf("including folder: %v, name: %v\n", dir, foldername)
		z.IncludeFolderBegin(foldername)
	}

	// Retrieve all files in current directory
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("walk dir unknown err when processing dir %v", dir)
	}

	var dirs []string

	// Process each file in the directory
	for _, file := range files {
		name := file.Name()
		symlink := file.Mode() & os.ModeSymlink != 0
		file_path := path.Join(dir, name)

		if symlink {
			// Symbolic link is an indirection, thus read and include
			fmt.Printf("%v is symlink.", file_path)
			real_dest, err := os.Readlink(file_path)
			if err != nil {
				log.Fatalf("error. Can't read symlink file. %v", real_dest)
			}
			// TODO: Can we replace with file redirecting to here? Could eliminate symbolic links
			z.IncludeSymlink(name, real_dest)
		} else {
			if !file.IsDir() {
				fmt.Printf("including file: %v\n", name)
				z.IncludeFile(name, dir)
				} else {
						dirs = append(dirs, name)
			}
		}
	}

	// Recursively search each directory (DFS)
	// After file processing to improve spatial locatlity for files
	for _, subDir := range dirs {
		z.WalkDir(path.Join(dir, subDir), subDir, false)
	}

	// root dir not marked as directory
	if !root {
		z.IncludeFolderEnd()
	}
}

// TODO: Change to interface for metadata to have diff types of metadata
// IncludeFolderBegin implements Manager.IncludeFolderBegin
func (z *zarManager) IncludeFolderBegin(name string) {
	h := &fileMetadata{
			Begin	: -1,
			End	: -1,
			Name	: name,
			Type	: Directory,
	}

	// Add to the image's metadata at end
	z.metadata = append(z.metadata, *h)
}

// IncludeFolderEnd implements IncludeFolderEnd
func (z *zarManager) IncludeFolderEnd() {
	h := &fileMetadata{
			Begin	: -1,
			End	: -1,
			Name	: "..",
			Type	: Directory,
	}

	// Add to the image's metadata at end
	z.metadata = append(z.metadata, *h)
}

// IncludeSymlink adds metadata to the image file for a symbolic link. This
// allows for paths to be indirections. Not included in interface because
// not necessarily fundamental for correctness.
//
// parameter (name)	: name of file
// parameter (link)	: the actual path to the desired file
func (z *zarManager) IncludeSymlink(name string, link string) {
	h := &fileMetadata{
			Begin	: -1,
			End	: -1,
			Name	: name,
			Link	: link,
			Type	: Symlink,
	}
	z.metadata = append(z.metadata, *h)
}

// IncludeFile implements Manager.IncludeFile
func (z *zarManager) IncludeFile(fn string, basedir string) (int64, error) {
	content, err := ioutil.ReadFile(path.Join(basedir, fn))
	if err != nil {
		log.Fatalf("can't include file %v, err: %v", fn, err)
		return 0, nil
	}

	// Retrieve the current offset into the file and write the file contents
	oldCounter := z.writer.count
	real_end, err := z.writer.Write(content, z.pageAlign)
	if err != nil {
			log.Fatalf("can't write to file")
			return 0, err
	}

	// Create the file metadata
	h := &fileMetadata{
			Begin	: oldCounter,
			End	: real_end,
			Name	: fn,
			Type	: RegularFile,
	}
	z.metadata = append(z.metadata, *h)

	return real_end, err
}

// TODO: Is gob the best choice here?
// WriteHeader implements Manager.WriteHeader
func (z *zarManager) WriteHeader() error {
	headerLoc := z.writer.count	// Offset for metadata in image file
	fmt.Printf("header location: %v bytes\n", headerLoc)

	mEnc := gob.NewEncoder(z.writer.zarw)

	fmt.Println("current metadata:", z.metadata)
	mEnc.Encode(z.metadata)

	// Write location of metadata to end of file
	z.writer.WriteInt64(int64(headerLoc))

	if err := z.writer.Close(); err != nil {
		log.Fatalf("can't close zar file: %v", err)
	}
	return nil
}

// configManager is a struct for writing image files from a configuration file. The configuration file
// will specify which files to read (relative to root dir) and in what order to put them in img file.
type configManager struct {
	// Inherits zarManager's methods
	*zarManager

	// Format of the input file (e.g. YAML, csv, ..)
	format string

	// The configuration file with the structure of the img file specified in the format in the format field
	configFile *os.File
}

// WalkDir implements Manager.WalkDir. Overrides zarManager'si
func (c *configManager) WalkDir(dir string, foldername string, root bool) {

	switch c.format {
	case "seq":
	// seq format is as follows
	// <File (f) or Start Dir (sd) or End Dir (ed) > | < path excluding file name > | < name > \n
	// TODO: Create a file reader struct to allow for generic reading of different formats.
	// TODO: For now, prototype will always assume seq
	default:
		log.Fatalf("Config format not recognized")
	}

	// Close file once scanning is complete
	defer c.configFile.Close()
	scanner := bufio.NewScanner(c.configFile)
	scanner.Split(bufio.ScanLines)

	// Read each line in the config file
	for scanner.Scan() {
		// Parse the line TODO: Save path along way so config does not need path and name separate
		s := strings.Split(scanner.Text(), "|")
		action, path, name := s[0], s[1], s[2]

		switch action {
		case "f":
			c.IncludeFile(name, path)
		case "sd":
			c.IncludeFolderBegin(name)
		case "ed":
			c.IncludeFolderEnd()
		default:
			log.Fatalf("Config action not recognized")
		}
	}
}

// writeImage acts as the "main" method by creating and initializing the zarManager,
// beginning the recursive walk of the directories, and writing the metadata header
//
// parameter (dir)	: the root dir name
// parameter (output)	: the name of the image file
// parameter (pageAlign): whether the files in the image will be page aligned
// parameter (config)	: whether the image file is initialized from a config file
// parameter (configPath): the path to the config file
// parameter (format)	: the format of the config file
func writeImage(dir string, output string, pageAlign bool, config bool, configPath string, format string) {
	var z *zarManager
	var c *configManager

	z = &zarManager{pageAlign:pageAlign}

	// Create the manager
	// TODO: Make this not redundant code
	if config {
		// Open the config file
		f, err := os.Open(configPath)
		if err != nil {
			log.Fatalf("can't open config file %v, err: %v", configPath, err)
		}

		c = &configManager{
			zarManager	: z,
			format		: format,
			configFile	: f,
		}
		c.writer.Init(output)

		// Begin recursive walking of directories
		c.WalkDir(dir, dir, true)

		// Write the metadata to end of file
		c.WriteHeader()
	} else {
		z.writer.Init(output)

		// Begin recursive walking of directories
		z.WalkDir(dir, dir, true)

		// Write the metadata to end of file
		z.WriteHeader()
	}
}

// TODO: Break up into smaller methods
// readImage will open the given file, extract the metadata, and print out
// the structure and/or data for each file and directory in the image file.
//
// parameter (img)	: name of the image file to be read
// parameter (detail)	: whether to print extra information (file data)
func readImage(img string, detail bool) error {
	f, err := os.Open(img)
	if err != nil {
		log.Fatalf("can't open image file %v, err: %v", img, err)
		return err
	}

	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("can't stat image file %v, err: %v", img, err)
	}

	length := int(fi.Size()) // MMAP limitation. May not support large file in32 bit system
	fmt.Printf("this image file has %v bytes\n", length)

	// mmap image into address space
	mmap, err := syscall.Mmap(int(f.Fd()), 0, length, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		log.Fatalf("can't mmap the image file, err: %v", err)
	}

	if detail {
		fmt.Println("MMAP data:", mmap)
	}

	// header location is specifed by int64 at last 10 bits (bytes?)
	headerLoc := mmap[length - 10 : length]
	fmt.Println("header data:", headerLoc)

	// Setup reader for header data
	headerReader := bytes.NewReader(headerLoc)
	n, err := binary.ReadVarint(headerReader)
	if err != nil {
		log.Fatalf("can't read header location, err: %v", err)
	}
	fmt.Printf("headerLoc: %v bytes\n", n)

	var metadata []fileMetadata
	header := mmap[int(n) : length - 10]
	fmt.Println("metadata data:", header)

	// Decode the metadata in the header
	metadataReader := bytes.NewReader(header)
	dec := gob.NewDecoder(metadataReader)
	errDec := dec.Decode(&metadata)
	if errDec != nil {
		  log.Fatalf("can't decode metadata data, err: %v", errDec)
			return err
	}
	fmt.Println("metadata data decoded:", metadata)

	level := 0
	space := 2

	// Print the structure (and data) of the image file
	for _, v := range metadata {
		for i := 0; i < space * level; i++ {
			fmt.Printf(" ")
		}
		if v.Begin == -1 {
			if v.Type == Directory {
				if v.Name != ".." {
					fmt.Printf("[folder] %v\n", v.Name)
					level += 1
				} else {
					fmt.Printf("[flag] leave folder\n")
					level -= 1
				}
			} else {
				fmt.Printf("[symlink] %v -> %v\n", v.Name, v.Link)
			}
		} else {
			var fileString string
			if detail {
				fileBytes := mmap[v.Begin : v.End]
				fileString = string(fileBytes)
			} else {
				fileString = "ignored"
			}
			fmt.Printf("[regular file] %v (data: %v)\n", v.Name, fileString)
		}
	}
	return nil
}

func main() {
	// TODO: Add config file for version number
	fmt.Println("zar image generator version 1")

	// TODO: Add flag for info logging
	// Handle flags
	dir := flag.String("dir", "./", "select the root dir to generate image")
	img := flag.String("img", "test.img", "select the image to read")
	output := flag.String("o", "test.img", "output img name")
	writeMode := flag.Bool("w", false, "generate image mode")
	readMode := flag.Bool("r", false, "read image mode")
	pageAlign := flag.Bool("pagealign", false, "align the page")
	detailMode := flag.Bool("detail", false, "show original context when read")
	config := flag.Bool("config", false, "img generated from config file")
	configPath := flag.String("configPath", "", "path to config file for img")
	configFormat := flag.String("configFormat", "seq", "format of config. Known: seq")
	flag.Parse()

	// TODO: Create a config struct for all flags
	if *writeMode {
		fmt.Printf("root dir: %v\n", *dir)
		writeImage(*dir, *output, *pageAlign, *config, *configPath, *configFormat)
	}

	if (*readMode) {
		fmt.Printf("img selected: %v\n", *img)
		readImage(*img, *detailMode)
	}
}
