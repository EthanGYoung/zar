// Package filter implements a library for constructing a filter for img layer
package filter


import (
)

type Filter interface {
	// Initialize creates a filter with the specified initial conditions
	//
	// 
	Initialize()

	// AddElement adds an element to the filter by hashing the element into the filter
	//
	// elem:	Represents an element to add to the bloom filter 
	AddElement(elem []byte)

	// RemoveElement removes an element from the filter
	//
	//
	RemoveElement()

	// TestElement checks if the specific element exists in data structure
	//
	// Return: False if not present in filter, true if possibly present
	TestElement(elem []byte) bool

}

// FilterMetadata is a struct that depicts what will get written to the image metadata for a filter
type FilterMetadata struct {
	// Active indicates whether or not a bloom filter is enforced for this layer
	Active bool

	// Name represents the name of the filter used for this layer. Must be same as in ContainerFS
	Name string

	// FilterStructSize is the numbber of bytes (bits?) that the BloomFilter struct takes up
	FilterLoc int64

	// FilterStructSize is the size in bytes of the encoded structure
	FilterStructSize int64
}
