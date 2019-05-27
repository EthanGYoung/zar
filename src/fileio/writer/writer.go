// Package writer implements a library for writing to the image file
package writer

import(
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"encoding/binary"
)

const(
        pageBoundary = 4096     // The boundary that needs to be upheld for page alignment
 )

// FileWriter struct writes to a file
type FileWriter struct {
        // w is used as the writer to the file f
        W *bufio.Writer // TODO: Rename

        // Count is the cumulative bytes written to the file f
        Count int64

        // f is the file object that the writer will write to
        F *os.File // TODO: Rename
}

// Initializes a writer by creating the image file and attaching a writer to it\
//
// Parameter (fn): Name of image file
func (w *FileWriter) Init(fn string) error {
        if w.Count != 0 {
                err := "unknown error, writer Counter is not 0 when initializing"
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
        w.F = f
        w.W = bufio.NewWriter(f)
        return nil
}

// NOTE: in the new version FileWriter.Writer return the "real" end
// Write writes the data to the zar file. The caller can specify whether or not
// to keep the file page aligned
//
// parameter (data)     : the data to be written
// parameter (pageAlign): whether to page align the data
func (w *FileWriter) Write(data []byte, pageAlign bool) (int64, error) {
        // Writes to FileWriter
        n, err := w.W.Write(data)
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
                        n2, err = w.W.Write(s)
                }
        }

        // Updates offsets
        realEnd := w.Count + int64(n)
        w.Count += int64(n + n2)

        return realEnd, err
}

// WriteInt64 writes a int64 to the FileWriter
//
// parameter (v): the value to be written
func (w *FileWriter) WriteInt64(v int64) (int64, error) {
        buf := make([]byte, binary.MaxVarintLen64)
        binary.PutVarint(buf, v)
        n, err := w.Write(buf, false)
        return n, err
}

// Close closes the filewriter by flushing any buffer
func (w *FileWriter) Close() error {
        fmt.Println("Written Bytes: ", w.Count, "+ metadata size")
        w.W.Flush()
        return w.F.Close()
}
