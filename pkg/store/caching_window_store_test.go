package store

import (
	"context"
	// "fmt"
	// "os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"

	// "strconv"
	"testing"
	// "time"
)

const (
	windowSize   = 10
	defaultTs    = 10
	maxCacheSize = 150
)

func getCachingWindowStore(ctx context.Context) *CachingWindowStoreG[string, string] {
	underlyingStore := NewInMemorySkipMapWindowStore[string, string]("test1", 50, windowSize,
		false, StringCompare)
	cachingStore := NewCachingWindowStoreG[string, string](ctx, windowSize, underlyingStore,
		func(k commtypes.KeyAndWindowStartTsG[string]) int64 {
			return int64(len(k.Key) + 8)
		}, commtypes.SizeOfString, maxCacheSize)
	return cachingStore
}

func TestShouldPutFetchFromCache(t *testing.T) {
	ctx := context.Background()
	cachingStore := getCachingWindowStore(ctx)
	err := cachingStore.Put(ctx, "key1", optional.Some("val1"), defaultTs, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	value, found, err := cachingStore.Get(ctx, "key1", defaultTs)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected to find value")
	}
	if value != "val1" {
		t.Fatal("expected to find value1")
	}
}

/*
func TestShouldPutFetchRangeFromCacheForNullKeyFrom(t *testing.T) {
	ctx := context.Background()
	cachingStore := getCachingWindowStore(ctx)
	err := cachingStore.Put(ctx, "a", optional.Some("a"), defaultTs, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	err = cachingStore.Put(ctx, "b", optional.Some("b"), defaultTs, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	err = cachingStore.Put(ctx, "c", optional.Some("c"), defaultTs+10, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	err = cachingStore.Put(ctx, "d", optional.Some("d"), defaultTs+20, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	err = cachingStore.Put(ctx, "e", optional.Some("e"), defaultTs+20, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	expected_key := []commtypes.KeyAndWindowStartTsG[string]{
		{Key: "a", WindowStartTs: defaultTs},
		{Key: "b", WindowStartTs: defaultTs},
		{Key: "c", WindowStartTs: defaultTs + 10},
		{Key: "d", WindowStartTs: defaultTs + 20},
	}
	expected_val := []string{"a", "b", "c", "d"}
	got_key := []commtypes.KeyAndWindowStartTsG[string]{}
	got_val := []string{}
	err = cachingStore.FetchWithKeyRange(ctx, "", "d",
		time.UnixMilli(defaultTs), time.UnixMilli(defaultTs+20),
		func(ts int64, key string, value string) error {
			got_key = append(got_key, commtypes.KeyAndWindowStartTsG[string]{Key: key, WindowStartTs: ts})
			got_val = append(got_val, value)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(got_key) != len(expected_key) {
		t.Fatalf("expected: %v, got: %v", expected_key, got_key)
	}
	if len(got_val) != len(expected_val) {
		t.Fatalf("expected: %v, got: %v", expected_val, got_val)
	}
	for i := 0; i < len(expected_key); i++ {
		if expected_key[i] != got_key[i] {
			t.Fatalf("expected: %v, got: %v", expected_key, got_key)
		}
		if expected_val[i] != got_val[i] {
			t.Fatalf("expected: %v, got: %v", expected_val, got_val)
		}
	}
}

func addItemToCache(ctx context.Context, cache *CachingWindowStoreG[string, string], t *testing.T) int {
	cacheSize := 0
	i := 0
	for cacheSize < maxCacheSize {
		key := strconv.Itoa(i)
		fmt.Fprintf(os.Stderr, "adding key %s to cache\n", key)
		err := cache.Put(ctx, key, optional.Some(key), defaultTs, TimeMeta{RecordTsMs: 0})
		if err != nil {
			t.Fatal(err)
		}
		cacheSize += len(key)*2 + 8 + int(3*ptrSize)
		i += 1
	}
	return i
}

func TestShouldFlushEvictedItemsIntoUnderlyingStore(t *testing.T) {
	fmt.Fprintf(os.Stderr, "TestShouldFlushEvictedItemsIntoUnderlyingStore\n")
	ctx := context.Background()
	cachingStore := getCachingWindowStore(ctx)
	added := addItemToCache(ctx, cachingStore, t)
	fmt.Fprintf(os.Stderr, "added %d items to cache\n", added-1)
	got_key := []commtypes.KeyAndWindowStartTsG[string]{}
	got_val := []string{}
	err := cachingStore.Fetch(ctx, "0", time.UnixMilli(defaultTs),
		time.UnixMilli(defaultTs), func(ts int64, key string, value string) error {
			got_key = append(got_key, commtypes.KeyAndWindowStartTsG[string]{Key: key, WindowStartTs: ts})
			got_val = append(got_val, value)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(got_key) != 1 {
		t.Errorf("expected to find 1 key, got %d", len(got_key))
	}
	if got_key[0].Key != "0" {
		t.Errorf("expected to find key 0, got %s", got_key[0].Key)
	}
	if got_key[0].WindowStartTs != defaultTs {
		t.Errorf("expected to find key 0, got %d", got_key[0].WindowStartTs)
	}
	if cachingStore.cache.len() != added-1 {
		t.Errorf("expected to find %d items in cache, got %d", added, cachingStore.cache.len())
	}
}
*/

func TestShouldFlushDirtyItemsWhenFlushedCalled(t *testing.T) {
	ctx := context.Background()
	cachingStore := getCachingWindowStore(ctx)
	err := cachingStore.Put(ctx, "1", optional.Some("a"), defaultTs, TimeMeta{RecordTsMs: 0})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cachingStore.Flush(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := cachingStore.wrappedStore.Get(ctx, "1", defaultTs)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected to find value")
	}
	if got != "a" {
		t.Fatal("expected to find value a")
	}
}
