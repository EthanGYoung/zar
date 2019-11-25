package main

import (
	"bytes"
	"encoding/binary"
	"encoding/base64"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"

	// TODO: Change paths to be remotely imported from github
	"manager"
	"filter"
	"stats"
)

// writeImage acts as the "main" method by creating and initializing the manager,
// beginning the recursive walk of the directories, and writing the metadata header
//
// parameter (dir)	: the root dir name
// parameter (output)	: the name of the image file
// parameter (pageAlign): whether the files in the image will be page aligned
// parameter (config)	: whether the image file is initialized from a config file
// parameter (configPath): the path to the config file
// parameter (format)	: the format of the config file
func writeImage(dir string, output string, pageAlign bool, config bool, configPath string, format string) {
	var z *manager.ZarManager
	var c *manager.CManager

	// Initializes all fields to 0
	var stats = &stats.ImgStats{}
	var filter = &filter.BloomFilter{} // Default to BloomFilter

	z = &manager.ZarManager{
		PageAlign	: pageAlign,
		Statistics	: stats,
		Filter		: filter,
	}

	// Create the manager
	// TODO: Make this not redundant code
	if config {
		// Open the config file
		f, err := os.Open(configPath)
		if err != nil {
			log.Fatalf("can't open config file %v, err: %v", configPath, err)
		}

		c = &manager.CManager{
			ZarManager	: z,
			Format		: format,
			ConfigFile	: f,
		}
		c.Writer.Init(output)

		// Begin recursive walking of directories
		c.WalkDir(dir, dir, true)

		// Recursively construct a filter for img
		c.GenerateFilter()

		// Write the metadata to end of file
		c.WriteHeader()
	} else {
		z.Writer.Init(output)

		// Begin recursive walking of directories
		z.WalkDir(dir, dir,0, 0, true)

		// Recursively construct a filter for img
		z.GenerateFilter()

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
	// Open image file
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

	filtMetadata := processFilterHeader(mmap, length)

	length = readFilter(mmap, filtMetadata)

	fileMetadata := processFileHeader(mmap, length)

	readFiles(mmap, fileMetadata, detail)

	return nil
}

func readFilter(mmap []byte, filtMetadata filter.FilterMetadata) (int) {
	// Find location of filter struct
	filtLoc := filtMetadata.FilterLoc
	filtSize := filtMetadata.FilterStructSize

	start := uint64(filtLoc)
	end := start + uint64(filtSize)

	// Decode filter
	gob.Register(filter.BloomFilter{})

	var bf filter.BloomFilter

	data := mmap[start:end]

	by, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil { fmt.Println(`failed base64 Decode`, err); }
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&bf)
	if err != nil { fmt.Println(`failed gob Decode`, err); }

	// Print
	fmt.Println("filter data decoded:", bf)

	return int(start) // Where next offset is
}

func readFiles(mmap []byte, metadata []manager.FileMetadata, detail bool) {
	level := 0
	space := 2

	// Print the structure (and data) of the image file
	for _, v := range metadata {
		for i := 0; i < space * level; i++ {
			fmt.Printf(" ")
		}
		if v.Begin == -1 {
			if v.Type == manager.Directory {
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
			fmt.Printf("[regular file] %v mode:%o (data: %v)\n", v.Name, v.Mode, fileString)
		}
	}

}

func processFilterHeader(mmap []byte, length int) (filter.FilterMetadata) {
	fmt.Println("Processing Filter Header")

	n := getHeaderLocation(mmap, length)

	// Decode the metadata in the header
	header := mmap[int(n) : length - 10]
	fmt.Println("metadata data:", header)

	gob.Register(filter.FilterMetadata{})

	var metadata filter.FilterMetadata

	by, err := base64.StdEncoding.DecodeString(string(header))
	if err != nil { fmt.Println(`failed base64 Decode`, err); }
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&metadata)
	if err != nil { fmt.Println(`failed gob Decode`, err); }

	fmt.Println("metadata data decoded:", metadata)
	return metadata
}

func processFileHeader(mmap []byte, length int) ([]manager.FileMetadata) {
	fmt.Println("Processing file header. Length=" + strconv.Itoa(length))

	n := getHeaderLocation(mmap, length)

	// Decode the metadata in the header
	var metadata []manager.FileMetadata
	header := mmap[int(n) : length - 10]
	fmt.Println("metadata data:", header)

	gob.Register(manager.FileMetadata{})
	gob.Register([]manager.FileMetadata{})

	by, err := base64.StdEncoding.DecodeString(string(header))
	if err != nil { fmt.Println(`failed base64 Decode`, err); }
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&metadata)
	if err != nil { fmt.Println(`failed gob Decode`, err); }

	fmt.Println("metadata data decoded:", metadata)
	return metadata

}

func getHeaderLocation(mmap []byte, length int) (int64) {
	// Filter header location is specifed by int64 at last 10 bits (bytes?)
	headerLoc := mmap[length - 10 : length]
	fmt.Println("header data:", headerLoc)

	// Setup reader for header data
	headerReader := bytes.NewReader(headerLoc)
	n, err := binary.ReadVarint(headerReader)
	if err != nil {
		log.Fatalf("can't read header location, err: %v", err)
	}
	fmt.Printf("headerLoc: %v bytes\n", n)

	return n
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
