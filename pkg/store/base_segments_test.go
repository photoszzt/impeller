package store

import (
	"fmt"
	"testing"
)

const (
	bstest_segment_interval = 100
	bstest_retention_period = 4 * bstest_segment_interval
)

func TestShouldGetSegmentIdsFromTimestamp(t *testing.T) {
	segments := NewBaseSegments("test", bstest_retention_period, bstest_segment_interval)
	if ret := segments.SegmentId(0); ret != 0 {
		t.Fatalf("expected %d, got %d", 0, ret)
	}
	if ret := segments.SegmentId(bstest_segment_interval); ret != 1 {
		t.Fatalf("expected %d, got %d", 1, ret)
	}
	if ret := segments.SegmentId(bstest_segment_interval * 2); ret != 2 {
		t.Fatalf("expected %d, got %d", 2, ret)
	}

	if ret := segments.SegmentId(bstest_segment_interval * 3); ret != 3 {
		t.Fatalf("expected %d, got %d", 3, ret)
	}
}

func TestShouldBaseSegmentIntervalOnRetentionAndNumSegments(t *testing.T) {
	segments := NewBaseSegments("test", bstest_segment_interval*8, bstest_segment_interval*2)
	if ret := segments.SegmentId(0); ret != 0 {
		t.Fatalf("expected %d, got %d", 0, ret)
	}
	if ret := segments.SegmentId(bstest_segment_interval); ret != 0 {
		t.Fatalf("expected %d, got %d", 0, ret)
	}
	if ret := segments.SegmentId(bstest_segment_interval * 2); ret != 1 {
		t.Fatalf("expected %d, got %d", 1, ret)
	}
}

func TestShouldGetSegmentNameFromId(t *testing.T) {
	segments := NewBaseSegments("test", bstest_retention_period, bstest_segment_interval)
	expected := "test.0"
	if ret := segments.SegmentName(0); ret != expected {
		t.Fatalf("expected %s, got %s", expected, ret)
	}
	expected = fmt.Sprintf("test.%d", bstest_segment_interval)
	if ret := segments.SegmentName(1); ret != expected {
		t.Fatalf("expected %s, got %s", expected, ret)
	}
	expected = fmt.Sprintf("test.%d", 2*bstest_segment_interval)
	if ret := segments.SegmentName(2); ret != expected {
		t.Fatalf("expected %s, got %s", expected, ret)
	}
}
