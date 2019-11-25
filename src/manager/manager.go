// Package manager implements a library for constructing an image file
package manager


import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"fileio/writer"
	"filter"
	"stats"
	"strings"
)

// fileType is an integer representating the file type (RegularFile, Directory, Symlink)
type fileType int

const (
	// Represent the possible file types for files
	RegularFile fileType = iota
	Directory
	Symlink
	WhiteoutFile
)

// Manager is an interface for creating the image file.
// This interface allows for multiple implementations of its creation.
type Manager interface {
        // WalkDir recursively traverses each directory below the root director and processes files
        // by creating Metadata.
        //
        // Parameter (dir)              : name of path relative to root dir
        // parameter (foldername)       : name of current folder
        // parameter (root)             : whether or not dir is the root dir
        WalkDir(dir string, foldername string, mod_time int64, mode os.FileMode, root bool)

        // IncludeFolderBegin initializes Metadata for the beginning of a file
        //
        // parameter (name)     : name of the file beginning
        IncludeFolderBegin(name string, mod_time int64, mode os.FileMode)

        // IncludeFolderEnd initializes Metadata for the end of a file
        IncludeFolderEnd()

        // IncludeFile reads the given file, adds it to the file, and creates the Metadata.
        //
        // parameter (fn)       : name of the file to be read
        // paramter (basedir)   : name of the current directory relative to root
        // return               : new offset into the image file
        IncludeFile(fn string, basedir string, mod_time int64) (int64, error)

	// TODO: Add IncludeWhiteoutFile and IncludeSymlink to interface

	// GenerateFilter creates a filter based on the files in the img file
	GenerateFilter()

        // WriterHeader writes the Metadata for the imagefile to the end of the image file.
        // The location of the beginning of the header is written at the very end as an int64
        WriteHeader() error
}

// FileMetadata holds information for the location of a file in the image file
type FileMetadata struct {
        // Begin indicates the beginning of a file (pointer) in the file
        Begin int64

        // End indicates the ending of a file (pointer) in the file
        End int64

        // Name indicates the name of a specific file in the file
        Name string

        // If the file is a symlink, this entry is used for link info
        Link string

	// File modification time
        ModTime int64

        // Type indicated the type of a specific file (dir, symlink or regular file)
        Type fileType

	// TODO: What does this do
	Mode os.FileMode
}

// Manager is the main driver of creating the image file. It writes the data and stores Metadata.
type ZarManager struct {
        // PageAlign indicates whether files will be aligned at page boundaries
        PageAlign bool

        // The FileWriter for this zar image
        Writer writer.FileWriter

        // Metadata is a list of FileMetadata structs indicating start and end of directories and files
        Metadata []FileMetadata

	// FilterMetadata is a struct refering to info about the bloom filter
	FilterMetadata filter.FilterMetadata

	// Statistics is a ImgStats struct that tracks relevant statistics for the image file
	Statistics *stats.ImgStats

	// Filter is a filter used for this image file
	Filter *filter.BloomFilter
}

type DirInfo struct {
        Name string
        ModTime int64
	Mode os.FileMode
}

// WalkDir implemented Manager.WalkDir
func (z *ZarManager) WalkDir(dir string, foldername string, mod_time int64, mode os.FileMode, root bool) {
        // root dir not marked as directory
        if !root {
                fmt.Printf("including folder: %v, name: %v\n", dir, foldername)
                z.IncludeFolderBegin(foldername, mod_time, mode)
        }

        // Retrieve all files in current directory
        files, err := ioutil.ReadDir(dir)
        if err != nil {
                log.Fatalf("walk dir unknown err when processing dir %v", dir)
        }

        var dirs []*DirInfo

        // Process each file in the directory
        for _, file := range files {
                name := file.Name()
		mode := file.Mode()
                symlink := file.Mode() & os.ModeSymlink != 0
                device := file.Mode() & os.ModeDevice != 0
		size := file.Size()
		file_path := path.Join(dir, name)
                mod_time := file.ModTime().UnixNano()

		if device {
	                  if size != 0 {
	                    log.Fatalf("character device with non-zero size is not a whiteout file.")
	                  }
	                  z.IncludeWhiteoutFile(name, mod_time)
                } else if symlink {
                        // Symbolic link is an indirection, thus read and include
                        fmt.Printf("%v is symlink.", file_path)
                        real_dest, err := os.Readlink(file_path)
                        if err != nil {
                                log.Fatalf("error. Can't read symlink file. %v", real_dest)
                        }
                        // TODO: Can we replace with file redirecting to here? Could eliminate symbolic links
                        z.IncludeSymlink(name, real_dest, mod_time, mode)
                } else {
                        if !file.IsDir() {
                                fmt.Printf("including file: %v\n", name)
                                z.IncludeFile(name, dir, mod_time, mode)
                        } else {
                                dirs = append(dirs, &DirInfo{name, mod_time, mode})
                        }
                }
        }

        // Recursively search each directory (DFS)
        // After file processing to improve spatial locatlity for files
        for _, subDir := range dirs {
                z.WalkDir(path.Join(dir, subDir.Name), subDir.Name, subDir.ModTime, subDir.Mode, false)
        }

        // root dir not marked as directory
        if !root {
                z.IncludeFolderEnd()
        }
}

