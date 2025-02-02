package commtypes

import (
	"testing"
	"time"
)

var (
	twt_start = 50
	twt_end   = 100
)

func checkWindowOverlap(curWindow *TimeWindow, start, end int64, expected bool, t *testing.T) {
	w1, err := NewTimeWindow(start, end)
	if err != nil {
		t.Fatal(err.Error())
	}
	o, err := curWindow.Overlap(w1)
	if err != nil {
		t.Fatal(err.Error())
	}
	if o != expected {
		t.Fatalf("expected overlap: %v, got: %v", expected, o)
	}
}

func TestShouldNotOverlapIfOtherWindowIsBeforeThisWindow(t *testing.T) {
	window, err := NewTimeWindow(int64(twt_start), int64(twt_end))
	if err != nil {
		t.Fatal(err.Error())
	}
	checkWindowOverlap(window, 0, 25, false, t)
	checkWindowOverlap(window, 0, int64(twt_start)-1, false, t)
	checkWindowOverlap(window, 0, int64(twt_start), false, t)
}

func TestShouldOverlapIfOtherWindowEndIsWithinThisWindow(t *testing.T) {
	/*
	 * This:        [-------)
	 * Other: [---------)
	 */
	window, err := NewTimeWindow(int64(twt_start), int64(twt_end))
	if err != nil {
		t.Fatal(err.Error())
	}
	checkWindowOverlap(window, 0, int64(twt_start)+1, true, t)
	checkWindowOverlap(window, 0, 75, true, t)
	checkWindowOverlap(window, 0, int64(twt_end)-1, true, t)

	checkWindowOverlap(window, int64(twt_start)-1, int64(twt_start)+1, true, t)
	checkWindowOverlap(window, int64(twt_start)-1, 75, true, t)
	checkWindowOverlap(window, int64(twt_start)-1, int64(twt_end)-1, true, t)
}

func TestShouldOverlapIfOtherWindowContainsThisWindow(t *testing.T) {
	/*
	 * This:        [-------)
	 * Other: [------------------)
	 */
	window, err := NewTimeWindow(int64(twt_start), int64(twt_end))
	if err != nil {
		t.Fatal(err.Error())
	}
	checkWindowOverlap(window, 0, int64(twt_end), true, t)
	checkWindowOverlap(window, 0, int64(twt_end)+1, true, t)
	checkWindowOverlap(window, 0, 150, true, t)

	checkWindowOverlap(window, int64(twt_start)-1, int64(twt_end), true, t)
	checkWindowOverlap(window, int64(twt_start)-1, int64(twt_end)+1, true, t)
	checkWindowOverlap(window, int64(twt_start)-1, 150, true, t)

	checkWindowOverlap(window, int64(twt_start), int64(twt_end), true, t)
	checkWindowOverlap(window, int64(twt_start), int64(twt_end)+1, true, t)

	checkWindowOverlap(window, int64(twt_start), 150, true, t)
}

func TestShouldOverlapIfOtherWindowIsWithinThisWindow(t *testing.T) {
	/*
	 * This:        [-------)
	 * Other:         [---)
	 */
	window, err := NewTimeWindow(int64(twt_start), int64(twt_end))
	if err != nil {
		t.Fatal(err.Error())
	}
	checkWindowOverlap(window, int64(twt_start), 75, true, t)
	checkWindowOverlap(window, int64(twt_start), int64(twt_end), true, t)
	checkWindowOverlap(window, 75, int64(twt_end), true, t)
}

func TestShouldOverlapIfOtherWindowStartIsWithinThisWindow(t *testing.T) {
	/*
	 * This:        [-------)
	 * Other:           [-------)
	 */
	window, err := NewTimeWindow(int64(twt_start), int64(twt_end))
	if err != nil {
		t.Fatal(err.Error())
	}
	checkWindowOverlap(window, int64(twt_start), int64(twt_end)+1, true, t)
	checkWindowOverlap(window, int64(twt_start), 150, true, t)
	checkWindowOverlap(window, 75, int64(twt_end)+1, true, t)
	checkWindowOverlap(window, 75, 150, true, t)
}

func TestShouldNotOverlapIsOtherWindowIsAfterThisWindow(t *testing.T) {
	/*
	 * This:        [-------)
	 * Other:               [------)
	 */
	window, err := NewTimeWindow(int64(twt_start), int64(twt_end))
	if err != nil {
		t.Fatal(err.Error())
	}
	checkWindowOverlap(window, int64(twt_end), int64(twt_end)+1, false, t)
	checkWindowOverlap(window, int64(twt_end), 150, false, t)
	checkWindowOverlap(window, int64(twt_end)+1, 150, false, t)
	checkWindowOverlap(window, 125, 150, false, t)
}

func TestShouldReturnMatchedWindowsOrderedByTimestamp(t *testing.T) {
	tws, err := NewTimeWindowsNoGrace(time.Duration(12) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	tws, err = tws.AdvanceBy(time.Duration(5) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	_, keys, err := tws.WindowsFor(21)
	if err != nil {
		t.Fatal(err.Error())
	}
	if keys[0] != 10 {
		t.Fatal("should be 10")
	}
	if keys[1] != 15 {
		t.Fatal("should be 15")
	}
	if keys[2] != 20 {
		t.Fatal("should be 20")
	}
}
