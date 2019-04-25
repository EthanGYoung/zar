package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"encoding/binary"
	"encoding/gob"
	//"io"
	"io/ioutil"
	"log"
	"path"
	"os"
	"syscall"
)

type fileWriter struct {
	zarw *bufio.Writer
	count int64
	f *os.File
}

type fileMetadata struct {
	Begin int64
	End int64
	Name string
}

type zarManager struct {
 pagealign bool
 writer fileWriter
 metadata []fileMetadata
}

func (w *fileWriter) Init(fn string) error {
	if w.count != 0 {
		err := "unkown error, writer counter is not 0 when initializing"
		log.Fatalf(err)
		return errors.New(err)
	}
	f, err := os.Create(fn)
	if err != nil {
		log.Fatalf("can't open zar output file %v, err: %v", fn, err)
		return err
	}
	w.f = f
	w.zarw = bufio.NewWriter(f)
	return nil
}
// NOTE: in the new version fileWriter.Writer return the "real" end
func (w *fileWriter) Write(data []byte, pagealign bool) (int64, error) {
	n, err := w.zarw.Write(data)
	if err != nil {
		return int64(n), err
	}

	n2 := 0
	if pagealign {
		const align = 4096
		pad := (align - n % align) % 4096
		fmt.Printf("current write size: %v, padding size: %v\n", n, pad)
		if pad > 0 {
			s := make([]byte, pad)
			n2, err = w.zarw.Write(s)
		}
	}

	//fmt.Printf("Write data %v to file, old count: %v, length: %v\n", data, w.count, n)
	realEnd := w.count + int64(n)
	w.count += int64(n + n2)

	return realEnd, err
}

func (w *fileWriter) WriteInt64(v int64) (int64, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, v)
	n, err := w.Write(buf, false)
	w.count += int64(n)
	return n, err
}

func (w *fileWriter) Close() error {
	fmt.Println("Written Bytes: ", w.count)
	w.zarw.Flush()
	return w.f.Close()
}

func (z *zarManager) IncludeFile(fn string, basedir string) (int64, error) {
	content, err := ioutil.ReadFile(path.Join(basedir, fn))
	if err != nil {
		log.Fatalf("can't include file %v, err: %v", fn, err)
		return 0, nil
	}
	oldCounter := z.writer.count
	real_end, err := z.writer.Write(content, z.pagealign)
	if err != nil {
			log.Fatalf("can't write to file")
			return 0, err
	}

	h := &fileMetadata{
			Begin : oldCounter,
			End	  : real_end,
			Name  : fn,
	}
	z.metadata = append(z.metadata, *h)
	return real_end, err
}

func (z *zarManager) IncludeFolderBegin(name string) {
	h := &fileMetadata{
			Begin : -1,
			End	  : -1,
			Name  : name,
	}
	z.metadata = append(z.metadata, *h)
}

func (z *zarManager) IncludeFolderEnd() {
	h := &fileMetadata{
			Begin : -1,
			End	  : -1,
			Name  : "..",
	}
	z.metadata = append(z.metadata, *h)
}

func (z *zarManager) WriteHeader() error {
	headerLoc := z.writer.count
	mEnc := gob.NewEncoder(z.writer.zarw)
	fmt.Println("current metadata:", z.metadata)
	//**test**
	var test bytes.Buffer
	mEncTest := gob.NewEncoder(&test)
	mEncTest.Encode(z.metadata)
	fmt.Println("test gob encode result:", test)
	//**test**
	mEnc.Encode(z.metadata)
	fmt.Printf("header location: %v bytes\n", headerLoc)
	z.writer.WriteInt64(int64(headerLoc))
	if err := z.writer.Close(); err != nil {
		log.Fatalf("can't close zar file: %v", err)
	}
	return nil
}

func writeImage(dir string, output string, pagealign bool) {
	z := &zarManager{pagealign:pagealign}
	z.writer.Init(output)
	walkDir(dir, dir, z, true)
	z.WriteHeader()
}

func walkDir(dir string, foldername string, z *zarManager, root bool) {
	if !root {
		fmt.Printf("including folder: %v, name: %v\n", dir, foldername)
		z.IncludeFolderBegin(foldername)
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("walk dir unknown err when processing dir %v", dir)
	}
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() {
			fmt.Printf("including file: %v\n", name)
			z.IncludeFile(name, dir)
		} else {
			walkDir(path.Join(dir, name), name, z, false)
		}
	}
	if !root {
		z.IncludeFolderEnd()
	}
}

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
	mmap, err := syscall.Mmap(int(f.Fd()), 0, length, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		log.Fatalf("can't mmap the image file, err: %v", err)
	}
	if detail {
		fmt.Println("MMAP data:", mmap)
	}
	headerLoc := mmap[length - 10 : length]
	fmt.Println("header data:", headerLoc)
	headerReader := bytes.NewReader(headerLoc)
	n, err := binary.ReadVarint(headerReader)
	if err != nil {
		log.Fatalf("can't read header location, err: %v", err)
	}
	fmt.Printf("headerLoc: %v bytes\n", n)

	header := mmap[int(n) : length - 10]
	fmt.Println("metadata data:", header)
	metadataReader := bytes.NewReader(header)
	var metadata []fileMetadata
	dec := gob.NewDecoder(metadataReader)
	errDec := dec.Decode(&metadata)
	if errDec != nil {
		  log.Fatalf("can't decode metadata data, err: %v", errDec)
			return err
	}
	fmt.Println(metadata)
	for _, v := range metadata {
		if v.Begin == -1 {
			fmt.Printf("enter folder: %v\n", v.Name)
		} else {
			var fileString string
			if detail {
				fileBytes := mmap[v.Begin : v.End]
				fileString = string(fileBytes)
			} else {
				fileString = "ignored"
			}
			fmt.Printf("file: %v, data: %v\n", v.Name, fileString)
		}
	}
	return nil
}

func main() {

	fmt.Println("zar image generator version 1")
	dirPtr := flag.String("dir", "./", "select the dir to generate image")
	imgPtr := flag.String("img", "test.img", "select the image to read")
	outputPtr := flag.String("o", "test.img", "output img name")
	writeModePtr := flag.Bool("w", false, "generate image mode")
	readModePtr := flag.Bool("r", false, "read image mode")
	pageAlignPtr := flag.Bool("pagealign", false, "align the page")
	detailModePtr := flag.Bool("detail", false, "show original context when read")
	flag.Parse()

	if *writeModePtr {
		fmt.Printf("dir selected: %v\n", *dirPtr)
		writeImage(*dirPtr, *outputPtr, *pageAlignPtr)
	}

	if (*readModePtr) {
		fmt.Printf("img selected: %v\n", *imgPtr)
		readImage(*imgPtr, *detailModePtr)
	}
}
