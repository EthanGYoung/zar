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
	// Parameter (dn)               : name of current directory
	// parameter (rootdir)			: path of parent dir relative to root of image
	// parameter (basedir)       	: absolute path of parent directory to dn
	// parameter (root)             : whether or not dir is the root dir
	WalkDir(dn, rootdir, basedir string, root bool)

	// AddDirectory either updates number of directories or adds name to the filter.
	// If stat is false and current directory is a symlink, adds metadata for dir symlinks
	//
	// parameter (name)     : name of the current directory
	// parameter (rootdir)	: path of parent dir relative to root of image
	// parameter (basedir)  : absolute path of parent directory to dn
	// parameter (stat)		: (true) - Increments element count, else adds path to filter 
	AddDirectory(dn, rootdir, basedir string, stat bool)

	// AddFile either updates number of elements or adds file path to filter
	//
	// parameter (fn)       : name of the file to be read
	// parameter (rootdir)	: path of parent dir relative to root of image
	// paramter (stat)		: (true) - Increments element count, else adds path to filter
	AddFile(fn, rootdir string, stat bool)

	// TODO: Add IncludeWhiteoutFile and IncludeSymlink to interface

	// GenerateFilter creates a filter based on the statistics
	GenerateFilter()

	// WriterHeader writes the Metadata for the imagefile to the end of the image file.
	// The location of the beginning of the header is written at the very end as an int64
	WriteHeader() error
}

// Manager is the main driver of creating the image file. It writes the data and stores Metadata.
type ZarManager struct {
	// The FileWriter for this zar image
	Writer writer.FileWriter

	// Statistics is a ImgStats struct that tracks relevant statistics for the image file
	Statistics *stats.ImgStats

	// FilterMetadata is a struct refering to info about the filter
	FilterMetadata filter.FilterMetadata

	// Filter is a filter used for this image file
	Filter *filter.Filter
}


// WalkDir implemented Manager.WalkDir
func (z *ZarManager) WalkDir(dn, rootdir, basedir string, root bool) {
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

// IncludeFolderBegin implements Manager.IncludeFolderBegin
func (z *ZarManager) AddDirectory(dn, rootdir, basedir string, stat bool) {
	if (stat) {
		z.Statistics.AddDir()
	} else {
		fullPath := path.Join(basedir, dn)
		imgPath := path.Join(rootdir, dn)
		isSymlink := file.Mode() & os.ModeSymlink != 0
		
		if (symlink) {
			// Generate indirect path for metadata
			// Create DirSymLink metadata
		}

		// Hashing imgPath because want the path relative to root of container
		z.Filter.AddElement([]byte(imgPath))
	}
}

// IncludeFile implements Manager.IncludeFile
func (z *ZarManager) AddFile(fn, rootdir string, stat bool) {
	if (stat) {
		z.Statistics.AddFile()
	} else {
		imgPath := path.Join(rootdir, fn)

		// Hashing imgPath because want the path relative to root of container
		z.Filter.AddElement([]byte(imgPath))
	}
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
