//go:build !printstats
// +build !printstats

package stats

func (c *PrintLogStatsCollector[E]) AddSample(sample E) {}
