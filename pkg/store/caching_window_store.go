package store

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/txn_data"
	"time"
)

type CachingWindowStoreG[K comparable, V any] struct {
	cache             *Cache[commtypes.KeyAndWindowStartTsG[K], V]
	wrappedStore      CoreWindowStoreG[K, V]
	flushCallbackFunc WindowStoreCacheFlushCallbackFunc[K, V]
}

var (
	_ = CoreWindowStoreG[int, int](&CachingWindowStoreG[int, int]{})
	_ = CachedWindowStateStore[int, int](&CachingWindowStoreG[int, int]{})
)

func NewCachingWindowStoreG[K comparable, V any](ctx context.Context,
	windowSize int64,
	store WindowStoreBackedByChangelogG[K, V],
	sizeOfK func(commtypes.KeyAndWindowStartTsG[K]) int64,
	sizeOfV func(V) int64,
	maxCacheBytes int64,
) *CachingWindowStoreG[K, V] {
	c := &CachingWindowStoreG[K, V]{
		wrappedStore: store,
	}
	c.cache = NewCache(func(entries []LRUElement[commtypes.KeyAndWindowStartTsG[K], V]) error {
		return CommonWindowStoreCacheFlushCallback(ctx, entries, c.wrappedStore.(CoreWindowStoreG[K, V]), c.flushCallbackFunc, windowSize)
	}, sizeOfK, sizeOfV, maxCacheBytes)
	return c
}

func (c *CachingWindowStoreG[K, V]) Name() string { return c.wrappedStore.Name() }
func (c *CachingWindowStoreG[K, V]) Put(ctx context.Context, key K,
	value optional.Option[V], windowStartTimestamp int64, tm TimeMeta,
) error {
	return c.cache.PutMaybeEvict(commtypes.KeyAndWindowStartTsG[K]{Key: key, WindowStartTs: windowStartTimestamp},
		LRUEntry[V]{value: value, isDirty: true, tm: tm})
}

func (c *CachingWindowStoreG[K, V]) PutWithoutPushToChangelogG(ctx context.Context,
	key K, value optional.Option[V], windowStartTs int64,
) error {
	return c.wrappedStore.PutWithoutPushToChangelogG(ctx, key, value, windowStartTs)
}

func (c *CachingWindowStoreG[K, V]) PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error {
	keyTs := key.(commtypes.KeyAndWindowStartTsG[K])
	return c.wrappedStore.PutWithoutPushToChangelogG(ctx, keyTs.Key, optional.Some(value.(V)), keyTs.WindowStartTs)
}

func (c *CachingWindowStoreG[K, V]) Get(ctx context.Context, key K,
	windowStartTimestamp int64,
) (V, bool, error) {
	entry, found := c.cache.get(commtypes.KeyAndWindowStartTsG[K]{Key: key, WindowStartTs: windowStartTimestamp})
	if found {
		v, ok := entry.value.Take()
		return v, ok, nil
	} else {
		return c.wrappedStore.Get(ctx, key, windowStartTimestamp)
	}
}

func (c *CachingWindowStoreG[K, V]) Fetch(ctx context.Context, key K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64 /* ts */, K, V) error,
) error {
	panic("not implemented")
}

func (c *CachingWindowStoreG[K, V]) FetchWithKeyRange(ctx context.Context, keyFrom K,
	keyTo K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	panic("not implemented")
}

func (c *CachingWindowStoreG[K, V]) FetchAll(ctx context.Context, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	panic("not implemented")
}

func (c *CachingWindowStoreG[K, V]) IterAll(iterFunc func(int64, K, V) error) error {
	panic("not implemented")
}

func (c *CachingWindowStoreG[K, V]) Flush(ctx context.Context) (uint32, error) {
	c.cache.flush(nil)
	return c.wrappedStore.Flush(ctx)
}
func (c *CachingWindowStoreG[K, V]) TableType() TABLE_TYPE { return IN_MEM }

func (s *CachingWindowStoreG[K, V]) SetCacheFlushCallback(f WindowStoreCacheFlushCallbackFunc[K, V]) {
	s.flushCallbackFunc = f
}

func (s *CachingWindowStoreG[K, V]) Snapshot(ctx context.Context, tplogoff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool) {
	s.wrappedStore.Snapshot(ctx, tplogoff, chkptMeta, resetBg)
}

func (s *CachingWindowStoreG[K, V]) WaitForAllSnapshot() error {
	return s.wrappedStore.WaitForAllSnapshot()
}

func (s *CachingWindowStoreG[K, V]) SetWinSnapshotCallback(ctx context.Context, f WinSnapshotCallback[K, V]) {
	s.wrappedStore.SetWinSnapshotCallback(ctx, f)
}

func (s *CachingWindowStoreG[K, V]) RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error {
	return s.wrappedStore.RestoreFromSnapshot(ctx, snapshot)
}

func (s *CachingWindowStoreG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat,
	keySerde commtypes.SerdeG[commtypes.KeyAndWindowStartTsG[K]],
	origKeySerde commtypes.SerdeG[K],
	valSerde commtypes.SerdeG[V],
) error {
	return nil
}

func (s *CachingWindowStoreG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]] {
	return s.wrappedStore.GetKVSerde()
}

func (s *CachingWindowStoreG[K, V]) BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error {
	return s.wrappedStore.BuildKeyMeta(kms)
}

func (c *CachingWindowStoreG[K, V]) SetInstanceId(id uint8) {
	c.wrappedStore.SetInstanceId(id)
}

func (c *CachingWindowStoreG[K, V]) GetInstanceId() uint8 {
	return c.wrappedStore.GetInstanceId()
}
