package store

import (
	"context"
	"sharedlog-stream/pkg/optional"
	"testing"
)

func checkMapEqual(t testing.TB, expected map[int]string, got map[int]string) {
	if len(expected) != len(got) {
		t.Fatalf("expected and got have different length, expected. expected: %v, got: %v", expected, got)
	}
	for k, v := range expected {
		vgot := got[k]
		if vgot != v {
			t.Fatalf("k: %d, expected: %s, got: %s", k, v, vgot)
		}
	}
}

func checkErr(err error, t testing.TB) {
	if err != nil {
		t.Fatal(err.Error())
	}
}

/*
func ShouldNotIncludeDeletedFromRangeResult(ctx context.Context, store CoreKeyValueStore, t testing.TB) {
	checkErr(store.Put(ctx, 0, "zero"), t)
	checkErr(store.Put(ctx, 1, "one"), t)
	checkErr(store.Put(ctx, 2, "two"), t)
	checkErr(store.Delete(ctx, 0), t)
	checkErr(store.Delete(ctx, 1), t)

	expected := make(map[int]string)
	expected[2] = "two"

	val2, ok, err := store.Get(ctx, 2)
	if err != nil {
		t.Fatalf("fail to get 2: %v", err)
	}
	if !ok {
		t.Fatal("2 should be in the map")
	}
	if val2 != "two" {
		t.Fatalf("expected two, got %s", val2)
	}
	ret := make(map[int]string)
	err = store.Range(ctx, nil, nil, func(kt commtypes.KeyT, vt commtypes.ValueT) error {
		ret[kt.(int)] = vt.(string)
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret)
}
*/

