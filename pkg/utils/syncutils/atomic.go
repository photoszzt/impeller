package syncutils

import "sync/atomic"

// AtomicBool mimics an atomic boolean.
type AtomicBool uint32

// Set atomically sets the boolean.
func (b *AtomicBool) Set(v bool) {
	s := uint32(0)
	if v {
		s = 1
	}
	atomic.StoreUint32((*uint32)(b), s)
}

// Get atomically gets the boolean.
func (b *AtomicBool) Get() bool {
	return atomic.LoadUint32((*uint32)(b)) != 0
}

// Swap atomically swaps the value.
func (b *AtomicBool) Swap(v bool) bool {
	wanted := uint32(0)
	if v {
		wanted = 1
	}
	return atomic.SwapUint32((*uint32)(b), wanted) != 0
}
