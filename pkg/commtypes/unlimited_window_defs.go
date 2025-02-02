package commtypes

import (
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DEFAULT_START_TIMESTAMP_MS = 0
)

var _ = EnumerableWindowDefinition(&UnlimitedWindows{})

type UnlimitedWindows struct {
	startMs int64
}

func NewUnlimitedWindows(startMs int64) *UnlimitedWindows {
	return &UnlimitedWindows{
		startMs: startMs,
	}
}

func NewDefaultUnlimitedWindows() *UnlimitedWindows {
	return &UnlimitedWindows{
		startMs: DEFAULT_START_TIMESTAMP_MS,
	}
}

func NewUnlimitedWindowsStartOn(start time.Time) *UnlimitedWindows {
	startMs := start.UnixMilli()
	if startMs < 0 {
		log.Fatal().Msg("start time (ms) cannot be negative")
	}
	return &UnlimitedWindows{
		startMs: startMs,
	}
}

func (w *UnlimitedWindows) MaxSize() int64 {
	return math.MaxInt64
}

func (w *UnlimitedWindows) GracePeriodMs() int64 {
	return 0
}

func (w *UnlimitedWindows) WindowsFor(timestamp int64) (map[int64]Window, []int64, error) {
	windows := make(map[int64]Window)
	keys := make([]int64, 0)
	if timestamp >= w.startMs {
		windows[w.startMs] = NewUnlimitedWindow(w.startMs)
		keys = append(keys, w.startMs)
	}
	return windows, keys, nil
}
