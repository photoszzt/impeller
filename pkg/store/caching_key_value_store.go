package store

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/txn_data"
)

type CachingKeyValueStoreG[K comparable, V any] struct {
	wrappedStore      CoreKeyValueStoreG[K, V]
	cache             *Cache[K, V]
	flushCallbackFunc KVStoreCacheFlushCallbackFunc[K, V]
	name              string
}

var (
	_ CoreKeyValueStoreG[int, int]  = (*CachingKeyValueStoreG[int, int])(nil)
	_ CachedKeyValueStore[int, int] = (*CachingKeyValueStoreG[int, int])(nil)
)

func NewCachingKeyValueStore[K comparable, V any](ctx context.Context,
	store CoreKeyValueStoreG[K, V],
	sizeOfK func(K) int64,
	sizeOfV func(V) int64,
	maxCacheBytes int64,
) *CachingKeyValueStoreG[K, V] {
	c := &CachingKeyValueStoreG[K, V]{
		name:         store.Name(),
		wrappedStore: store,
	}
	c.cache = NewCache(func(entries []LRUElement[K, V]) error {
		return CommonKVStoreCacheFlushCallback(ctx, entries, c.wrappedStore, c.flushCallbackFunc)
	}, sizeOfK, sizeOfV, maxCacheBytes)
	return c
}

func (c *CachingKeyValueStoreG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat, keySerde commtypes.SerdeG[K], valSerde commtypes.SerdeG[V]) error {
	return c.wrappedStore.SetKVSerde(serdeFormat, keySerde, valSerde)
}

func (c *CachingKeyValueStoreG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[K, V]] {
	return c.wrappedStore.GetKVSerde()
}
func (c *CachingKeyValueStoreG[K, V]) Name() string { return c.name }
func (c *CachingKeyValueStoreG[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	entry, found := c.cache.get(key)
	if found {
		v, ok := entry.value.Take()
		return v, ok, nil
	} else {
		return c.wrappedStore.Get(ctx, key)
	}
}

func (c *CachingKeyValueStoreG[K, V]) SetCacheFlushCallback(flushCallbackFunc KVStoreCacheFlushCallbackFunc[K, V]) {
	c.flushCallbackFunc = flushCallbackFunc
}

func (c *CachingKeyValueStoreG[K, V]) Range(ctx context.Context, from optional.Option[K], to optional.Option[K], iterFunc func(K, V) error) error {
	panic("not implemented")
}

func (c *CachingKeyValueStoreG[K, V]) ApproximateNumEntries() (uint64, error) {
	panic("not implemented")
}

func (c *CachingKeyValueStoreG[K, V]) Put(ctx context.Context, key K, value optional.Option[V], tm TimeMeta) error {
	return c.putInternal(key, value, tm)
}

func (c *CachingKeyValueStoreG[K, V]) putInternal(key K, value optional.Option[V], tm TimeMeta) error {
	return c.cache.PutMaybeEvict(key, LRUEntry[V]{value: value, isDirty: true, tm: tm})
}

func (c *CachingKeyValueStoreG[K, V]) PutIfAbsent(ctx context.Context, key K, value V, tm TimeMeta) (optional.Option[V], error) {
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

func (c *CachingKeyValueStoreG[K, V]) getInternal(ctx context.Context, key K) (optional.Option[V], error) {
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
	func (c *CachingKeyValueStoreG[K, V]) PutAll(ctx context.Context, msgs []*commtypes.Message) error {
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
func (c *CachingKeyValueStoreG[K, V]) Delete(ctx context.Context, key K) error {
	return c.cache.put(key, LRUEntry[V]{value: optional.None[V]()})
}
func (c *CachingKeyValueStoreG[K, V]) TableType() TABLE_TYPE { return IN_MEM }
func (c *CachingKeyValueStoreG[K, V]) PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error {
	return c.wrappedStore.PutWithoutPushToChangelog(ctx, key, value)
}

func (c *CachingKeyValueStoreG[K, V]) Flush(ctx context.Context) (uint32, error) {
	c.cache.flush(nil)
	return c.wrappedStore.Flush(ctx)
}

func (c *CachingKeyValueStoreG[K, V]) Snapshot(ctx context.Context, tpLogOff []commtypes.TpLogOff,
	unprocessed []commtypes.ChkptMetaData, resetBg bool,
) {
	c.wrappedStore.Snapshot(ctx, tpLogOff, unprocessed, resetBg)
}

func (c *CachingKeyValueStoreG[K, V]) WaitForAllSnapshot() error {
	return c.wrappedStore.WaitForAllSnapshot()
}

func (c *CachingKeyValueStoreG[K, V]) SetSnapshotCallback(ctx context.Context, f KVSnapshotCallback[K, V]) {
	c.wrappedStore.SetSnapshotCallback(ctx, f)
}

func (c *CachingKeyValueStoreG[K, V]) RestoreFromSnapshot(snapshot [][]byte) error {
	return c.wrappedStore.RestoreFromSnapshot(snapshot)
}

func (c *CachingKeyValueStoreG[K, V]) BuildKeyMeta(ctx context.Context, kms map[string][]txn_data.KeyMaping) error {
	return c.wrappedStore.BuildKeyMeta(ctx, kms)
}

func (c *CachingKeyValueStoreG[K, V]) SetInstanceId(id uint8) {
	c.wrappedStore.SetInstanceId(id)
}

func (c *CachingKeyValueStoreG[K, V]) GetInstanceId() uint8 {
	return c.wrappedStore.GetInstanceId()
}
