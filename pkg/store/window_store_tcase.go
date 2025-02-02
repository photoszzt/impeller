package store

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"testing"
	"time"
)

const (
	TEST_WINDOW_SIZE      = int64(3)
	TEST_SEGMENT_INTERVAL = int64(60_000)
	TEST_RETENTION_PERIOD = 2 * TEST_SEGMENT_INTERVAL
)

func assertGet(ctx context.Context, store CoreWindowStore, k uint32, expected_val string, startTime int64) error {
	val, ok, err := store.Get(ctx, k, startTime)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("key %d should exists", 0)
	}
	if val != expected_val {
		return fmt.Errorf("should be %s, but got %s", expected_val, val)
	}
	return nil
}

func assertGetG(ctx context.Context, store CoreWindowStoreG[uint32, string], k uint32, expected_val string, startTime int64) error {
	val, ok, err := store.Get(ctx, k, startTime)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("key %d should exists", 0)
	}
	if val != expected_val {
		return fmt.Errorf("should be %s, but got %s", expected_val, val)
	}
	return nil
}

func assertFetch(ctx context.Context, store CoreWindowStore, k uint32, timeFrom int64, timeTo int64) (map[string]struct{}, error) {
	res := make(map[string]struct{})
	err := store.Fetch(ctx, k, time.UnixMilli(timeFrom), time.UnixMilli(timeTo),
		func(i int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			// debug.Fprintf(os.Stderr, "ts: %d, kt %v, vt %v\n", i, kt, vt)
			val := vt.(string)
			res[val] = struct{}{}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("fail to fetch: %v", err)
	}
	return res, nil
}

func putFirstBatch(ctx context.Context, store CoreWindowStore, startTime int64) error {
	err := store.Put(ctx, uint32(0), "zero", startTime)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(1), "one", startTime+1)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two", startTime+2)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(4), "four", startTime+4)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(5), "five", startTime+5)
	return err
}

func putSecondBatch(ctx context.Context, store CoreWindowStore, startTime int64) error {
	err := store.Put(ctx, uint32(2), "two+1", startTime+3)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two+2", startTime+4)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two+3", startTime+5)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two+4", startTime+6)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two+5", startTime+7)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two+6", startTime+8)
	return err
}

func check_get(ctx context.Context, k uint32, expected_val string, time int64, store CoreWindowStore, t testing.TB) {
	val, ok, err := store.Get(ctx, k, time)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("key %d should exists\n", 0)
	}
	val_str, ok := val.(string)
	if !ok {
		val_str = string(val.([]byte))
	}
	if val_str != expected_val {
		t.Fatalf("should be %s, but got %s", expected_val, val)
	}
}

func GetAndRangeTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatch(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	k := uint32(0)
	expected_val := "zero"
	check_get(ctx, k, expected_val, startTime, store, t)

	k = uint32(1)
	expected_val = "one"
	check_get(ctx, k, expected_val, startTime+1, store, t)

	k = uint32(2)
	expected_val = "two"
	check_get(ctx, k, expected_val, startTime+2, store, t)

	k = uint32(4)
	expected_val = "four"
	check_get(ctx, k, expected_val, startTime+4, store, t)

	k = uint32(5)
	expected_val = "five"
	check_get(ctx, k, expected_val, startTime+5, store, t)

	resM, err := assertFetch(ctx, store, 0, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("fail to fetch: %v", err)
	}
	expected := "zero"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	err = putSecondBatch(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to put second batch")
	}

	k = uint32(2)
	expected_val = "two+1"
	check_get(ctx, k, expected_val, startTime+3, store, t)
	err = assertGet(ctx, store, 2, "two+2", startTime+4)
	if err != nil {
		t.Fatalf("assertGet err: %v", err)
	}

	err = assertGet(ctx, store, 2, "two+3", startTime+5)
	if err != nil {
		t.Fatalf("assertGet err: %v", err)
	}

	err = assertGet(ctx, store, 2, "two+4", startTime+6)
	if err != nil {
		t.Fatalf("assertGet err: %v", err)
	}

	err = assertGet(ctx, store, 2, "two+5", startTime+7)
	if err != nil {
		t.Fatalf("assertGet err: %v", err)
	}

	err = assertGet(ctx, store, 2, "two+6", startTime+8)
	if err != nil {
		t.Fatalf("assertGet err: %v", err)
	}

	// TODO: fetch is broken
	resM, err = assertFetch(ctx, store, 2, startTime-2-TEST_WINDOW_SIZE, startTime-2+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	if len(resM) != 0 {
		t.Fatalf("expected empty list but got %v", resM)
	}

	resM, err = assertFetch(ctx, store, 2, startTime-1-TEST_WINDOW_SIZE, startTime-1+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	if _, ok := resM["two"]; !ok {
		t.Fatalf("expected list contains two but got %v", resM)
	}

	resM, err = assertFetch(ctx, store, 2, startTime-TEST_WINDOW_SIZE, startTime+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+1-TEST_WINDOW_SIZE, startTime+1+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+2-TEST_WINDOW_SIZE, startTime+2+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+3-TEST_WINDOW_SIZE, startTime+3+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+4-TEST_WINDOW_SIZE, startTime+4+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+5-TEST_WINDOW_SIZE, startTime+5+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+6-TEST_WINDOW_SIZE, startTime+6+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+7-TEST_WINDOW_SIZE, startTime+7+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+8-TEST_WINDOW_SIZE, startTime+8+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+9-TEST_WINDOW_SIZE, startTime+9+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+10-TEST_WINDOW_SIZE, startTime+10+TEST_WINDOW_SIZE)
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

	resM, err = assertFetch(ctx, store, 2, startTime+11-TEST_WINDOW_SIZE, startTime+11+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	expected = "two+6"
	if _, ok := resM[expected]; !ok {
		t.Fatalf("expected list contains %s but got %v", expected, resM)
	}

	resM, err = assertFetch(ctx, store, 2, startTime+12-TEST_WINDOW_SIZE, startTime+12+TEST_WINDOW_SIZE)
	if err != nil {
		t.Fatalf("assertFetch err: %v", err)
	}
	if len(resM) != 0 {
		t.Fatalf("expected empty list but got %v", resM)
	}
}

func ShouldGetAllNonDeletedMsgsTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := store.Put(ctx, uint32(0), "zero", startTime)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(1), "one", startTime+1)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(2), "two", startTime+2)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(3), "three", startTime+3)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(4), "four", startTime+4)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(1), nil, startTime+1)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	err = store.Put(ctx, uint32(3), nil, startTime+3)
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	k := uint32(0)
	expected := "zero"
	check_get(ctx, k, expected, startTime, store, t)

	k = 2
	expected = "two"
	check_get(ctx, k, expected, startTime+2, store, t)

	k = 4
	expected = "four"
	check_get(ctx, k, expected, startTime+4, store, t)

	val, _, err := store.Get(ctx, uint32(1), startTime+1)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if val != nil {
		t.Errorf("expected key 1 doesn't exist but got %v", val)
	}

	val, _, err = store.Get(ctx, uint32(3), startTime+3)
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if val != nil {
		t.Errorf("expected key 3 doesn't exist but got %v", val)
	}
}

func ExpirationTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	currentTime := int64(0)
	err := store.Put(ctx, uint32(1), "one", int64(currentTime))
	if err != nil {
		t.Fatalf("put err: %v", err)
	}
	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), "two", int64(currentTime))
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), "three", int64(currentTime))
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), "four", int64(currentTime))
	if err != nil {
		t.Fatalf("put err: %v", err)
	}

	// increase current time to the full RETENTION_PERIOD to expire first record
	currentTime += TEST_RETENTION_PERIOD / 4
	err = store.Put(ctx, uint32(1), "five", int64(currentTime))
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
	err = store.Put(ctx, uint32(1), "six", int64(currentTime))
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

func ShouldGetAllTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatch(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.IterAll(func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
			Key:       uint32(0),
			Value:     "zero",
			Timestamp: startTime,
		},
		{
			Key:       uint32(1),
			Value:     "one",
			Timestamp: startTime + 1,
		},
		{
			Key:       uint32(2),
			Value:     "two",
			Timestamp: startTime + 2,
		},
		{
			Key:       uint32(4),
			Value:     "four",
			Timestamp: startTime + 4,
		},
		{
			Key:       uint32(5),
			Value:     "five",
			Timestamp: startTime + 5,
		},
	}
	checkSlice(ref_msgs, msgs, t)
}

func checkSlice(ref_msgs []commtypes.Message, msgs []commtypes.Message, t testing.TB) {
	if len(msgs) != len(ref_msgs) {
		t.Fatalf("store contains different number of elements. expected: %d, got %d", len(ref_msgs), len(msgs))
	}

	for idx, msg := range msgs {
		ref_msg := ref_msgs[idx]
		if msg.Key.(uint32) != ref_msg.Key {
			t.Fatalf("%dth item's key should be %v but got %v",
				idx, ref_msg.Key, msg.Key)
		}
		if msg.Value.(string) != ref_msg.Value {
			t.Fatalf("%dth item's val should be %v but got %v",
				idx, ref_msg.Value, msg.Value)
		}
		if msg.Timestamp != ref_msg.Timestamp {
			t.Fatalf("%dth item's ts should be %v but got %v",
				idx, ref_msg.Timestamp, msg.Timestamp)
		}
	}
}

