//go:build !stats
// +build !stats

package stats

func (c *ThroughputCounter) Tick(count uint64)           {}
func (c *ConcurrentThroughputCounter) Tick(count uint64) {}
