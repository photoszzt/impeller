//go:build printstats
// +build printstats

package stats

import (
	"fmt"
	"os"
)

func (c *PrintLogStatsCollector[E]) AddSample(sample E) {
	l_d := len(c.data)
	cap_d := cap(c.data)
	if l_d >= cap_d {
		fmt.Fprintf(os.Stderr, "%s (%d samples): data=%v\n", c.tag, l_d, c.data)
		c.data = make([]E, 0, cap_d)
	}
	c.data = append(c.data, sample)
}
