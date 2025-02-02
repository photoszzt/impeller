package store

import (
	"context"
	"testing"
)

func getSkipmapKeyValueStoreG() CoreKeyValueStoreG[int, string] {
	store := NewInMemorySkipmapKeyValueStoreG[int, string]("test1", IntLessFunc)
	return store
}

func TestShouldNotIncludeDeletedFromRangeResult(t *testing.T) {
	ctx := context.Background()
	skipmapStoreG := getSkipmapKeyValueStoreG()
	ShouldNotIncludeDeletedFromRangeResultG(ctx, skipmapStoreG, t)
}

func TestShouldDeleteIfSerializedValueIsNull(t *testing.T) {
	ctx := context.Background()
	skipmapStoreG := getSkipmapKeyValueStoreG()
	ShouldDeleteIfSerializedValueIsNullG(ctx, skipmapStoreG, t)
}