// TODO: Change to interface for Metadata to have diff types of Metadata
// IncludeFolderBegin implements Manager.IncludeFolderBegin
func (z *ZarManager) IncludeFolderBegin(name string, mod_time int64, mode os.FileMode) {
        h := &FileMetadata{
		Begin   : -1,
		End     : -1,
		Name    : name,
		Type    : Directory,
		ModTime : mod_time,
		Mode	: mode,
        }

        // Add to the image's Metadata at end
        z.Metadata = append(z.Metadata, *h)

	z.Statistics.AddDir()
}

// IncludeFolderEnd implements IncludeFolderEnd
func (z *ZarManager) IncludeFolderEnd() {
        h := &FileMetadata{
                        Begin   : -1,
                        End     : -1,
                        Name    : "..",
                        Type    : Directory,
        }

        // Add to the image's Metadata at end
        z.Metadata = append(z.Metadata, *h)
}

func (z *ZarManager) IncludeWhiteoutFile(name string, mod_time int64) {
	// Create the file Metadata
	h := &FileMetadata{
		  Begin   : -1,
		  End     : -1,
		  Name    : name,
		  Type    : WhiteoutFile,
		  ModTime : mod_time,
	}
	z.Metadata = append(z.Metadata, *h)
}

// IncludeSymlink adds Metadata to the image file for a symbolic link. This
// allows for paths to be indirections. Not included in interface because
// not necessarily fundamental for correctness.
//
// parameter (name)     : name of file
// parameter (link)     : the actual path to the desired file
// parameter (mod_time) : the modification time fo the file

func (z *ZarManager) IncludeSymlink(name string, link string, modTime int64, mode os.FileMode) {
        h := &FileMetadata{
		Begin   : -1,
		End     : -1,
		Name    : name,
		Link    : link,
		Type    : Symlink,
		ModTime : modTime,
		Mode	: mode,
        }
        z.Metadata = append(z.Metadata, *h)

	z.Statistics.AddSymLink()
}

// IncludeFile implements Manager.IncludeFile
func (z *ZarManager) IncludeFile(fn string, basedir string, mod_time int64, mode os.FileMode) (int64, error) {
        content, err := ioutil.ReadFile(path.Join(basedir, fn))
        if err != nil {
                log.Fatalf("can't include file %v, err: %v", fn, err)
                return 0, nil
        }

        // Retrieve the current offset into the file and write the file contents
        oldCounter := z.Writer.Count
        real_end, err := z.Writer.Write(content, z.PageAlign)
        if err != nil {
                        log.Fatalf("can't write to file")
                        return 0, err
        }

        // Create the file Metadata
        h := &FileMetadata{
		Begin   : oldCounter,
		End     : real_end,
		Name    : fn,
		Type    : RegularFile,
		ModTime : mod_time,
		Mode	: mode,
        }
        z.Metadata = append(z.Metadata, *h)

	z.Statistics.AddFile()

        return real_end, err
}