func ShouldNotIncludeDeletedFromRangeResultG(ctx context.Context, store CoreKeyValueStoreG[int, string], t testing.TB) {
	checkErr(store.Put(ctx, 0, optional.Some("zero"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 1, optional.Some("one"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 2, optional.Some("two"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Delete(ctx, 0), t)
	checkErr(store.Delete(ctx, 1), t)

	expected := make(map[int]string)
	expected[2] = "two"

	val2, ok, err := store.Get(ctx, 2)
	if err != nil {
		t.Fatalf("fail to get 2: %v", err)
	}
	if !ok {
		t.Fatal("2 should be in the map")
	}
	if val2 != "two" {
		t.Fatalf("expected two, got %s", val2)
	}
	ret := make(map[int]string)
	err = store.Range(ctx, optional.None[int](), optional.None[int](), func(kt int, vt string) error {
		ret[kt] = vt
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret)
}

/*
func ShouldDeleteIfSerializedValueIsNull(ctx context.Context, store CoreKeyValueStore, t testing.TB) {
	checkErr(store.Put(ctx, 0, "zero"), t)
	checkErr(store.Put(ctx, 1, "one"), t)
	checkErr(store.Put(ctx, 2, "two"), t)
	checkErr(store.Put(ctx, 0, nil), t)
	checkErr(store.Put(ctx, 1, nil), t)
	expected := make(map[int]string)
	expected[2] = "two"

	val2, ok, err := store.Get(ctx, 2)
	if err != nil {
		t.Fatalf("fail to get 2: %v", err)
	}
	if !ok {
		t.Fatal("2 should be in the map")
	}
	if val2 != "two" {
		t.Fatalf("expected two, got %s", val2)
	}
	ret := make(map[int]string)
	err = store.Range(ctx, nil, nil, func(kt commtypes.KeyT, vt commtypes.ValueT) error {
		ret[kt.(int)] = vt.(string)
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret)
}
*/

func opStr(v string) optional.Option[string] {
	return optional.Some(v)
}

func ShouldDeleteIfSerializedValueIsNullG(ctx context.Context, store CoreKeyValueStoreG[int, string], t testing.TB) {
	checkErr(store.Put(ctx, 0, opStr("zero"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 1, opStr("one"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 2, opStr("two"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 0, optional.None[string](), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 1, optional.None[string](), TimeMeta{RecordTsMs: 0}), t)
	expected := make(map[int]string)
	expected[2] = "two"

	val2, ok, err := store.Get(ctx, 2)
	if err != nil {
		t.Fatalf("fail to get 2: %v", err)
	}
	if !ok {
		t.Fatal("2 should be in the map")
	}
	if val2 != "two" {
		t.Fatalf("expected two, got %s", val2)
	}
	ret := make(map[int]string)
	err = store.Range(ctx, optional.None[int](), optional.None[int](), func(kt int, vt string) error {
		ret[kt] = vt
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret)
}

func checkExists(expected bool, exists bool, t testing.TB) {
	if exists != expected {
		t.Fatalf("expected: %v, exists: %v", expected, exists)
	}
}

func checkGet(ctx context.Context, store CoreKeyValueStoreG[int, string], t testing.TB, key int, expected string) {
	ret, exists, err := store.Get(ctx, key)
	checkErr(err, t)
	checkExists(true, exists, t)
	if ret != expected {
		t.Fatalf("expected %v, got %v", expected, ret)
	}
}

func PutGetRange(ctx context.Context, store CoreKeyValueStoreG[int, string], t testing.TB) {
	checkErr(store.Put(ctx, 0, opStr("zero"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 1, opStr("one"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 2, opStr("two"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 4, opStr("four"), TimeMeta{RecordTsMs: 0}), t)
	checkErr(store.Put(ctx, 5, opStr("five"), TimeMeta{RecordTsMs: 0}), t)

	checkGet(ctx, store, t, 0, "zero")
	checkGet(ctx, store, t, 1, "one")
	checkGet(ctx, store, t, 2, "two")
	ret, exists, err := store.Get(ctx, 3)
	checkErr(err, t)
	checkExists(false, exists, t)
	if exists {
		t.Fatalf("expected not exists, got %v", ret)
	}
	checkGet(ctx, store, t, 4, "four")
	checkGet(ctx, store, t, 5, "five")

	checkErr(store.Delete(ctx, 5), t)

	expected := make(map[int]string)
	expected[2] = "two"
	expected[4] = "four"

	ret_range := make(map[int]string)
	err = store.Range(ctx, optional.Some(2), optional.Some(4), func(kt int, vt string) error {
		ret_range[kt] = vt
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret_range)

	ret_range = make(map[int]string)
	err = store.Range(ctx, optional.Some(2), optional.Some(6), func(kt int, vt string) error {
		ret_range[kt] = vt
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret_range)
}

/*
func checkGet(ctx context.Context, store CoreKeyValueStore, t testing.TB, key int, expected string) {
	ret, exists, err := store.Get(ctx, key)
	checkErr(err, t)
	checkExists(true, exists, t)
	if ret.(string) != expected {
		t.Fatalf("expected %v, got %v", expected, ret)
	}
}

func PutGetRange(ctx context.Context, store CoreKeyValueStore, t testing.TB) {
	checkErr(store.Put(ctx, 0, "zero"), t)
	checkErr(store.Put(ctx, 1, "one"), t)
	checkErr(store.Put(ctx, 2, "two"), t)
	checkErr(store.Put(ctx, 4, "four"), t)
	checkErr(store.Put(ctx, 5, "five"), t)

	checkGet(ctx, store, t, 0, "zero")
	checkGet(ctx, store, t, 1, "one")
	checkGet(ctx, store, t, 2, "two")
	ret, exists, err := store.Get(ctx, 3)
	checkErr(err, t)
	checkExists(false, exists, t)
	if ret != nil {
		t.Fatalf("expected nil, got %v", ret)
	}
	checkGet(ctx, store, t, 4, "four")
	checkGet(ctx, store, t, 5, "five")

	checkErr(store.Delete(ctx, 5), t)

	expected := make(map[int]string)
	expected[2] = "two"
	expected[4] = "four"

	ret_range := make(map[int]string)
	err = store.Range(ctx, 2, 4, func(kt commtypes.KeyT, vt commtypes.ValueT) error {
		ret_range[kt.(int)] = vt.(string)
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret_range)

	ret_range = make(map[int]string)
	err = store.Range(ctx, 2, 6, func(kt commtypes.KeyT, vt commtypes.ValueT) error {
		ret_range[kt.(int)] = vt.(string)
		return nil
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	checkMapEqual(t, expected, ret_range)
}
*/
