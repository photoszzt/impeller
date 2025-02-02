//go:build stats
// +build stats

package stats

import (
	"fmt"
	"os"

	"golang.org/x/exp/slices"
)

func (c *ConcurrentStatsCollector[E]) AddSample(sample E) {
	c.mu.Lock()
	c.StatsCollector.AddSample(sample)
	c.mu.Unlock()
}

func (c *StatsCollector[E]) AddSample(sample E) {
	c.data = append(c.data, sample)
	rt_check := c.report_timer.Check()
	if uint32(len(c.data)) >= c.min_report_samples && rt_check {
		slices.Sort(c.data)
		p50 := POf(c.data, 0.5)
		p90 := POf(c.data, 0.9)
		p99 := POf(c.data, 0.99)
		duration := c.report_timer.Mark()
		fmt.Fprintf(os.Stderr, "%s stats (%d samples): dur=%v, p50=%v, p90=%v, p99=%v\n",
			c.tag, len(c.data), duration, p50, p90, p99)
		c.data = make([]E, 0, c.min_report_samples)
	}
}

func (c *ConcurrentPrintLogStatsCollector[E]) AddSample(sample E) {
	c.mu.Lock()
	c.PrintLogStatsCollector.AddSample(sample)
	c.mu.Unlock()
}