// GenerateFilter implements manager.GenerateFilter
func (z *ZarManager) GenerateFilter() {
	// Check type of filter -> Default BloomFilter, later pass in

	// Create initial filter -> Default Bloom, but later have swithc statement
	z.Filter = &filter.BloomFilter{NumElem:z.Statistics.NumFiles}

	// Initialize filter (TODO: Check error)
	z.Filter.Initialize()

	// Construct filter
	z.constructFilter()

	// Create FilterMetadata
	z.FilterMetadata = filter.FilterMetadata{
		Active:true,
		Name:"BloomFilter", // Default to BloomFilter
	}

	// TODO: Write filter to file here instead of in Header Method
}

// ConstructFilter initializes a filter by looping over FileMetadata
// and adding each file to the filter
// Algorithm: 
//	- string to hold current path
//	- When encounter startDir, append '/dirname' to string
//	- When encounter file/symlink, hash string + filename into filter
// 	- When encounter endDir, remove previous name from string
func (z *ZarManager) constructFilter() {
	fmt.Println("Constructing Filter")

	var path = ""

	for i:=0; i < len(z.Metadata); i++ {
		name:=z.Metadata[i].Name
		switch MetaType := z.Metadata[i].Type; MetaType {
		case (RegularFile):
			// Add to filter
			z.Filter.AddElement([]byte(path + "/" + name))
		case (Directory):
			// Name = ".." means end of directory
			if (name == "..") {
				// Remove dir name at end
				intPath := strings.Split(path, "/")
				intPath = intPath[:len(intPath)-1]
				path = strings.Join(intPath, "/")
			} else {
				path += "/" + name
			}
		case (Symlink):
			// Treat like a file, add and hash
			z.Filter.AddElement([]byte(path + "/" + name))
		}
	}
}

// TODO: Is gob the best choice here? Need to use it to encode structs
// TODO: In future, can this be laid out as the struct, and directly mapped into memory?
//	YES! Use BinaryMarshaler to make custom layout
// WriteHeader implements Manager.WriteHeader
func (z *ZarManager) WriteHeader() error {
	z.WriteFileMetadata()

	z.WriteFilterMetadata()

	if err := z.Writer.Close(); err != nil {
                log.Fatalf("can't close zar file: %v", err)
        }
        return nil
}

func (z *ZarManager) WriteFileMetadata() {
        headerLoc := z.Writer.Count     // Offset for Metadata in image file
        fmt.Printf("header location: %v bytes\n", headerLoc)

	// Marshal metadata
	gob.Register(FileMetadata{})

	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(z.Metadata)
	if err != nil { fmt.Println(`failed gob Encode`, err) }

        fmt.Println("current Metadata:", z.Metadata)
	z.Writer.Write([]byte(base64.StdEncoding.EncodeToString(b.Bytes())), false) // Not pageAligned

        // Write location of Metadata to end of file
        z.Writer.WriteInt64(int64(headerLoc))

	// Flush the writer
	z.Writer.W.Flush()
}

func (z *ZarManager) WriteFilterMetadata() {
	initLoc := z.Writer.Count

	// Write filter data to file (Need to marshal Bloom Filter struct)
	gob.Register(filter.BloomFilter{})

	buf := bytes.Buffer{}
	e := gob.NewEncoder(&buf)
	err := e.Encode(z.Filter)
	if err != nil { fmt.Println(`failed gob Encode`, err) }

	fmt.Println("Writing BloomFilter:", z.Filter)
	z.Writer.Write([]byte(base64.StdEncoding.EncodeToString(buf.Bytes())), false) // Not pageAligned

	// Set size of BloomFilter
        filterLoc := z.Writer.Count     // Offset for Metadata in image file

	z.FilterMetadata.FilterLoc = initLoc
	z.FilterMetadata.FilterStructSize = filterLoc - initLoc

	// Write filter metadata to file
        fmt.Printf("filter location: %v bytes\n", filterLoc)

	// Marshal Metadata
	gob.Register(filter.FilterMetadata{})

	b := bytes.Buffer{}
	e = gob.NewEncoder(&b)
	err = e.Encode(z.FilterMetadata)
	if err != nil { fmt.Println(`failed gob Encode`, err) }

        fmt.Println("current FilterMetadata:", z.FilterMetadata)
	z.Writer.Write([]byte(base64.StdEncoding.EncodeToString(b.Bytes())), false) // Not pageAligned

	// Write location of Metadata to end of file
        z.Writer.WriteInt64(int64(filterLoc))

	// Flush the writer
	z.Writer.W.Flush()
}
