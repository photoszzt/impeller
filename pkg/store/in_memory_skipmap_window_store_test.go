package store

import (
	"context"
	"testing"
)

func getSkipMapWindowStore(retainDuplicates bool) *InMemorySkipMapWindowStoreG[uint32, string] {
	return NewInMemorySkipMapWindowStore[uint32, string]("test1", TEST_RETENTION_PERIOD, TEST_WINDOW_SIZE, retainDuplicates,
		CompareFuncG[uint32](IntegerCompare[uint32]))
}

func TestSkipMapGetAndRange(t *testing.T) {
	ctx := context.Background()
	store := getSkipMapWindowStore(false)
	GetAndRangeTestG(ctx, store, t)
}

func TestSkipMapShouldGetAllNonDeletedMsgs(t *testing.T) {
	ctx := context.Background()
	store := getSkipMapWindowStore(false)
	ShouldGetAllNonDeletedMsgsTestG(ctx, store, t)
}

func TestSkipMapExpiration(t *testing.T) {
	ctx := context.Background()
	store := getSkipMapWindowStore(false)
	ExpirationTestG(ctx, store, t)
}

func TestSkipMapShouldGetAll(t *testing.T) {
	ctx := context.Background()
	store := getSkipMapWindowStore(false)
	ShouldGetAllTestG(ctx, store, t)
}

func TestSkipMapShouldGetAllReturnTimestampOrdered(t *testing.T) {
	ctx := context.Background()
	store := getSkipMapWindowStore(false)
	ShouldGetAllReturnTimestampOrderedTestG(ctx, store, t)
}

func TestSkipMapFetchRange(t *testing.T) {
	store := getSkipMapWindowStore(false)
	ctx := context.Background()
	FetchRangeTestG(ctx, store, t)
}

func TestSkipMapPutAndFetchBefore(t *testing.T) {
	store := getSkipMapWindowStore(false)
	ctx := context.Background()
	PutAndFetchBeforeTestG(ctx, store, t)
}

func TestSkipMapPutAndFetchAfter(t *testing.T) {
	store := getSkipMapWindowStore(false)
	ctx := context.Background()
	PutAndFetchAfterTestG(ctx, store, t)
}

func TestSkipMapPutSameKeyTs(t *testing.T) {
	store := getSkipMapWindowStore(true)
	ctx := context.Background()
	PutSameKeyTsTestG(ctx, store, t)
}
