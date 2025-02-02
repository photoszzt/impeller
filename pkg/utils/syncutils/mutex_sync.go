//go:build !deadlock
// +build !deadlock

package syncutils

import "sync"

type Mutex struct {
	sync.Mutex
}

type RWMutex struct {
	sync.RWMutex
}
