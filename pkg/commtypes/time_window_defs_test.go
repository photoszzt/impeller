package commtypes

import (
	"sharedlog-stream/pkg/utils"
	"testing"
	"time"
)

const (
	TEST_TWS_SIZE  = 123
	TEST_TWS_GRACE = 1024
)

func TestShouldSetWindowSize(t *testing.T) {
	tws, err := NewTimeWindowsNoGrace(time.Duration(TEST_TWS_SIZE) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	if tws.SizeMs != TEST_TWS_SIZE {
		t.Fatalf("expected %d, got %d", TEST_TWS_SIZE, tws.SizeMs)
	}
	tws, err = NewTimeWindowsWithGrace(time.Duration(TEST_TWS_SIZE)*time.Millisecond,
		time.Duration(TEST_TWS_GRACE)*time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	if tws.SizeMs != TEST_TWS_SIZE {
		t.Fatalf("expected %d, got %d", TEST_TWS_SIZE, tws.SizeMs)
	}
}

func TestShouldSetWindowAdvance(t *testing.T) {
	adv := 4
	tws, err := NewTimeWindowsNoGrace(time.Duration(TEST_TWS_GRACE) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	tws, err = tws.AdvanceBy(time.Duration(adv) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	if tws.AdvanceMs != int64(adv) {
		t.Fatalf("expected %d, got %d", adv, tws.AdvanceMs)
	}
}

func TestShouldComputeWindowsForHoppingWindows(t *testing.T) {
	tws, err := NewTimeWindowsNoGrace(time.Duration(12) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	tws, err = tws.AdvanceBy(time.Duration(5) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	matched, _, err := tws.WindowsFor(21)
	if err != nil {
		t.Fatal(err.Error())
	}
	expect_size := 12/5 + 1
	l := len(matched)
	if l != expect_size {
		utils.FatalMsg(expect_size, l, t)
	}

	expected, err := NewTimeWindow(10, 22)
	if err != nil {
		t.Fatal(err.Error())
	}
	w := matched[10]
	if expected.StartTs != w.Start() && expected.EndTs != w.End() {
		utils.FatalMsg(expected, w, t)
	}

	expected, err = NewTimeWindow(15, 27)
	if err != nil {
		t.Fatal(err.Error())
	}
	w = matched[15]
	if expected.StartTs != w.Start() && expected.EndTs != w.End() {
		utils.FatalMsg(expected, w, t)
	}

	expected, err = NewTimeWindow(20, 32)
	if err != nil {
		t.Fatal(err.Error())
	}
	w = matched[20]
	if expected.StartTs != w.Start() && expected.EndTs != w.End() {
		utils.FatalMsg(expected, w, t)
	}
}

func TestShouldComputeWindowsForBarelyOverlappingHoppingWindows(t *testing.T) {
	tws, err := NewTimeWindowsNoGrace(time.Duration(6) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	tws, err = tws.AdvanceBy(time.Duration(5) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	matched, _, err := tws.WindowsFor(7)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1, got %d\n", len(matched))
	}
	expected, err := NewTimeWindow(5, 11)
	if err != nil {
		t.Fatal(err.Error())
	}
	w := matched[5]
	if expected.StartTs != w.Start() && expected.EndTs != w.End() {
		utils.FatalMsg(expected, w, t)
	}
}

func TestShouldComputeWindowsForTumblingWindows(t *testing.T) {
	tws, err := NewTimeWindowsNoGrace(time.Duration(12) * time.Millisecond)
	if err != nil {
		t.Fatal(err.Error())
	}
	matched, _, err := tws.WindowsFor(21)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1, got %d\n", len(matched))
	}
	expected, err := NewTimeWindow(12, 24)
	if err != nil {
		t.Fatal(err.Error())
	}
	w := matched[12]
	if expected.StartTs != w.Start() && expected.EndTs != w.End() {
		utils.FatalMsg(expected, w, t)
	}
}
