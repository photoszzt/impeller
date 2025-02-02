package store

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/txn_data"
	"time"
)

// If the entry is expired, it's
// not removed from the cache. It will try to insert again when it's evicted while
// the wrapped store might have already advanced. If the retension time is long, it
// would generate an entry in the changelog that is not needed.
type CachingWindowStoreBackedByChangelogG[K comparable, V any] struct {
	cache             *Cache[commtypes.KeyAndWindowStartTsG[K], V]
	wrappedStore      WindowStoreBackedByChangelogG[K, V]
	flushCallbackFunc WindowStoreCacheFlushCallbackFunc[K, V]
}

var (
	_ = CoreWindowStoreG[int, int](&CachingWindowStoreBackedByChangelogG[int, int]{})
	_ = WindowStoreBackedByChangelogG[int, int](&CachingWindowStoreBackedByChangelogG[int, int]{})
	_ = CachedWindowStateStore[int, int](&CachingWindowStoreBackedByChangelogG[int, int]{})
)

func CommonWindowStoreCacheFlushCallback[K any, V any](ctx context.Context,
	entries []LRUElement[commtypes.KeyAndWindowStartTsG[K], V], wrappedStore CoreWindowStoreG[K, V],
	flushCallbackFunc WindowStoreCacheFlushCallbackFunc[K, V], windowSize int64,
) error {
	for _, entry := range entries {
		newVal := entry.entry.value
		kTs := entry.key
		oldVal, ok, err := wrappedStore.Get(ctx, kTs.Key, kTs.WindowStartTs)
		if err != nil {
			return err
		}
		if newVal.IsSome() || ok {
			var change commtypes.ChangeG[V]
			if ok {
				change = commtypes.ChangeG[V]{
					OldVal: optional.Some(oldVal),
					NewVal: newVal,
				}
			} else {
				change = commtypes.ChangeG[V]{
					OldVal: optional.None[V](),
					NewVal: newVal,
				}
			}
			err = wrappedStore.Put(ctx, entry.key.Key, entry.entry.value, entry.key.WindowStartTs, entry.entry.tm)
			if err != nil {
				return err
			}
			t, err := commtypes.NewTimeWindow(kTs.WindowStartTs, kTs.WindowStartTs+windowSize)
			if err != nil {
				return err
			}
			k := commtypes.WindowedKeyG[K]{
				Key:    kTs.Key,
				Window: t,
			}
			if flushCallbackFunc != nil {
				err = flushCallbackFunc(ctx, commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[V]]{
					Key:           optional.Some(k),
					Value:         optional.Some(change),
					TimestampMs:   entry.entry.tm.RecordTsMs,
					StartProcTime: entry.entry.tm.StartProcTs,
				})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewCachingWindowStoreBackedByChangelogG[K comparable, V any](ctx context.Context,
	windowSize int64,
	store WindowStoreBackedByChangelogG[K, V],
	sizeOfK func(commtypes.KeyAndWindowStartTsG[K]) int64,
	sizeOfV func(V) int64,
	maxCacheBytes int64,
) *CachingWindowStoreBackedByChangelogG[K, V] {
	c := &CachingWindowStoreBackedByChangelogG[K, V]{
		wrappedStore: store,
	}
	c.cache = NewCache(func(entries []LRUElement[commtypes.KeyAndWindowStartTsG[K], V]) error {
		return CommonWindowStoreCacheFlushCallback(ctx, entries, c.wrappedStore.(CoreWindowStoreG[K, V]), c.flushCallbackFunc, windowSize)
	}, sizeOfK, sizeOfV, maxCacheBytes)
	return c
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) Name() string { return c.wrappedStore.Name() }
func (c *CachingWindowStoreBackedByChangelogG[K, V]) Put(ctx context.Context, key K,
	value optional.Option[V], windowStartTimestamp int64, tm TimeMeta,
) error {
	return c.cache.PutMaybeEvict(commtypes.KeyAndWindowStartTsG[K]{Key: key, WindowStartTs: windowStartTimestamp},
		LRUEntry[V]{value: value, isDirty: true, tm: tm})
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) PutWithoutPushToChangelogG(ctx context.Context,
	key K, value optional.Option[V], windowStartTs int64,
) error {
	return c.wrappedStore.PutWithoutPushToChangelogG(ctx, key, value, windowStartTs)
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error {
	keyTs := key.(commtypes.KeyAndWindowStartTsG[K])
	return c.wrappedStore.PutWithoutPushToChangelogG(ctx, keyTs.Key, optional.Some(value.(V)), keyTs.WindowStartTs)
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) Get(ctx context.Context, key K,
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

func (c *CachingWindowStoreBackedByChangelogG[K, V]) Fetch(ctx context.Context, key K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64 /* ts */, K, V) error,
) error {
	panic("not implemented")
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) FetchWithKeyRange(ctx context.Context, keyFrom K,
	keyTo K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	panic("not implemented")
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) FetchAll(ctx context.Context, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	panic("not implemented")
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) IterAll(iterFunc func(int64, K, V) error) error {
	panic("not implemented")
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) Flush(ctx context.Context) (uint32, error) {
	c.cache.flush(nil)
	return c.wrappedStore.Flush(ctx)
}
func (c *CachingWindowStoreBackedByChangelogG[K, V]) TableType() TABLE_TYPE { return IN_MEM }
func (c *CachingWindowStoreBackedByChangelogG[K, V]) SetTrackParFunc(f exactly_once_intr.TrackProdSubStreamFunc) {
	c.wrappedStore.SetTrackParFunc(f)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) ConsumeOneLogEntry(ctx context.Context, parNum uint8) (int, error) {
	return s.wrappedStore.ConsumeOneLogEntry(ctx, parNum)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) ConfigureExactlyOnce(rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	s.wrappedStore.ConfigureExactlyOnce(rem, guarantee)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) ChangelogTopicName() string {
	return s.wrappedStore.ChangelogTopicName()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) GetInitialProdSeqNum() uint64 {
	return s.wrappedStore.GetInitialProdSeqNum()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) ResetInitialProd() {
	s.wrappedStore.ResetInitialProd()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) SetLastMarkerSeq(lastMarkerSeq uint64) {
	s.wrappedStore.SetLastMarkerSeq(lastMarkerSeq)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) Stream() sharedlog_stream.Stream {
	return s.wrappedStore.Stream()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) SubstreamNum() uint8 {
	return s.wrappedStore.SubstreamNum()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) SetCacheFlushCallback(f WindowStoreCacheFlushCallbackFunc[K, V]) {
	s.flushCallbackFunc = f
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) Snapshot(ctx context.Context, tplogoff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool) {
	s.wrappedStore.Snapshot(ctx, tplogoff, chkptMeta, resetBg)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) WaitForAllSnapshot() error {
	return s.wrappedStore.WaitForAllSnapshot()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) SetWinSnapshotCallback(ctx context.Context, f WinSnapshotCallback[K, V]) {
	s.wrappedStore.SetWinSnapshotCallback(ctx, f)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error {
	return s.wrappedStore.RestoreFromSnapshot(ctx, snapshot)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat,
	keySerde commtypes.SerdeG[commtypes.KeyAndWindowStartTsG[K]],
	origKeySerde commtypes.SerdeG[K],
	valSerde commtypes.SerdeG[V],
) error {
	return nil
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]] {
	return s.wrappedStore.GetKVSerde()
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error) {
	return s.wrappedStore.FindLastEpochMetaWithAuxData(ctx, parNum)
}

func (s *CachingWindowStoreBackedByChangelogG[K, V]) BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error {
	return s.wrappedStore.BuildKeyMeta(kms)
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) SetStreamFlushCallbackFunc(f exactly_once_intr.FlushCallbackFunc) {
	c.wrappedStore.SetStreamFlushCallbackFunc(f)
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) SetInstanceId(id uint8) {
	c.wrappedStore.SetInstanceId(id)
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) GetInstanceId() uint8 {
	return c.wrappedStore.GetInstanceId()
}

func (c *CachingWindowStoreBackedByChangelogG[K, V]) OutputRemainingStats() {
	c.wrappedStore.OutputRemainingStats()
}
