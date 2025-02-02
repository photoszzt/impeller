package stats

import (
	"sharedlog-stream/pkg/utils/syncutils"
	"time"
)

type ThroughputCounter struct {
	tag          string
	count        uint64
	last_count   uint64
	report_timer ReportTimer
}

func NewThroughputCounter(tag string, duration time.Duration) ThroughputCounter {
	return ThroughputCounter{
		tag:          tag,
		count:        0,
		last_count:   0,
		report_timer: NewReportTimer(duration),
	}
}

func (c *ThroughputCounter) GetCount() uint64 {
	return c.count
}

type ConcurrentThroughputCounter struct {
	mu           syncutils.Mutex
	tag          string
	count        uint64
	last_count   uint64
	report_timer ReportTimer
}

func NewConcurrentThroughputCounter(tag string, duration time.Duration) *ConcurrentThroughputCounter {
	return &ConcurrentThroughputCounter{
		tag:          tag,
		count:        0,
		last_count:   0,
		report_timer: NewReportTimer(duration),
	}
}

// this method is only called at the end where nothing is updating it
func (c *ConcurrentThroughputCounter) GetCount() uint64 {
	return c.count
}