func outOfOrderPut(ctx context.Context, store CoreWindowStore, startTime int64) error {
	err := store.Put(ctx, uint32(4), "four", startTime+4)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(0), "zero", startTime)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(2), "two", startTime+2)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(3), "three", startTime+3)
	if err != nil {
		return err
	}
	err = store.Put(ctx, uint32(1), "one", startTime+1)
	if err != nil {
		return err
	}
	return nil
}

func ShouldGetAllReturnTimestampOrderedTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := outOfOrderPut(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to setup entries in store: %v", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.IterAll(func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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

func FetchRangeTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatch(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	debug.Fprint(os.Stderr, "1\n")
	msgs := make([]commtypes.Message, 0)
	err = store.FetchWithKeyRange(ctx, uint32(0), uint32(1),
		time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime+TEST_WINDOW_SIZE),
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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

func PutAndFetchBeforeTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatch(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime-TEST_WINDOW_SIZE),
		time.UnixMilli(startTime), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+1), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+2), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+3), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+4), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+5), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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

	err = putSecondBatch(ctx, store, startTime)
	if err != nil {
		t.Fatal(err.Error())
	}

	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime-1-TEST_WINDOW_SIZE), time.UnixMilli(startTime-1), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+0-TEST_WINDOW_SIZE), time.UnixMilli(startTime+0), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+1), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+2), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+3), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+4), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+5), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+6), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+7), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+8), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+9), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+10), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+11), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+12), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+13), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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

func PutAndFetchAfterTest(ctx context.Context, store CoreWindowStore, t testing.TB) {
	startTime := TEST_SEGMENT_INTERVAL - 4
	err := putFirstBatch(ctx, store, startTime)
	if err != nil {
		t.Fatalf("fail to set up window store: %v\n", err)
	}

	debug.Fprint(os.Stderr, "11\n")
	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(0), time.UnixMilli(startTime),
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

	debug.Fprint(os.Stderr, "12\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(1), time.UnixMilli(startTime+1), time.UnixMilli(startTime+1+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+2), time.UnixMilli(startTime+2+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(3), time.UnixMilli(startTime+3), time.UnixMilli(startTime+3+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(4), time.UnixMilli(startTime+4), time.UnixMilli(startTime+4+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(5), time.UnixMilli(startTime+5), time.UnixMilli(startTime+5+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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

	err = putSecondBatch(ctx, store, startTime)
	if err != nil {
		t.Fatal(err.Error())
	}

	debug.Fprint(os.Stderr, "21\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime-2), time.UnixMilli(startTime-2+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime-1), time.UnixMilli(startTime-1+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime), time.UnixMilli(startTime+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+1), time.UnixMilli(startTime+1+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		{Key: uint32(2), Value: "two", Timestamp: startTime + 2},
		{Key: uint32(2), Value: "two+1", Timestamp: startTime + 3},
		{Key: uint32(2), Value: "two+2", Timestamp: startTime + 4},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "25\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+2), time.UnixMilli(startTime+2+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+3), time.UnixMilli(startTime+3+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+4+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+5+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+6+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+7), time.UnixMilli(startTime+7+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		{Key: uint32(2), Value: "two+5", Timestamp: startTime + 7},
		{Key: uint32(2), Value: "two+6", Timestamp: startTime + 8},
	}
	checkSlice(ref_msgs, msgs, t)

	debug.Fprint(os.Stderr, "31\n")
	msgs = make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+8),
		time.UnixMilli(startTime+8+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+9+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
	err = store.Fetch(ctx, uint32(2), time.UnixMilli(startTime+10), time.UnixMilli(startTime+10+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+11+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
		time.UnixMilli(startTime+12+TEST_WINDOW_SIZE), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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

func FetchDuplicates(ctx context.Context, store CoreWindowStore, t testing.TB) {
	currentTime := 0
	err := store.Put(ctx, uint32(1), "one", int64(currentTime))
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}
	err = store.Put(ctx, uint32(1), "one-2", int64(currentTime))
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	currentTime += int(TEST_WINDOW_SIZE) * 10
	err = store.Put(ctx, uint32(1), "two", int64(currentTime))
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}
	err = store.Put(ctx, uint32(1), "two-2", int64(currentTime))
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	currentTime += int(TEST_WINDOW_SIZE) * 10
	err = store.Put(ctx, uint32(1), "three", int64(currentTime))
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}
	err = store.Put(ctx, uint32(1), "three-2", int64(currentTime))
	if err != nil {
		t.Fatalf("fail to put err: %v", err)
	}

	msgs := make([]commtypes.Message, 0)
	err = store.Fetch(ctx, uint32(1), time.UnixMilli(0), time.UnixMilli(TEST_WINDOW_SIZE*10), func(ts int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
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
