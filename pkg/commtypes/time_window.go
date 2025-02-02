//go:generate msgp
//msgp:ignore TimeWindowJSONSerde TimeWindowMsgpSerde
package commtypes

import (
	"math"

	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"
)

type TimeWindow struct {
	BaseWindow
}

func (t TimeWindow) Equal(b TimeWindow) bool {
	return t.StartTs == b.StartTs && t.EndTs == b.EndTs
}

var (
	_              = Window(&TimeWindow{})
	notATimeWindow = xerrors.New("The window is not a TimeWindow")
)

func NewTimeWindow(startMs int64, endMs int64) (*TimeWindow, error) {
	if startMs == endMs {
		return nil, WindowEndNotLargerStart
	}
	return &TimeWindow{
		NewBaseWindow(startMs, endMs),
	}, nil
}

func (w *TimeWindow) Overlap(other Window) (bool, error) {
	_, ok := other.(*TimeWindow)
	if !ok {
		return false, notATimeWindow
	}
	return w.Start() < other.End() && other.Start() < w.End(), nil
}

func TimeWindowForSize(startMs int64, windowSize int64) (*TimeWindow, error) {
	endMs := startMs + windowSize
	if endMs < 0 {
		log.Warn().Msg("window end time was truncated to int64 max")
		endMs = math.MaxInt64
	}
	return NewTimeWindow(startMs, endMs)
}
