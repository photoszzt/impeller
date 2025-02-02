package store

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/txn_data"
)

type CachingKeyValueStoreBackedByChangelogG[K comparable, V any] struct {
	wrappedStore      KeyValueStoreBackedByChangelogG[K, V]
	cache             *Cache[K, V]
	flushCallbackFunc KVStoreCacheFlushCallbackFunc[K, V]
	name              string
}

var (
	_ CoreKeyValueStoreG[int, int]              = (*CachingKeyValueStoreBackedByChangelogG[int, int])(nil)
	_ KeyValueStoreBackedByChangelogG[int, int] = (*CachingKeyValueStoreBackedByChangelogG[int, int])(nil)
	_ CachedKeyValueStore[int, int]             = (*CachingKeyValueStoreBackedByChangelogG[int, int])(nil)
)

func CommonKVStoreCacheFlushCallback[K comparable, V any](ctx context.Context, entries []LRUElement[K, V],
	wrappedStore CoreKeyValueStoreG[K, V],
	flushCallbackFunc KVStoreCacheFlushCallbackFunc[K, V],
) error {
	for _, entry := range entries {
		newVal := entry.entry.value
		oldVal, ok, err := wrappedStore.Get(ctx, entry.key)
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
			err = wrappedStore.Put(ctx, entry.key, entry.entry.value, entry.entry.tm)
			if err != nil {
				return err
			}
			if flushCallbackFunc != nil {
				err = flushCallbackFunc(ctx, commtypes.MessageG[K, commtypes.ChangeG[V]]{
					Key:           optional.Some(entry.key),
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

func NewCachingKeyValueStoreBackedByChangelogG[K comparable, V any](ctx context.Context,
	store KeyValueStoreBackedByChangelogG[K, V],
	sizeOfK func(K) int64,
	sizeOfV func(V) int64,
	maxCacheBytes int64,
) *CachingKeyValueStoreBackedByChangelogG[K, V] {
	c := &CachingKeyValueStoreBackedByChangelogG[K, V]{
		name:         store.Name(),
		wrappedStore: store,
	}
	c.cache = NewCache(func(entries []LRUElement[K, V]) error {
		return CommonKVStoreCacheFlushCallback(ctx, entries, c.wrappedStore.(CoreKeyValueStoreG[K, V]), c.flushCallbackFunc)
	}, sizeOfK, sizeOfV, maxCacheBytes)
	return c
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat, keySerde commtypes.SerdeG[K], valSerde commtypes.SerdeG[V]) error {
	return c.wrappedStore.SetKVSerde(serdeFormat, keySerde, valSerde)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[K, V]] {
	return c.wrappedStore.GetKVSerde()
}
func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Name() string { return c.name }
func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	entry, found := c.cache.get(key)
	if found {
		v, ok := entry.value.Take()
		return v, ok, nil
	} else {
		return c.wrappedStore.Get(ctx, key)
	}
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetCacheFlushCallback(flushCallbackFunc KVStoreCacheFlushCallbackFunc[K, V]) {
	c.flushCallbackFunc = flushCallbackFunc
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Range(ctx context.Context, from optional.Option[K], to optional.Option[K], iterFunc func(K, V) error) error {
	panic("not implemented")
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) ApproximateNumEntries() (uint64, error) {
	panic("not implemented")
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Put(ctx context.Context, key K, value optional.Option[V], tm TimeMeta) error {
	return c.putInternal(key, value, tm)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) putInternal(key K, value optional.Option[V], tm TimeMeta) error {
	return c.cache.PutMaybeEvict(key, LRUEntry[V]{value: value, isDirty: true, tm: tm})
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) PutIfAbsent(ctx context.Context, key K, value V, tm TimeMeta) (optional.Option[V], error) {
	opV, err := c.getInternal(ctx, key)
	if err != nil {
		return optional.Option[V]{}, err
	}
	if opV.IsNone() {
		err = c.putInternal(key, optional.Some(value), tm)
		if err != nil {
			return optional.Option[V]{}, err
		}
	}
	return opV, nil
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) getInternal(ctx context.Context, key K) (optional.Option[V], error) {
	entry, found := c.cache.get(key)
	if found {
		return entry.value, nil
	} else {
		v, found, err := c.wrappedStore.Get(ctx, key)
		if err != nil {
			return optional.Option[V]{}, err
		}
		if !found {
			return optional.Option[V]{}, nil
		}
		retV := optional.Some(v)
		return retV, nil
	}
}

/*
	func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) PutAll(ctx context.Context, msgs []*commtypes.Message) error {
		maxTs := int64(0)
		for _, msg := range msgs {
			if msg.Timestamp > maxTs {
				maxTs = msg.Timestamp
			}
			err := c.Put(ctx, msg.Key.(K), optional.Some(msg.Value.(V)), maxTs)
			if err != nil {
				return err
			}
		}
		return nil
	}
*/
func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Delete(ctx context.Context, key K) error {
	return c.cache.put(key, LRUEntry[V]{value: optional.None[V]()})
}
func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) TableType() TABLE_TYPE { return IN_MEM }
func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetTrackParFunc(f exactly_once_intr.TrackProdSubStreamFunc) {
	c.wrappedStore.SetTrackParFunc(f)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error {
	return c.wrappedStore.PutWithoutPushToChangelog(ctx, key, value)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Flush(ctx context.Context) (uint32, error) {
	c.cache.flush(nil)
	return c.wrappedStore.Flush(ctx)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) ChangelogIsSrc() bool {
	return c.wrappedStore.ChangelogIsSrc()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) ChangelogTopicName() string {
	return c.wrappedStore.ChangelogTopicName()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) ConsumeOneLogEntry(ctx context.Context, parNum uint8) (int, error) {
	return c.wrappedStore.ConsumeOneLogEntry(ctx, parNum)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) ConfigureExactlyOnce(
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	c.wrappedStore.ConfigureExactlyOnce(rem, guarantee)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Stream() sharedlog_stream.Stream {
	return c.wrappedStore.Stream()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) GetInitialProdSeqNum() uint64 {
	return c.wrappedStore.GetInitialProdSeqNum()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) ResetInitialProd() {
	c.wrappedStore.ResetInitialProd()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetLastMarkerSeq(lastMarkerSeq uint64) {
	c.wrappedStore.SetLastMarkerSeq(lastMarkerSeq)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SubstreamNum() uint8 {
	return c.wrappedStore.SubstreamNum()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) Snapshot(ctx context.Context, tpLogOff []commtypes.TpLogOff,
	unprocessed []commtypes.ChkptMetaData, resetBg bool,
) {
	c.wrappedStore.Snapshot(ctx, tpLogOff, unprocessed, resetBg)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) WaitForAllSnapshot() error {
	return c.wrappedStore.WaitForAllSnapshot()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetSnapshotCallback(ctx context.Context, f KVSnapshotCallback[K, V]) {
	c.wrappedStore.SetSnapshotCallback(ctx, f)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetStreamFlushCallbackFunc(f exactly_once_intr.FlushCallbackFunc) {
	c.wrappedStore.SetStreamFlushCallbackFunc(f)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) RestoreFromSnapshot(snapshot [][]byte) error {
	return c.wrappedStore.RestoreFromSnapshot(snapshot)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error) {
	return c.wrappedStore.FindLastEpochMetaWithAuxData(ctx, parNum)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) BuildKeyMeta(ctx context.Context, kms map[string][]txn_data.KeyMaping) error {
	return c.wrappedStore.BuildKeyMeta(ctx, kms)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) SetInstanceId(id uint8) {
	c.wrappedStore.SetInstanceId(id)
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) GetInstanceId() uint8 {
	return c.wrappedStore.GetInstanceId()
}

func (c *CachingKeyValueStoreBackedByChangelogG[K, V]) OutputRemainingStats() {}
