package store

import (
	"context"
	"os"
	"reflect"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/optional"
	"testing"
)

func toStrSet(data []string) map[string]struct{} {
	d := make(map[string]struct{})
	for _, s := range data {
		d[s] = struct{}{}
	}
	return d
}

func checkPut(ctx context.Context,
	segments Segments, t testing.TB, expected map[string]struct{},
) {
	segs, err := segments.GetSegmentNamesFromRemote(ctx)
	if err != nil {
		t.Fatalf("get segs failed: %v", err)
	}
	segs_s := toStrSet(segs)
	if eq := reflect.DeepEqual(expected, segs_s); !eq {
		t.Fatalf("should equal. expected: %v, got: %v", expected, segs_s)
	}
}

func RollingTest(ctx context.Context, store *SegmentedWindowStoreG[uint32, string], segments Segments, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL * 2
	increment := TEST_SEGMENT_INTERVAL / 2

	checkErr(store.Put(ctx, uint32(0), optional.Some("zero"), startTime, TimeMeta{RecordTsMs: 0}), t)
	names := map[string]struct{}{segments.SegmentName(2): {}}
	debug.Fprint(os.Stderr, "1\n")
	checkPut(ctx, segments, t, names)

	checkErr(store.Put(ctx, uint32(1), optional.Some("one"), startTime+increment, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{segments.SegmentName(2): {}}
	debug.Fprint(os.Stderr, "2\n")
	checkPut(ctx, segments, t, names)

	checkErr(store.Put(ctx, uint32(2), optional.Some("two"), startTime+increment*2, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{segments.SegmentName(2): {}, segments.SegmentName(3): {}}
	debug.Fprint(os.Stderr, "3\n")
	checkPut(ctx, segments, t, names)

	checkErr(store.Put(ctx, uint32(4), optional.Some("four"), startTime+increment*4, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{
		segments.SegmentName(2): {},
		segments.SegmentName(3): {},
		segments.SegmentName(4): {},
	}
	debug.Fprint(os.Stderr, "4\n")
	checkPut(ctx, segments, t, names)

	checkErr(store.Put(ctx, uint32(5), optional.Some("five"), startTime+increment*5, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{
		segments.SegmentName(2): {},
		segments.SegmentName(3): {},
		segments.SegmentName(4): {},
	}
	debug.Fprint(os.Stderr, "5\n")
	checkPut(ctx, segments, t, names)

	debug.Fprint(os.Stderr, "6\n")
	resM, err := assertFetchG(ctx, store, 0, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected := map[string]struct{}{"zero": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatalf("should equal. expected: %v, got: %v", expected, resM)
	}
	debug.Fprint(os.Stderr, "7\n")

	resM, err = assertFetchG(ctx, store, 1, startTime-TEST_WINDOW_SIZE+increment, startTime+TEST_WINDOW_SIZE+increment)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"one": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "8\n")
	resM, err = assertFetchG(ctx, store, 2, startTime-TEST_WINDOW_SIZE+increment*2, startTime+TEST_WINDOW_SIZE+increment*2)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"two": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "9\n")
	resM, err = assertFetchG(ctx, store, 3, startTime-TEST_WINDOW_SIZE+increment*3, startTime+TEST_WINDOW_SIZE+increment*3)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "10\n")
	resM, err = assertFetchG(ctx, store, 4, startTime-TEST_WINDOW_SIZE+increment*4, startTime+TEST_WINDOW_SIZE+increment*4)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"four": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "11\n")
	resM, err = assertFetchG(ctx, store, 5, startTime-TEST_WINDOW_SIZE+increment*5, startTime+TEST_WINDOW_SIZE+increment*5)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"five": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "12\n")
	checkErr(store.Put(ctx, uint32(6), optional.Some("six"), startTime+increment*6, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{
		segments.SegmentName(3): {},
		segments.SegmentName(4): {},
		segments.SegmentName(5): {},
	}
	checkPut(ctx, segments, t, names)

	debug.Fprint(os.Stderr, "13\n")
	resM, err = assertFetchG(ctx, store, 0, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "14\n")
	resM, err = assertFetchG(ctx, store, 1, startTime-TEST_WINDOW_SIZE+increment, startTime+TEST_WINDOW_SIZE+increment)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "15\n")
	resM, err = assertFetchG(ctx, store, 2, startTime-TEST_WINDOW_SIZE+increment*2, startTime+TEST_WINDOW_SIZE+increment*2)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"two": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "16\n")
	resM, err = assertFetchG(ctx, store, 3, startTime-TEST_WINDOW_SIZE+increment*3, startTime+TEST_WINDOW_SIZE+increment*3)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "17\n")
	resM, err = assertFetchG(ctx, store, 4, startTime-TEST_WINDOW_SIZE+increment*4, startTime+TEST_WINDOW_SIZE+increment*4)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"four": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "18\n")
	resM, err = assertFetchG(ctx, store, 5, startTime-TEST_WINDOW_SIZE+increment*5, startTime+TEST_WINDOW_SIZE+increment*5)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"five": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "19\n")
	resM, err = assertFetchG(ctx, store, 6, startTime-TEST_WINDOW_SIZE+increment*6, startTime+TEST_WINDOW_SIZE+increment*6)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"six": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "20\n")
	checkErr(store.Put(ctx, uint32(7), optional.Some("seven"), startTime+increment*7, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{
		segments.SegmentName(3): {},
		segments.SegmentName(4): {},
		segments.SegmentName(5): {},
	}
	checkPut(ctx, segments, t, names)

	debug.Fprint(os.Stderr, "21\n")
	resM, err = assertFetchG(ctx, store, 0, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	debug.Fprint(os.Stderr, "22\n")
	resM, err = assertFetchG(ctx, store, 1, startTime-TEST_WINDOW_SIZE+increment, startTime+TEST_WINDOW_SIZE+increment)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 2, startTime-TEST_WINDOW_SIZE+increment*2, startTime+TEST_WINDOW_SIZE+increment*2)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"two": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 3, startTime-TEST_WINDOW_SIZE+increment*3, startTime+TEST_WINDOW_SIZE+increment*3)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 4, startTime-TEST_WINDOW_SIZE+increment*4, startTime+TEST_WINDOW_SIZE+increment*4)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"four": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 5, startTime-TEST_WINDOW_SIZE+increment*5, startTime+TEST_WINDOW_SIZE+increment*5)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"five": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 6, startTime-TEST_WINDOW_SIZE+increment*6, startTime+TEST_WINDOW_SIZE+increment*6)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"six": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 7, startTime-TEST_WINDOW_SIZE+increment*7, startTime+TEST_WINDOW_SIZE+increment*7)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"seven": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	checkErr(store.Put(ctx, uint32(8), optional.Some("eight"), startTime+increment*8, TimeMeta{RecordTsMs: 0}), t)
	names = map[string]struct{}{
		segments.SegmentName(4): {},
		segments.SegmentName(5): {},
		segments.SegmentName(6): {},
	}
	checkPut(ctx, segments, t, names)

	resM, err = assertFetchG(ctx, store, 0, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 1, startTime-TEST_WINDOW_SIZE+increment, startTime+TEST_WINDOW_SIZE+increment)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 2, startTime-TEST_WINDOW_SIZE+increment*2, startTime+TEST_WINDOW_SIZE+increment*2)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 3, startTime-TEST_WINDOW_SIZE+increment*3, startTime+TEST_WINDOW_SIZE+increment*3)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 4, startTime-TEST_WINDOW_SIZE+increment*4, startTime+TEST_WINDOW_SIZE+increment*4)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"four": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 5, startTime-TEST_WINDOW_SIZE+increment*5, startTime+TEST_WINDOW_SIZE+increment*5)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"five": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 6, startTime-TEST_WINDOW_SIZE+increment*6, startTime+TEST_WINDOW_SIZE+increment*6)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"six": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 7, startTime-TEST_WINDOW_SIZE+increment*7, startTime+TEST_WINDOW_SIZE+increment*7)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"seven": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	resM, err = assertFetchG(ctx, store, 8, startTime-TEST_WINDOW_SIZE+increment*8, startTime+TEST_WINDOW_SIZE+increment*8)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected = map[string]struct{}{"eight": {}}
	if !reflect.DeepEqual(expected, resM) {
		t.Fatal("should equal")
	}

	names = map[string]struct{}{
		segments.SegmentName(4): {},
		segments.SegmentName(5): {},
		segments.SegmentName(6): {},
	}
	checkPut(ctx, segments, t, names)
}
