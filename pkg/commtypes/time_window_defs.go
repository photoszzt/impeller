package commtypes

import (
	"sharedlog-stream/pkg/utils"
	"time"
)

// The fixed-size time-based window specifications used for aggregations.
//
// The semantics of time-based aggregation windows are: Every T1 (advance) milliseconds, compute the aggregate total for
// T2 (size) milliseconds.
//
//   - If {@code advance < size} a hopping windows is defined:
//     it discretize a stream into overlapping windows, which implies that a record maybe contained in one and or
//     more "adjacent" windows.
//
//   - If {@code advance == size} a tumbling window is defined:
//     it discretize a stream into non-overlapping windows, which implies that a record is only ever contained in
//     one and only one tumbling window.
//
// Thus, the specified {@link TimeWindow}s are aligned to the epoch.
// Aligned to the epoch means, that the first window starts at timestamp zero.
// For example, hopping windows with size of 5000ms and advance of 3000ms, have window boundaries
// [0;5000),[3000;8000),... and not [1000;6000),[4000;9000),... or even something "random" like [1452;6452),[4452;9452),...
type TimeWindows struct {
	// size of the windows in ms
	SizeMs int64
	// size of the window's advance interval in ms, i.e., by how much a window moves forward relative to
	// the previous one.
	AdvanceMs int64
	graceMs   int64
}

var _ = EnumerableWindowDefinition(&TimeWindows{})

// Return a window definition with the given window size, and with the advance interval being equal to the window
// size.
// The time interval represented by the N-th window is: {@code [N * size, N * size + size)}.
//
// This provides the semantics of tumbling windows, which are fixed-sized, gap-less, non-overlapping windows.
// Tumbling windows are a special case of hopping windows with {@code advance == size}.
//
// @param size The size of the window
// @return a new window definition with default maintain duration of 1 day
// @return error if the specified window size is zero or negative
func NewTimeWindowsNoGrace(size time.Duration) (*TimeWindows, error) {
	sizeMs := size.Milliseconds()
	if sizeMs <= 0 {
		return nil, DurationLeqZero
	}
	return &TimeWindows{
		SizeMs:    sizeMs,
		AdvanceMs: sizeMs,
		graceMs:   0,
	}, nil
}

func NewTimeWindowsWithGrace(size time.Duration, afterWindowEnd time.Duration) (*TimeWindows, error) {
	sizeMs := size.Milliseconds()
	if sizeMs <= 0 {
		return nil, DurationLeqZero
	}
	afterWindowEndMs := afterWindowEnd.Milliseconds()
	if afterWindowEndMs <= 0 {
		return nil, DurationLeqZero
	}
	return &TimeWindows{
		SizeMs:    sizeMs,
		AdvanceMs: sizeMs,
		graceMs:   afterWindowEndMs,
	}, nil
}

// Set the advance ("hop") the window by the given interval, which
// specifies by how much a window moves forward relative to the previous one.
// The time interval represented by the N-th window is: {@code [N * advance, N * advance + size)}.
//
// This provides the semantics of hopping windows, which are fixed-sized, overlapping windows.
//
// @param advance The advance interval ("hop") of the window, with the requirement that {@code 0 < advance.toMillis() <= sizeMs}.
// @return error if the advance interval is negative, zero, or larger than the window size
func (w *TimeWindows) AdvanceBy(advance time.Duration) (*TimeWindows, error) {
	advanceMs := advance.Milliseconds()
	if advanceMs <= 0 {
		return nil, WindowAdvanceSmallerThanZero
	}
	if advanceMs > int64(w.SizeMs) {
		return nil, WindowAdvanceLargerThanSize
	}
	return &TimeWindows{
		SizeMs:    w.SizeMs,
		AdvanceMs: advanceMs,
		graceMs:   w.graceMs,
	}, nil
}

func (w *TimeWindows) WindowsFor(timestamp int64) (map[int64]Window, []int64, error) {
	windowStart := utils.MaxInt64(0, timestamp-w.SizeMs+w.AdvanceMs) / w.AdvanceMs * w.AdvanceMs
	windows := make(map[int64]Window)
	keys := make([]int64, 0)
	for windowStart <= timestamp {
		window, err := NewTimeWindow(windowStart, windowStart+w.SizeMs)
		if err != nil {
			return nil, nil, err
		}
		windows[windowStart] = window
		keys = append(keys, windowStart)
		windowStart += w.AdvanceMs
	}
	return windows, keys, nil
}

func (w *TimeWindows) MaxSize() int64 {
	return w.SizeMs
}

func (w *TimeWindows) GracePeriodMs() int64 {
	return w.graceMs
}
