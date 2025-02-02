package store

import (
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"testing"
)

func TestShouldKeepTrackOfMostRecentlyAndLeastRecentlyUsed(t *testing.T) {
	toInsert := []commtypes.Message{
		{Key: "K1", Value: "V1"},
		{Key: "K2", Value: "V2"},
		{Key: "K3", Value: "V3"},
		{Key: "K4", Value: "V4"},
		{Key: "K5", Value: "V5"},
	}
	cache := NewCache(func(entries []LRUElement[string, string]) error { return nil },
		func(k string) int64 { return int64(len(k)) }, func(v string) int64 { return int64(len(v)) }, 4096)
	for _, msg := range toInsert {
		err := cache.put(msg.Key.(string), LRUEntry[string]{value: optional.Some(msg.Value.(string)), isDirty: true})
		if err != nil {
			t.Fatal(err)
		}
		head := cache.first()
		tail := cache.last()
		headV, _ := head.Value().Take()
		tailV, _ := tail.Value().Take()
		if headV != msg.Value.(string) {
			t.Errorf("expected %s, got %s", msg.Value.(string), headV)
		}
		if tailV != toInsert[0].Value.(string) {
			t.Errorf("expected %s, got %s", toInsert[0].Value.(string), tailV)
		}
		if cache.flushes() != 0 {
			t.Errorf("expected 0, got %d", cache.flushes())
		}
		if cache.hits() != 0 {
			t.Errorf("expected 0, got %d", cache.hits())
		}
		if cache.misses() != 0 {
			t.Errorf("expected 0, got %d", cache.misses())
		}
		if cache.overwrites() != 0 {
			t.Errorf("expected 0, got %d", cache.overwrites())
		}
	}
}

