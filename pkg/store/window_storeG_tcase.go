package store

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/optional"
	"testing"
	"time"
)

func assertFetchG(ctx context.Context, store CoreWindowStoreG[uint32, string],
	k uint32, timeFrom int64, timeTo int64,
) (map[string]struct{}, error) {
	res := make(map[string]struct{})
	err := store.Fetch(ctx, k, time.UnixMilli(timeFrom), time.UnixMilli(timeTo),
		func(i int64, kt uint32, vt string) error {
			debug.Fprintf(os.Stderr, "ts: %d, kt %v, vt %v\n", i, kt, vt)
			res[vt] = struct{}{}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("fail to fetch: %v", err)
	}
	return res, nil
}

func putFirstBatchG(ctx context.Context, store CoreWindowStoreG[uint32, string], startTime int64) error {
	err := store.Put(ctx, uint32(0), optional.Some("zero"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(1), optional.Some("one"), startTime+1, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two"), startTime+2, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(4), optional.Some("four"), startTime+4, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(5), optional.Some("five"), startTime+5, TimeMeta{RecordTsMs: 0})
	return err
}

func putSecondBatchG(ctx context.Context, store CoreWindowStoreG[uint32, string], startTime int64) error {
	err := store.Put(ctx, uint32(2), optional.Some("two+1"), startTime+3, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two+2"), startTime+4, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two+3"), startTime+5, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two+4"), startTime+6, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two+5"), startTime+7, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two+6"), startTime+8, TimeMeta{RecordTsMs: 0})
	return err
}

func check_getG(ctx context.Context, k uint32, expected_val string, time int64, store CoreWindowStoreG[uint32, string], t testing.TB) {
	val, ok, err := store.Get(ctx, k, time)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("key %d should exists\n", 0)
	}
	if val != expected_val {
		t.Fatalf("should be %s, but got %s", expected_val, val)
	}
}

func GetAndRangeTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	k := uint32(0)
	expected_val := "zero"
	check_getG(ctx, k, expected_val, startTime, store, t)

	k = uint32(1)
	expected_val = "one"
	check_getG(ctx, k, expected_val, startTime+1, store, t)

	k = uint32(2)
	expected_val = "two"
	check_getG(ctx, k, expected_val, startTime+2, store, t)

	k = uint32(4)
	expected_val = "four"
	check_getG(ctx, k, expected_val, startTime+4, store, t)

	k = uint32(5)
	expected_val = "five"
	check_getG(ctx, k, expected_val, startTime+5, store, t)

	resM, err := assertFetchG(ctx, store, 0, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected := "zero"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	err = putSecondBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to put second batch")
	}

	k = uint32(2)
	expected_val = "two+1"
	check_getG(ctx, k, expected_val, startTime+3, store, t)
	if err := assertGetG(ctx, store, k, "two+2", startTime+4); err != nil {
		t.Fatalf("assertGetG err: %v", err)
	}
	if err = assertGetG(ctx, store, k, "two+3", startTime+5); err != nil {
		t.Fatalf("assertGetG err: %v", err)
	}
	err = assertGetG(ctx, store, k, "two+4", startTime+6)
	if err != nil {
		t.Fatalf("assertGetG err: %v", err)
	}

	err = assertGetG(ctx, store, k, "two+5", startTime+7)
	if err != nil {
		t.Fatalf("assertGetG err: %v", err)
	}

	err = assertGetG(ctx, store, k, "two+6", startTime+8)
	if err != nil {
		t.Fatalf("assertGetG err: %v", err)
	}

	resM, err = assertFetchG(ctx, store, k, startTime-2-TEST_WINDOW_SIZE, startTime-2+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	if len(resM) != 0 {
		t.Fatalf("expected empty list but got %v", resM)
	}

	resM, err = assertFetchG(ctx, store, k, startTime-1-TEST_WINDOW_SIZE, startTime-1+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	if _, ok := resM["two"]; !ok {
		t.Fatalf("expected list contains two but got %v", resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+1-TEST_WINDOW_SIZE, startTime+1+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+2-TEST_WINDOW_SIZE, startTime+2+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+3-TEST_WINDOW_SIZE, startTime+3+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+4-TEST_WINDOW_SIZE, startTime+4+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+5-TEST_WINDOW_SIZE, startTime+5+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+6-TEST_WINDOW_SIZE, startTime+6+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+1"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+7-TEST_WINDOW_SIZE, startTime+7+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+2"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+8-TEST_WINDOW_SIZE, startTime+8+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+3"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+9-TEST_WINDOW_SIZE, startTime+9+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+4"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+10-TEST_WINDOW_SIZE, startTime+10+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+5"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+11-TEST_WINDOW_SIZE, startTime+11+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetchG(ctx, store, 2, startTime+12-TEST_WINDOW_SIZE, startTime+12+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	if len(resM) != 0 {
		t.Fatalf("expected empty list but got %v", resM)
	}
}

func ShouldGetAllNonDeletedMsgsTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := store.Put(ctx, uint32(0), optional.Some("zero"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(1), optional.Some("one"), startTime+1, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(2), optional.Some("two"), startTime+2, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(3), optional.Some("three"), startTime+3, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(4), optional.Some("four"), startTime+4, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(1), optional.None[string](), startTime+1, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(3), optional.None[string](), startTime+3, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	k := uint32(0)
	expected := "zero"
	check_getG(ctx, k, expected, startTime, store, t)

	k = 2
	expected = "two"
	check_getG(ctx, k, expected, startTime+2, store, t)

	k = 4
	expected = "four"
	check_getG(ctx, k, expected, startTime+4, store, t)

	val, exists, err := store.Get(ctx, uint32(1), startTime+1)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if exists {
		t.Errorf("expected key 1 doesn't exist but got %v", val)
	}

	val, exists, err = store.Get(ctx, uint32(3), startTime+3)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if exists {
		t.Errorf("expected key 3 doesn't exist but got %v", val)
	}
}

func ExpirationTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	currentTime := int64(0)
	err := store.Put(ctx, uint32(1), optional.Some("one"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}
	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), optional.Some("two"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), optional.Some("three"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), optional.Some("four"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	// increase current time to the full RETENTION_PERIOD to expire first record
	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), optional.Some("five"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	k := uint32(1)
	expected := "two"
	val, ok, err := store.Get(ctx, k, TEST_RETENTION_PERIOD/4)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	expected = "three"
	val, ok, err = store.Get(ctx, k, TEST_RETENTION_PERIOD/2)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	expected = "four"
	val, ok, err = store.Get(ctx, k, 3*TEST_RETENTION_PERIOD/4)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	expected = "five"
	val, ok, err = store.Get(ctx, k, TEST_RETENTION_PERIOD)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), optional.Some("six"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	expected = "three"
	val, ok, err = store.Get(ctx, k, TEST_RETENTION_PERIOD/2)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	expected = "four"
	val, ok, err = store.Get(ctx, k, 3*TEST_RETENTION_PERIOD/4)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	expected = "five"
	val, ok, err = store.Get(ctx, k, TEST_RETENTION_PERIOD)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}

	expected = "six"
	val, ok, err = store.Get(ctx, k, 5*TEST_RETENTION_PERIOD/4)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected key %d exists", 0)
	}
	if val != expected {
		t.Fatalf("got unexpected val: %s, expected %s", val, expected)
	}
}

func ShouldGetAllTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.IterAll(func(ts int64, kt uint32, vt string) error {
		msg := commtypes.Message{
			Key:       kt,
			Value:     vt,
			Timestamp: ts,
		}
		msgs = append(msgs, msg)
		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(4), Value: "four", Timestamp: startTime + 4},
		{Key: uint32(5), Value: "five", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)
}

func outOfOrderPutG(ctx context.Context, store CoreWindowStoreG[uint32, string], startTime int64) error {
	err := store.Put(ctx, uint32(4), optional.Some("four"), startTime+4, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(0), optional.Some("zero"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), optional.Some("two"), startTime+2, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(3), optional.Some("three"), startTime+3, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(1), optional.Some("one"), startTime+1, TimeMeta{RecordTsMs: 0})
	if err != nil {
		return err
	}
	return nil
}

func ShouldGetAllReturnTimestampOrderedTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := outOfOrderPutG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to setup entries in store: %v", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.IterAll(func(ts int64, kt uint32, vt string) error {
		msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
		msgs = append(msgs, msg)
		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(3), Value: "three", Timestamp: startTime + 3},
		{Key: uint32(4), Value: "four", Timestamp: startTime + 4},
	}
	checkSlice(ref_msgs, msgs, t)
}

func FetchRangeTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	debug.Fprint(os.Stderr, "1\n")
	msgs := make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(0), uint32(1),
		time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "2\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(1), uint32(1),
		time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "3\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(1), uint32(3),
		time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "4\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(0), uint32(5),
		time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "5\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(0), uint32(5),
		time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE+5),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(4), Value: "four", Timestamp: startTime + 4},
		{Key: uint32(5), Value: "five", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "6\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(0), uint32(5),
		time.UnixMilli(startTime+2),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE+5),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(4), Value: "four", Timestamp: startTime + 4},
		{Key: uint32(5), Value: "five", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "7\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(4), uint32(5),
		time.UnixMilli(startTime+2),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "8\n")
	msgs = make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(0), uint32(3),
		time.UnixMilli(startTime+3),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE+5),
		func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)
}

func PutAndFetchBeforeTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{
			Key:       uint32(0),
			Value:     "zero",
			Timestamp: startTime,
		},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(1), time.UnixMilli(startTime+1-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+1), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+2-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+2), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(3), time.UnixMilli(startTime+3-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+3), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(4), time.UnixMilli(startTime+4-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+4), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(4), Value: "four", Timestamp: startTime + 4},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(5), time.UnixMilli(startTime+5-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+5), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(5), Value: "five", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)

	err = putSecondBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatal(err.Error())
	}

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime-1-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime-1), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+0-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+0), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+1-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+1), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+2-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+2), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+3-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+3), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+4-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+4), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+5-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+5), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+6-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+6), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+7-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+7), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+8-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+8), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+9-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+9), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+10-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+10), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+11-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+11), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+12-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+12), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+13-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+13), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)
}

func PutAndFetchAfterTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	debug.Fprint(os.Stderr, "11\n")
	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "12\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(1), time.UnixMilli(startTime+1), time.UnixMilli(startTime+1+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
		msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
		msgs = append(msgs, msg)
		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(1), Value: "one", Timestamp: startTime + 1},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "13\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+2), time.UnixMilli(startTime+2+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
		msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
		msgs = append(msgs, msg)
		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "14\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(3), time.UnixMilli(startTime+3), time.UnixMilli(startTime+3+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
		msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
		msgs = append(msgs, msg)
		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "15\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(4), time.UnixMilli(startTime+4),
		time.UnixMilli(startTime+4+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(4), Value: "four", Timestamp: startTime + 4},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "16\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(5), time.UnixMilli(startTime+5),
		time.UnixMilli(startTime+5+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(5), Value: "five", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)

	err = putSecondBatchG(ctx, store, startTime)
	if err != nil {
		t.Fatal(err.Error())
	}

	debug.Fprint(os.Stderr, "21\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime-2),
		time.UnixMilli(startTime-2+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "22\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime-1),
		time.UnixMilli(startTime-1+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "23\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "24\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+1),
		time.UnixMilli(startTime+1+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "25\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+2),
		time.UnixMilli(startTime+2+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "26\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+3),
		time.UnixMilli(startTime+3+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "27\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+4),
		time.UnixMilli(startTime+4+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "28\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+5),
		time.UnixMilli(startTime+5+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+3", Timestamp: startTime + 5},
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "29\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+6),
		time.UnixMilli(startTime+6+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{
				Key:       kt,
				Value:     vt,
				Timestamp: ts,
			}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+4", Timestamp: startTime + 6},
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "30\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+7),
		time.UnixMilli(startTime+7+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{
				Key:       kt,
				Value:     vt,
				Timestamp: ts,
			}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{
			Key:       uint32(2),
			Value:     "two+5",
			Timestamp: startTime + 7,
		},
		{
			Key:       uint32(2),
			Value:     "two+6",
			Timestamp: startTime + 8,
		},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "31\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+8),
		time.UnixMilli(startTime+8+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "32\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+9),
		time.UnixMilli(startTime+9+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "33\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+10),
		time.UnixMilli(startTime+10+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "34\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+11),
		time.UnixMilli(startTime+11+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "35\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+12),
		time.UnixMilli(startTime+12+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)
}

func PutSameKeyTsTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := store.Put(ctx, uint32(0), "zero", startTime)
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
	}
	checkSlice(ref_msgs, msgs, t)

	err = store.Put(ctx, uint32(0), "zero", startTime)
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	err = store.Put(ctx, uint32(0), "zero+", startTime)
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	err = store.Put(ctx, uint32(0), "zero++", startTime)
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(0), Value: "zero+", Timestamp: startTime},
		{Key: uint32(0), Value: "zero++", Timestamp: startTime},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+1-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+1+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+2-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+2+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+3-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+3+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+4-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+4+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)
}

func PutSameKeyTsTestG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := store.Put(ctx, uint32(0), optional.Some("zero"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs := []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
	}
	checkSlice(ref_msgs, msgs, t)

	err = store.Put(ctx, uint32(0), optional.Some("zero"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	err = store.Put(ctx, uint32(0), optional.Some("zero+"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	err = store.Put(ctx, uint32(0), optional.Some("zero++"), startTime, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = []commtypes.Message{
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(0), Value: "zero", Timestamp: startTime},
		{Key: uint32(0), Value: "zero+", Timestamp: startTime},
		{Key: uint32(0), Value: "zero++", Timestamp: startTime},
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+1-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+1+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+2-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+2+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+3-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+3+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	checkSlice(ref_msgs, msgs, t)

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime+4-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+4+TEST_WINDOW_SIZE), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{Key: kt, Value: vt, Timestamp: ts}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}
	ref_msgs = make([]commtypes.Message, 0)
	checkSlice(ref_msgs, msgs, t)
}

func FetchDuplicatesG(ctx context.Context, store CoreWindowStoreG[uint32, string], t testing.TB) {
	currentTime := 0
	err := store.Put(ctx, uint32(1), optional.Some("one"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}
	err = store.Put(ctx, uint32(1), optional.Some("one-2"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	currentTime += int(TEST_WINDOW_SIZE) * 10
	err = store.Put(ctx, uint32(1), optional.Some("two"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}
	err = store.Put(ctx, uint32(1), optional.Some("two-2"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	currentTime += int(TEST_WINDOW_SIZE) * 10
	err = store.Put(ctx, uint32(1), optional.Some("three"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}
	err = store.Put(ctx, uint32(1), optional.Some("three-2"), int64(currentTime), TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(1), time.UnixMilli(0),
		time.UnixMilli(TEST_WINDOW_SIZE*10), func(ts int64, kt uint32, vt string) error {
			msg := commtypes.Message{
				Key:       kt,
				Value:     vt,
				Timestamp: ts,
			}
			msgs = append(msgs, msg)
			return nil
		})
	if err != nil {
		t.Fatal(err.Error())
	}

	ref_msgs := []commtypes.Message{
		{
			Key:       uint32(1),
			Value:     "one",
			Timestamp: 0,
		},
		{
			Key:       uint32(1),
			Value:     "one-2",
			Timestamp: 0,
		},
		{
			Key:       uint32(1),
			Value:     "two",
			Timestamp: TEST_WINDOW_SIZE * 10,
		},
		{
			Key:       uint32(1),
			Value:     "two-2",
			Timestamp: TEST_WINDOW_SIZE * 10,
		},
	}
	checkSlice(ref_msgs, msgs, t)
}
