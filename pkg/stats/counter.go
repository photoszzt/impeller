package stats

import "sync/atomic"

type Counter struct {
	tag   string
	count uint64
}

func NewCounter(tag string) Counter {
	return Counter{
		tag:   tag,
		count: 0,
	}
}

func (c *Counter) GetCount() uint64 {
	return c.count
}

type AtomicCounter struct {
	tag   string
	count atomic.Uint64
}

func NewAtomicCounter(tag string) AtomicCounter {
	return AtomicCounter{
		tag: tag,
	}
}

func (c *AtomicCounter) InitCounter() {
	c.count.Store(0)
}

func (c *AtomicCounter) GetCount() uint64 {
	return c.count.Load()
}
