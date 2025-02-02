//go:build stats
// +build stats

package stats

import (
	"fmt"
	"os"
)

func (c *ThroughputCounter) Tick(count uint64) {
	c.count += count
	rt_check := c.report_timer.Check()
	if c.count > c.last_count && rt_check {
		duration := c.report_timer.Mark()
		tp := float64(c.count-c.last_count) / duration.Seconds()
		fmt.Fprintf(os.Stderr, "%s counter: dur=%v, value=%v, rate=%v per second\n", c.tag, duration, c.count, tp)
		c.last_count = c.count
	}
}

func (c *ConcurrentThroughputCounter) Tick(count uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count += count
	rt_check := c.report_timer.Check()
	if c.count > c.last_count && rt_check {
		duration := c.report_timer.Mark()
		tp := float64(c.count-c.last_count) / duration.Seconds()
		c.last_count = c.count
		fmt.Fprintf(os.Stderr, "%s counter: dur=%v, value=%v, rate=%v per second\n", c.tag, duration, c.count, tp)
	}
}
