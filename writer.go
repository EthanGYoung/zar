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

func (w *fileWriter) Write(data []byte) (int, error) {
	n, err := w.zarw.Write(data)
	//fmt.Printf("Write data %v to file, old count: %v, length: %v\n", data, w.count, n)
	w.count += int64(n)

	return n, err
}

func (w *fileWriter) WriteInt64(v int64) (int, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, v)
	n, err := w.Write(buf)
	w.count += int64(n)
	return n, err
}

func (w *fileWriter) Close() error {
	fmt.Println("Written Bytes: ", w.count)
	w.zarw.Flush()
	return w.f.Close()
}

func (z *zarManager) IncludeFile(fn string, basedir string) (int, error) {
	content, err := ioutil.ReadFile(path.Join(basedir, fn))
	if err != nil {
		log.Fatalf("can't include file %v, err: %v", fn, err)
		return 0, nil
	}
	oldCounter := z.writer.count
	n, err := z.writer.Write(content)
	if err != nil {
			log.Fatalf("can't write to file")
			return 0, err
	}
	h := &fileMetadata{
			Begin : oldCounter,
			End	  : z.writer.count,
			Name  : fn,
	}
	z.metadata = append(z.metadata, *h)
	return n, err
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

func walkDir(dir string, output string) error {
	z := &zarManager{}
	z.writer.Init(output)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("walk dir unknown err when processing dir %v", dir)
		return err
	}
	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			fmt.Printf("including file: %v\n", name)
			z.IncludeFile(name, dir)
		}
	}
	z.WriteHeader()
	return nil
}

func readImage(img string) error {
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
	fmt.Println("MMAP data:", mmap)
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
		fileBytes := mmap[v.Begin : v.End]
		fileString := string(fileBytes)
		fmt.Printf("file: %v, data: %v\n", v.Name, fileString)
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
	flag.Parse()

	if *writeModePtr {
		fmt.Printf("dir selected: %v\n", *dirPtr)
		walkDir(*dirPtr, *outputPtr)
	}

	if (*readModePtr) {
		fmt.Printf("img selected: %v\n", *imgPtr)
		readImage(*imgPtr)
	}
}