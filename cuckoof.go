// Package cuckoof implements cuckoo filter
/*

This implements a (2-4)-cuckoo filter with 8-bit fingerprints.  This gives a
false positive rate of approximately 3.1%.

    http://mybiasedcoin.blogspot.nl/2014/10/cuckoo-filters.html
    https://www.cs.cmu.edu/~dga/papers/cuckoo-conext2014.pdf
    https://www.cs.cmu.edu/~binfan/papers/login_cuckoofilter.pdf

*/
package cuckoof

import (
	"math/rand"

	"github.com/dchest/siphash"
)

// CF is a cuckoo filter
type CF struct {
	t        [][4]byte
	occupied []byte
	rnd      uint64
}

// New returns a new cuckoo filter with size hash table entries.  Size must be a power of two
func New(size int) *CF {

	if size&(size-1) != 0 {
		panic("cuckoof: size must be a power of two")
	}

	return &CF{
		t:        make([][4]byte, size),
		occupied: make([]byte, size/2),
		rnd:      uint64(rand.Int63()),
	}
}

// Insert adds an element to the filter and returns if the insertion was successful.
func (cf *CF) Insert(x []byte) bool {

	h := siphash.Hash(0, 0, x)

	i1 := uint32(h) % uint32(len(cf.t))

	f := byte(h >> 32)

	i2 := (i1 ^ hashfp(f)) % uint32(len(cf.t))

	if idx, ok := cf.hasSpace(i1); ok {
		cf.setOccupied(i1, idx, f)
		return true
	}

	if idx, ok := cf.hasSpace(i2); ok {
		cf.setOccupied(i2, idx, f)
		return true
	}

	i := i1
	cf.rnd = rnd(cf.rnd)
	if cf.rnd&1 == 1 {
		i = i2
	}

	for n := 0; n < 500; n++ {
		f = cf.evict(i, f)
		i = (i ^ hashfp(f)) % uint32(len(cf.t))
		if idx, ok := cf.hasSpace(i); ok {
			cf.setOccupied(i, idx, f)
			return true
		}
	}

	return false
}

// Lookup queries the cuckoo filter for an item
func (cf *CF) Lookup(x []byte) bool {
	h := siphash.Hash(0, 0, x)

	i1 := uint32(h) % uint32(len(cf.t))

	f := byte(h >> 32)

	if cf.hasFP(i1, f) {
		return true
	}

	i2 := (i1 ^ hashfp(f)) % uint32(len(cf.t))

	return cf.hasFP(i2, f)
}

// Delete removes an item from the cuckoo filter
func (cf *CF) Delete(x []byte) bool {
	h := siphash.Hash(0, 0, x)

	i1 := uint32(h) % uint32(len(cf.t))

	f := byte(h >> 32)

	if cf.delFP(i1, f) {
		return true
	}

	i2 := (i1 ^ hashfp(f)) % uint32(len(cf.t))

	return cf.delFP(i2, f)
}

func (cf *CF) evict(row uint32, f byte) byte {
	cf.rnd = rnd(cf.rnd)

	// random bucket
	bucket := cf.rnd & 3
	e := cf.t[row][bucket]
	cf.t[row][bucket] = f

	return e
}

func (cf *CF) hasFP(row uint32, f byte) bool {
	b := cf.occupied[row/2]
	t := row & 1
	b = (b >> (uint(t) * 4)) & 0xF

	return false ||
		b&0x01 == 0x01 && cf.t[row][0] == f ||
		b&0x02 == 0x02 && cf.t[row][1] == f ||
		b&0x04 == 0x04 && cf.t[row][2] == f ||
		b&0x08 == 0x08 && cf.t[row][3] == f
}

func (cf *CF) delFP(row uint32, f byte) bool {
	b := cf.occupied[row/2]
	t := row & 1
	b = (b >> (uint(t) * 4)) & 0xF

	switch {
	case b&0x01 == 0x01 && cf.t[row][0] == f:
		cf.occupied[row/2] &^= (1 << 0) << (uint(t) * 4)
		return true
	case b&0x02 == 0x02 && cf.t[row][1] == f:
		cf.occupied[row/2] &^= (1 << 1) << (uint(t) * 4)
		return true
	case b&0x04 == 0x04 && cf.t[row][2] == f:
		cf.occupied[row/2] &^= (1 << 2) << (uint(t) * 4)
		return true
	case b&0x08 == 0x08 && cf.t[row][3] == f:
		cf.occupied[row/2] &^= (1 << 3) << (uint(t) * 4)
		return true
	}

	return false
}

func (cf *CF) setOccupied(row uint32, idx byte, f byte) {
	t := row & 1
	cf.t[row][idx] = f
	cf.occupied[row/2] |= (1 << idx) << (uint(t) * 4)
}

var freebits = [16]byte{
	0, // 0000
	1, // 0001
	0, // 0010
	2, // 0011
	0, // 0100
	1, // 0101
	0, // 0110
	3, // 0111
	0, // 1000
	1, // 1001
	0, // 1010
	2, // 1011
	0, // 1100
	1, // 1101
	0, // 1110
	0, // 1111
}

func (cf *CF) hasSpace(row uint32) (byte, bool) {
	b := cf.occupied[row/2]
	t := row & 1
	b = (b >> (uint(t) * 4)) & 0xF
	return freebits[b], b != 0xF
}

// rnd is an xorshift/multiple random number generator
func rnd(x uint64) uint64 {
	x ^= x >> 12 // a
	x ^= x << 25 // b
	x ^= x >> 27 // c
	x *= 2685821657736338717
	return x
}

// hash a fingerprint with 2 rounds of an xorshift-mult rng
func hashfp(b byte) uint32 {
	x := rnd(rnd(uint64(b)))
	return uint32(x) ^ uint32(x>>32)
}
