package stats

import (

)

const (

)

// stats keeps track of information pertinent to creating the filter
type ImgStats struct {
	// NumFiles represents number of unique files in file (Include symbolic links)
	NumFiles uint64

	// NumDirs represents number of directories in image file
	NumDirs uint64
}

// AddFile increments NumFile in the ImgStats struct
func (s *ImgStats) AddFile() {
	s.NumFiles++
}

// AddDir increments NumDirs in the ImgStats struct
func (s *ImgStats) AddDir() {
	s.NumDirs++
}