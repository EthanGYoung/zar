package filter

import (
	"github.com/spaolacci/murmur3"
	"math"
)

const(
	DEFAULT_PROB = 0.000001
)

// BloomFilter is a struct for creating a Bloom filter for an image file. A
// A Bloom filter specifies whether a specific file path is "definitily" not
// in the image file or is "maybe" in the file with a certain probability. 
// This struct implements the Filter interface.
type BloomFilter struct {
	// FPProb (False Positive Probability) is the desired probability of a false positive in the filter
	FPProb float64

	// NumHashes represents the number of hash functions for this bloom filter (k)
	NumHashes uint64

	// NumElem represents the number of elements in this filter (n)
	NumElem uint64

	// FilterSize represents the number of bits in this filter (m)
	FilterSize uint64

	// BitSet is the array of bits that implements a bloom filter
	BitSet []bool
}


// Initialize implements Filter.Initialize. Assumes b.NumElem is set to number of expected elements
// and FPProb is set
func (b *BloomFilter) Initialize() {
	// Check for error conditions
	if (b.NumElem < 1) {
		// Return error
		return
	}

	// Initialize FPProb
	if (b.FPProb == 0) { b.FPProb = DEFAULT_PROB }

	// Compute filter size and initialize bitarray
	b.FilterSize = b.calcFilterSize()
	b.BitSet = make([]bool, b.FilterSize)

	// Compute number of hashes (k)
	b.NumHashes = b.calcNumHashes()

}

// calcFilterSize calculates the optimal size of bit array given prob and elements
// Assumes FPProb and NumElem is set
// m = ceil((n*log(p)) / log(1 / pow(2, log(2))) 
func (b *BloomFilter) calcFilterSize() uint64 {
	return uint64(math.Ceil((float64(b.NumElem) * math.Log(b.FPProb)) / math.Log(1 / math.Pow(2, math.Log(2)))))
}

// calcNumHashes calculates the aptimal number of hashes given the filter size and the number of elements
// Assumes FilterSize and NumElem set
// k = round((m / n) * log(2))
func (b *BloomFilter) calcNumHashes() uint64 {
	return uint64(math.Round(float64(b.FilterSize / b.NumElem) * math.Log(2)))
}

// AddElement implements Filter.AddElement
func (b *BloomFilter) AddElement(elem []byte) {
	// Get the hashed value of the element
	h1, h2 := b.hashElement(elem)

	intHash := h1

	// Set bits in bitset to represent added element -> TODO: Does int cast affect anything?
	for i:=0; i < int(b.NumHashes); i++ {
		intHash += (b.NumHashes*h2)
		bitToSet := intHash % b.FilterSize
		b.setBit(bitToSet)
	}

}

// hashElement hashes the elem passed in based on the murmur hash function
// TODO: Unsure if Sum128 is correct
func (b *BloomFilter) hashElement(elem []byte) (uint64, uint64) {
	return murmur3.Sum128(elem)
}

// setBits will set bit at position to true
func (b *BloomFilter) setBit(position uint64) {
	b.BitSet[position] = true
}

// RemoveElement removes an element from the filter
//
//
func (b *BloomFilter) RemoveElement() {
	// No-op for bloom filter
}

// TestElement implements Filter.TestElement
func (b *BloomFilter) TestElement(elem []byte) bool {
	// TODO: Make this modular with add element
	// Get the hashed value of the element
	h1, h2 := b.hashElement(elem)

	intHash := h1

	// TODO: Look into this, may be perf issue..
	var testFilter = make([]bool, b.FilterSize)

	// Create a test bit array
	copy(testFilter, b.BitSet)

	// Set bits in bitset to represent added element
	for i:=0; i < int(b.NumHashes); i++ {
		intHash += (b.NumHashes*h2)
		bitToSet := intHash % b.FilterSize
		testFilter[bitToSet] = true
	}

	// Test if found by checking that all bits set are same as original
	return b.checkBitSetEquality(testFilter)
}

// checkBitSetEquality checks if a test bloom filter equals the current bloom filter
func (b *BloomFilter) checkBitSetEquality(test []bool) (bool) {
	if (len(test) != len(b.BitSet)) { return false }

	for i:=0; i < int(b.FilterSize); i++ {
		if (b.BitSet[i] != test[i]) { return false }
	}

	return true
}
