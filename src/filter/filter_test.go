package filter_test

import (
	"filter"
	"testing"
)


func TestBF(t *testing.T) {
	// Create BF
	bf := &filter.BloomFilter{
		FPProb:0.0001,
		NumElem:5,
	}

	bf.Initialize()

	// Confirm follow equations correctly
	if bf.FilterSize != 96 {
		t.Errorf("bf.FilterSize != 96, got %d", bf.FilterSize)
	}
	if bf.NumHashes != 13 {
		t.Errorf("bf.NumHashes != 13, got %d", bf.NumHashes)
	}

	// Test by adding elements
	bf.AddElement([]byte("hello"))
	bf.AddElement([]byte("world"))
	bf.AddElement([]byte("sir"))
	bf.AddElement([]byte("madam"))
	bf.AddElement([]byte("io"))

	if !bf.TestElement([]byte("hello")) {
		t.Errorf("bf.TestElement([]byte('hello') return false when it should have returned true")
	}

	if !bf.TestElement([]byte("world")) {
		t.Errorf("bf.TestElement([]byte('world') return false when it should have returned true")
	}

	if bf.TestElement([]byte("hi")) {
		t.Errorf("bf.TestElement([]byte('hi') return true when it should have returned false")
	}
}
