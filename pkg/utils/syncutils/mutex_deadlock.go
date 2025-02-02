//go:build deadlock
// +build deadlock

package syncutils

import "github.com/sasha-s/go-deadlock"

type Mutex struct {
	deadlock.Mutex
}

type RWMutex struct {
	deadlock.RWMutex
}
