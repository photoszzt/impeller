package stats

import (
	"sync"
	"time"
)

type ReportTimer struct {
	once     sync.Once
	lastTs   time.Time
	duration time.Duration
}

func NewReportTimer(duration time.Duration) ReportTimer {
	return ReportTimer{
		duration: duration,
		lastTs:   time.Time{},
	}
}

func (r *ReportTimer) Check() bool {
	r.once.Do(func() {
		r.lastTs = time.Now()
	})
	return time.Since(r.lastTs) >= r.duration
}

func (r *ReportTimer) Mark() time.Duration {
	duration := time.Since(r.lastTs)
	r.lastTs = time.Now()
	return duration
}
