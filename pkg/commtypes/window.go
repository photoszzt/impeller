//go:generate msgp
//msgp:ignore Window
package commtypes

import (
	"fmt"
	"time"

	"golang.org/x/xerrors"
)

var (
	WindowEndNotLargerStart = xerrors.New("Window endMs should be greater than window startMs")
)

type Window interface {
	// returns window start in unix timestamp (ms)
	Start() int64
	// returns window end in unix timestamp (ms)
	End() int64
	// returns window start time
	StartTime() *time.Time
	// returns window end time
	EndTime() *time.Time
	// check if the given window overlaps with this window
	Overlap(other Window) (bool, error)
}

type BaseWindow struct {
	startT  *time.Time `json:"-" msg:"-"`
	endT    *time.Time `json:"-" msg:"-"`
	StartTs int64      `json:"startTs" msg:"startTs"`
	EndTs   int64      `json:"endTs" msg:"endTs"`
}

var _ = fmt.Stringer(BaseWindow{})

func (bw BaseWindow) String() string {
	return fmt.Sprintf("BaseWindow: {StartTs: %d, EndTs: %d}", bw.StartTs, bw.EndTs)
}

func NewBaseWindow(startTs int64, endTs int64) BaseWindow {
	startSecPart := startTs / 1000
	startNsPart := (startTs - startSecPart*1000) * 1000000
	startTime := time.Unix(int64(startSecPart), int64(startNsPart))
	endSecPart := endTs / 1000
	endNsPart := (endTs - endSecPart*1000) * 1000000
	endTime := time.Unix(int64(endSecPart), int64(endNsPart))
	return BaseWindow{
		StartTs: startTs,
		EndTs:   endTs,
		startT:  &startTime,
		endT:    &endTime,
	}
}

func (w *BaseWindow) Start() int64 {
	return w.StartTs
}

func (w *BaseWindow) End() int64 {
	return w.EndTs
}

func (w *BaseWindow) StartTime() *time.Time {
	return w.startT
}

func (w *BaseWindow) EndTime() *time.Time {
	return w.endT
}
