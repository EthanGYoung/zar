package img_test

import (
	"filter"
	"manager"
	"testing"
	"stats"
	"strconv"
)

// TODO: Move this to the manager tests

// TestFilterConstruction tests that, given the metadata, the filter can be accurately constructed
func TestFilterConstruction(t *testing.T) {
	var DummyMetadata = []manager.FileMetadata {
		// Structure
		// root
		//	apples.txt
		//	Groceries
		//	..
		//	OtherApples.txt
		// Create dummy metadata (Dont care about Begin, End, Link, ModTime
		manager.FileMetadata{
			Name:"root",
			Type:manager.Directory,
		},
		manager.FileMetadata{
			Name:"Apples.txt",
			Type:manager.RegularFile,
		},
		manager.FileMetadata{
			Name:"Groceries",
			Type:manager.Directory,
		},
		manager.FileMetadata{
			Name:"..",
			Type:manager.Directory,
		},
		manager.FileMetadata{
			Name:"OtherApples.txt",
			Type:manager.Symlink,
		},
	}


	// Copying beginning of writeImage in main.go
	var z *manager.ZarManager

	// Initializes all fields to 0
	var stats = &stats.ImgStats{
		NumFiles:2,
		NumSymLinks:1,
		NumDirs:2,
	}
	var filt = &filter.BloomFilter{NumElem:5} // Default to BloomFilter

	z = &manager.ZarManager{
		Statistics	: stats,
		Filter		: filt,
		Metadata	: DummyMetadata,
	}

	// Create the bloom filter
	z.GenerateFilter()

	path:="root/Apples.txt"
	exp:=true
	if z.Filter.TestElement([]byte(path)) {
		t.Errorf("z.Filter.TestElement([]byte('" + path + "') return " + strconv.FormatBool(!exp) + "  when it should have returned " + strconv.FormatBool(exp))
	}

	path="root/OtherApples.txt"
	exp=true
	if z.Filter.TestElement([]byte(path)) {
		t.Errorf("z.Filter.TestElement([]byte('" + path + "') return " + strconv.FormatBool(!exp) + "  when it should have returned " + strconv.FormatBool(exp))
	}

	path="root/Oranges.txt"
	exp=false
	if z.Filter.TestElement([]byte(path)) {
		t.Errorf("z.Filter.TestElement([]byte('" + path + "') return " + strconv.FormatBool(!exp) + "  when it should have returned " + strconv.FormatBool(exp))
	}
}

// TODO: TestStats
