package hashfuncs

import (
	"github.com/cespare/xxhash/v2"
	"golang.org/x/exp/constraints"
)

type HashSum64[K any] interface {
	HashSum64(k K) uint64
}

type IntegerHasher[K constraints.Integer] struct{}

func (h IntegerHasher[K]) HashSum64(k K) uint64 {
	return uint64(k)
}

type ByteHasher struct{}

func (h ByteHasher) HashSum64(k byte) uint64 {
	return uint64(k)
}

type StringHasher struct{}

func (sh StringHasher) HashSum64(k string) uint64 {
	return xxhash.Sum64String(k)
}

type ByteSliceHasher struct{}

func (sh ByteSliceHasher) HashSum64(k []byte) uint64 {
	return xxhash.Sum64(k)
}
