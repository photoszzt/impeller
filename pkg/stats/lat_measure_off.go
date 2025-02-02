//go:build !stats
// +build !stats

package stats

import "time"

var (
	EMPTY_TIME = time.Time{}
)

const (
	ZERO_DURATION = time.Duration(0)
)

func TimerBegin() time.Time { return EMPTY_TIME }

func Elapsed(start time.Time) time.Duration { return ZERO_DURATION }