func TestShouldPutGet(t *testing.T) {
	cache := NewCache(func(entries []LRUElement[int, int]) error { return nil },
		commtypes.SizeOfInt, commtypes.SizeOfInt, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(10), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(1, LRUEntry[int]{value: optional.Some(11), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(2, LRUEntry[int]{value: optional.Some(12), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	entry, found := cache.get(0)
	if !found {
		t.Errorf("expected found, got not found")
	}
	v, _ := entry.Value().Take()
	if v != 10 {
		t.Errorf("expected 10, got %d", v)
	}
	entry, found = cache.get(1)
	if !found {
		t.Errorf("expected found, got not found")
	}
	v, _ = entry.Value().Take()
	if v != 11 {
		t.Errorf("expected 10, got %d", v)
	}
	entry, found = cache.get(2)
	if !found {
		t.Errorf("expected found, got not found")
	}
	v, _ = entry.Value().Take()
	if v != 12 {
		t.Errorf("expected 10, got %d", v)
	}
}

func TestShouldPutIfAbsent(t *testing.T) {
	cache := NewCache(func(entries []LRUElement[int, int]) error { return nil },
		commtypes.SizeOfInt, commtypes.SizeOfInt, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(10), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	_, _, err = cache.putIfAbsent(0, LRUEntry[int]{value: optional.Some(20), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	_, _, err = cache.putIfAbsent(1, LRUEntry[int]{value: optional.Some(30), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	entry, found := cache.get(0)
	if !found {
		t.Errorf("expected found, got not found")
	}
	v, _ := entry.Value().Take()
	if v != 10 {
		t.Errorf("expected 11, got %d", v)
	}
	entry, found = cache.get(1)
	if !found {
		t.Errorf("expected found, got not found")
	}
	v, _ = entry.Value().Take()
	if v != 30 {
		t.Errorf("expected 11, got %d", v)
	}
}

func TestShouldDeleteAndUpdateSize(t *testing.T) {
	cache := NewCache(func(entries []LRUElement[int, int]) error { return nil },
		commtypes.SizeOfInt, commtypes.SizeOfInt, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(10), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	deleted, found := cache.delete(0)
	if !found {
		t.Errorf("expected found, got not found")
	}
	dVal, _ := deleted.Value().Take()
	if dVal != 10 {
		t.Errorf("expected 10, got %d", dVal)
	}
}

func TestShouldOverwriteAll(t *testing.T) {
	cache := NewCache(func(entries []LRUElement[int, int]) error { return nil },
		func(k int) int64 { return 4 }, func(v int) int64 { return 4 }, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(0), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(0, LRUEntry[int]{value: optional.Some(1), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(0, LRUEntry[int]{value: optional.Some(2), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	entry, _ := cache.get(0)
	v, _ := entry.Value().Take()
	if v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
	if cache.overwrites() != 2 {
		t.Errorf("expected 2, got %d", cache.overwrites())
	}
}

func TestShouldEvictEldestEntry(t *testing.T) {
	cache := NewCache(func(entries []LRUElement[int, int]) error { return nil },
		commtypes.SizeOfInt, commtypes.SizeOfInt, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(10), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(1, LRUEntry[int]{value: optional.Some(20), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(2, LRUEntry[int]{value: optional.Some(30), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.evict()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	_, found := cache.get(0)
	if found {
		t.Errorf("expected not found, got found")
	}
}

func TestFlushTwice(t *testing.T) {
	flushed := make([]LRUElement[int, int], 0)
	cache := NewCache(func(entries []LRUElement[int, int]) error {
		flushed = append(flushed, entries...)
		return nil
	}, func(k int) int64 { return 4 }, func(v int) int64 { return 4 }, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(10), isDirty: true})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(1, LRUEntry[int]{value: optional.Some(20), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(2, LRUEntry[int]{value: optional.Some(30), isDirty: true})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.flush(nil)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if cache.len() != 3 {
		t.Errorf("expected cache still contains 3 element, got %v", cache.len())
	}
	if len(flushed) != 2 {
		t.Errorf("expected flushing 3 elements, got %v", len(flushed))
	}
	err = cache.flush(nil)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if len(flushed) != 2 {
		t.Errorf("snd flush should not add new elements, got %v elements", len(flushed))
	}
}

func TestShouldFlushDirtEntriesOnEviction(t *testing.T) {
	flushed := make([]LRUElement[int, int], 0)
	cache := NewCache(func(entries []LRUElement[int, int]) error {
		flushed = append(flushed, entries...)
		return nil
	}, func(k int) int64 { return 4 }, func(v int) int64 { return 4 }, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.Some(10), isDirty: true})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(1, LRUEntry[int]{value: optional.Some(20), isDirty: false})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(2, LRUEntry[int]{value: optional.Some(30), isDirty: true})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.evict()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if len(flushed) != 2 {
		t.Errorf("expected 2, got %d, flushed contains: %+v", len(flushed), flushed)
	}
	if flushed[0].key != 0 {
		t.Errorf("expected 0, got %d", flushed[0].key)
	}
	v, _ := flushed[0].entry.value.Take()
	if v != 10 {
		t.Errorf("expected 10, got %d", v)
	}
	if flushed[1].key != 2 {
		t.Errorf("expected 2, got %d", flushed[1].key)
	}
	v, _ = flushed[1].entry.value.Take()
	if v != 30 {
		t.Errorf("expected 30, got %d", v)
	}
	if cache.flushes() != 1 {
		t.Errorf("expected 1, got %d", cache.flushes())
	}
}

func TestShouldRemoveDeletedValuesOnFlush(t *testing.T) {
	cache := NewCache(func(entries []LRUElement[int, int]) error { return nil },
		func(k int) int64 { return 4 }, func(v int) int64 { return 4 }, 2048)
	err := cache.put(0, LRUEntry[int]{value: optional.None[int](), isDirty: true})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	err = cache.put(1, LRUEntry[int]{value: optional.Some(20), isDirty: true})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	cache.flush(nil)
	if cache.len() != 1 {
		t.Errorf("expected 1, got %d", cache.len())
	}
	ret, ok := cache.get(1)
	if !ok {
		t.Errorf("expected found, got not found")
	}
	if ret.value.IsNone() {
		t.Errorf("expected present, got not present")
	}
}

func TestBasicPutGet(t *testing.T) {
	toInsert := []commtypes.MessageG[string, string]{
		{Key: optional.Some("K1"), Value: optional.Some("V1")},
		{Key: optional.Some("K2"), Value: optional.Some("V2")},
		{Key: optional.Some("K3"), Value: optional.Some("V3")},
		{Key: optional.Some("K4"), Value: optional.Some("V4")},
		{Key: optional.Some("K5"), Value: optional.Some("V5")},
	}
	size := 5 * (4 + 3*ptrSize)
	cache := NewCache(func(entries []LRUElement[string, string]) error { return nil },
		commtypes.SizeOfString, commtypes.SizeOfString, size)
	for _, msg := range toInsert {
		err := cache.PutMaybeEvict(msg.Key.Unwrap(), LRUEntry[string]{value: msg.Value, isDirty: true})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	}

	for _, msg := range toInsert {
		entry, ok := cache.get(msg.Key.Unwrap())
		if !ok {
			t.Errorf("expected found, got not found")
		}
		v, _ := entry.Value().Take()
		expVal := msg.Value.Unwrap()
		if v != expVal {
			t.Errorf("expected %s, got %s", expVal, v)
		}
		if !entry.IsDirty() {
			t.Error("expected dirty, got clean")
		}
	}
}

func TestEvic(t *testing.T) {
	toInsert := []commtypes.MessageG[string, string]{
		{Key: optional.Some("K1"), Value: optional.Some("V1")},
		{Key: optional.Some("K2"), Value: optional.Some("V2")},
		{Key: optional.Some("K3"), Value: optional.Some("V3")},
		{Key: optional.Some("K4"), Value: optional.Some("V4")},
		{Key: optional.Some("K5"), Value: optional.Some("V5")},
	}
	expected := []commtypes.MessageG[string, string]{{Key: optional.Some("K1"), Value: optional.Some("V1")}}
	size := 4 + 3*ptrSize
	received := make([]commtypes.MessageG[string, string], 0, 5)
	cache := NewCache(func(entries []LRUElement[string, string]) error {
		for _, entry := range entries {
			v, ok := entry.entry.value.Take()
			if !ok {
				t.Errorf("expected present, got not present")
			}
			received = append(received, commtypes.MessageG[string, string]{Key: optional.Some(entry.key), Value: optional.Some(v)})
		}
		return nil
	}, commtypes.SizeOfString, commtypes.SizeOfString, size)
	for _, msg := range toInsert {
		err := cache.PutMaybeEvict(msg.Key.Unwrap(), LRUEntry[string]{value: msg.Value, isDirty: true})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	}
	for idx, expect := range expected {
		recKey := received[idx].Key.Unwrap()
		expKey := expect.Key.Unwrap()
		if recKey != expKey {
			t.Errorf("expected %s, got %s", expKey, recKey)
		}
		recVal := received[idx].Value.Unwrap()
		expVal := expected[idx].Value.Unwrap()
		if recVal != expVal {
			t.Errorf("expected %s, got %s", expVal, recVal)
		}
	}
	if cache.evicts() != 4 {
		t.Errorf("expected 4, got %d", cache.evicts())
	}
}
