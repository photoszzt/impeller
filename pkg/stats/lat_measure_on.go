//go:build stats
// +build stats

package stats

import "time"

func TimerBegin() time.Time {
	return time.Now()
}

func Elapsed(start time.Time) time.Duration {
	return time.Since(start)
}
