package store

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils"

	"github.com/rs/zerolog/log"
	"github.com/zhangyunhao116/skipmap"
	"golang.org/x/sync/errgroup"
)

type (
	KVSnapshotCallback[K, V any] func(ctx context.Context,
		tpLogOff []commtypes.TpLogOff,
		chkptMeta []commtypes.ChkptMetaData,
		snapshot []*commtypes.KeyValuePair[K, V]) error
)

type InMemorySkipmapKeyValueStoreG[K, V any] struct {
	store            *skipmap.FuncMap[K, V]
	bgCtx            context.Context
	bgErrG           *errgroup.Group
	kvPairSerde      commtypes.SerdeG[*commtypes.KeyValuePair[K, V]]
	keySerde         commtypes.SerdeG[K]
	snapshotCallback KVSnapshotCallback[K, V]
	name             string
	insId            uint8
}

var (
	_ = CoreKeyValueStoreG[int, int](&InMemorySkipmapKeyValueStoreG[int, int]{})
	_ = CachedKeyValueStore[int, int](&InMemorySkipmapKeyValueStoreG[int, int]{})
)

func NewInMemorySkipmapKeyValueStoreG[K, V any](name string, lessFunc LessFunc[K]) *InMemorySkipmapKeyValueStoreG[K, V] {
	return &InMemorySkipmapKeyValueStoreG[K, V]{
		name:             name,
		store:            skipmap.NewFunc[K, V](lessFunc),
		snapshotCallback: nil,
	}
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) SetSnapshotCallback(ctx context.Context, f KVSnapshotCallback[K, V]) {
	st.bgErrG, st.bgCtx = errgroup.WithContext(ctx)
	st.snapshotCallback = f
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) WaitForAllSnapshot() error {
	debug.Assert(st.bgErrG != nil, "snapshot callback is not set")
	return st.bgErrG.Wait()
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat, keySerde commtypes.SerdeG[K], valSerde commtypes.SerdeG[V]) error {
	var err error
	st.keySerde = keySerde
	st.kvPairSerde, err = commtypes.GetKeyValuePairSerdeG(serdeFormat, keySerde, valSerde)
	return err
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[K, V]] {
	return st.kvPairSerde
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) Name() string {
	return st.name
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	ret, exists := st.store.Load(key)
	return ret, exists, nil
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) Put(ctx context.Context, key K, value optional.Option[V], tm TimeMeta) error {
	v, ok := value.Take()
	if !ok {
		st.store.Delete(key)
	} else {
		st.store.Store(key, v)
	}
	return nil
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) PutIfAbsent(ctx context.Context, key K, value V, tm TimeMeta) (optional.Option[V], error) {
	val, loaded := st.store.LoadOrStore(key, value)
	if loaded {
		return optional.Some(val), nil
	} else {
		return optional.None[V](), nil
	}
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error {
	if utils.IsNil(value) {
		return st.Put(ctx, key.(K), optional.None[V](), TimeMeta{RecordTsMs: 0})
	} else {
		return st.Put(ctx, key.(K), optional.Some(value.(V)), TimeMeta{RecordTsMs: 0})
	}
}

/*
func (st *InMemorySkipmapKeyValueStoreG[K, V]) PutAll(ctx context.Context, kvs []*commtypes.Message) error {
	maxTs := int64(0)
	for _, kv := range kvs {
		if kv.Timestamp > maxTs {
			maxTs = kv.Timestamp
		}
		var err error
		if utils.IsNil(kv.Value) {
			err = st.Put(ctx, kv.Key.(K), optional.None[V](), maxTs)
		} else {
			err = st.Put(ctx, kv.Key.(K), optional.Some(kv.Value.(V)), maxTs)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
*/

func (st *InMemorySkipmapKeyValueStoreG[K, V]) Delete(ctx context.Context, key K) error {
	st.store.Delete(key)
	return nil
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) ApproximateNumEntries() (uint64, error) {
	return uint64(st.store.Len()), nil
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) Range(ctx context.Context,
	from optional.Option[K], to optional.Option[K],
	iterFunc func(K, V) error,
) error {
	f, okF := from.Take()
	t, okT := to.Take()
	_ = f
	_ = t
	if !okF && !okT {
		st.store.Range(func(key K, value V) bool {
			err := iterFunc(key, value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[Error] Range: %v\n", err)
				return false
			}
			return true
		})
	} else if !okF && okT {
		panic("not implemented")
	} else if okF && !okT {
		panic("not implemented")
	} else {
		panic("not implemented")
	}
	return nil
}

// not thread-safe
func (st *InMemorySkipmapKeyValueStoreG[K, V]) Snapshot(
	ctx context.Context,
	tpLogOff []commtypes.TpLogOff, meta []commtypes.ChkptMetaData,
	resetBg bool,
) {
	debug.Assert(st.bgErrG != nil, "snapshot callback is not set")
	// cpyBeg := time.Now()
	out := make([]*commtypes.KeyValuePair[K, V], 0, st.store.Len())
	st.store.Range(func(key K, value V) bool {
		p := &commtypes.KeyValuePair[K, V]{
			Key:   key,
			Value: value,
		}
		out = append(out, p)
		return true
	})
	if meta != nil || resetBg {
		st.bgErrG, st.bgCtx = errgroup.WithContext(ctx)
	}
	st.bgErrG.Go(func() error {
		return st.snapshotCallback(st.bgCtx, tpLogOff, meta, out)
	})
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) RestoreFromSnapshot(snapshot [][]byte) error {
	for _, kv := range snapshot {
		p, err := st.kvPairSerde.Decode(kv)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decode key-value pair")
			return err
		}
		st.store.Store(p.Key, p.Value)
	}
	return nil
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) TableType() TABLE_TYPE {
	return IN_MEM
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) Flush(ctx context.Context) (uint32, error) {
	return 0, nil
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) SetCacheFlushCallback(
	KVStoreCacheFlushCallbackFunc[K, V]) {
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) BuildKeyMeta(ctx context.Context, kms map[string][]txn_data.KeyMaping) error {
	kms[st.Name()] = make([]txn_data.KeyMaping, 0)
	hasher := hashfuncs.ByteSliceHasher{}
	return st.Range(ctx, optional.None[K](), optional.None[K](), func(k K, v V) error {
		kBytes, _, err := st.keySerde.Encode(k)
		if err != nil {
			return err
		}
		hash := hasher.HashSum64(kBytes)
		kms[st.Name()] = append(kms[st.Name()], txn_data.KeyMaping{
			Key:         kBytes,
			Hash:        hash,
			SubstreamId: st.insId,
		})
		return nil
	})
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) SetInstanceId(id uint8) {
	st.insId = id
}

func (st *InMemorySkipmapKeyValueStoreG[K, V]) GetInstanceId() uint8 {
	return st.insId
}
