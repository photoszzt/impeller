//go:build !stats
// +build !stats

package stats

func (c *Counter) Tick(count uint32) {}

func (c *Counter) Report() {}

func (c *AtomicCounter) Tick(count uint32) {}
func (c *AtomicCounter) Report()           {}
