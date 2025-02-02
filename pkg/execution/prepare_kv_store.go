package execution

import (
	"context"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_with_changelog"
	"sharedlog-stream/pkg/stream_task"
)

type CommonStoreParam[K comparable, V any] struct {
	StoreName     string
	SizeOfK       func(K) int64
	SizeOfV       func(V) int64
	MaxCacheBytes int64
	UseCache      bool
	GuaranteeMth  exactly_once_intr.GuaranteeMth
}

type KVStoreParam[K comparable, V any] struct {
	Compare store.LessFunc[K]
	CommonStoreParam[K, V]
}

func GetInMemorySkipMapKVStore[K comparable, V, VStore any](
	p *KVStoreParam[K, V],
	parNum uint8,
	serdeFormat commtypes.SerdeFormat,
	msgSerde commtypes.MessageGSerdeG[K, VStore],
) (*store.InMemorySkipmapKeyValueStoreG[K, VStore], error) {
	st := store.NewInMemorySkipmapKeyValueStoreG[K, VStore](
		p.StoreName, p.Compare)
	st.SetInstanceId(parNum)
	err := st.SetKVSerde(serdeFormat,
		msgSerde.GetKeySerdeG(), msgSerde.GetValSerdeG())
	if err != nil {
		return nil, err
	}
	return st, nil
}

func GetKVStore[K comparable, V any](
	ctx context.Context,
	p *KVStoreParam[K, V],
	mp *store_with_changelog.MaterializeParam[K, commtypes.ValueTimestampG[V]],
) (
	cachedStore store.CachedKeyValueStore[K, commtypes.ValueTimestampG[V]],
	kvos *store.KVStoreOps,
	f stream_task.SetupSnapshotCallbackFunc,
	err error,
) {
	if p.GuaranteeMth == exactly_once_intr.ALIGN_CHKPT || p.GuaranteeMth == exactly_once_intr.NO_GUARANTEE {
		st, err := GetInMemorySkipMapKVStore(
			p, mp.ParNum(), mp.SerdeFormat(), mp.MessageSerde())
		if err != nil {
			return nil, nil, nil, err
		}
		if p.UseCache {
			sizeOfVTs := commtypes.ValueTimestampGSize[V]{
				ValSizeFunc: p.SizeOfV,
			}
			cachedStore = store.NewCachingKeyValueStore[K, commtypes.ValueTimestampG[V]](
				ctx, st, p.SizeOfK, sizeOfVTs.SizeOfValueTimestamp, p.MaxCacheBytes)
		} else {
			cachedStore = st
		}
		f = func(ctx context.Context, serdeFormat commtypes.SerdeFormat,
			mc snapshot_store.SnapshotStore,
		) error {
			chkptSerde, err := commtypes.GetCheckpointSerdeG(serdeFormat)
			if err != nil {
				return err
			}
			stream_task.SetKVStoreChkpt[K, commtypes.ValueTimestampG[V]](
				ctx,
				mc.(*snapshot_store.RedisSnapshotStore), cachedStore, chkptSerde)
			return nil
		}
		kvos = &store.KVStoreOps{
			Kvo: []store.KeyValueStoreOp{cachedStore},
			Kvc: nil,
		}
	} else {
		var aggStore store.CachedKeyValueStoreBackedByChangelogG[K, commtypes.ValueTimestampG[V]]
		kvstore, err := store_with_changelog.CreateInMemorySkipmapKVTableWithChangelogG(mp, p.Compare)
		if err != nil {
			return nil, nil, nil, err
		}
		if p.UseCache {
			sizeOfVTs := commtypes.ValueTimestampGSize[V]{
				ValSizeFunc: p.SizeOfV,
			}
			aggStore = store.NewCachingKeyValueStoreBackedByChangelogG[K, commtypes.ValueTimestampG[V]](
				ctx, kvstore, p.SizeOfK, sizeOfVTs.SizeOfValueTimestamp, p.MaxCacheBytes)
		} else {
			aggStore = kvstore
		}
		cachedStore = aggStore
		kvos = &store.KVStoreOps{
			Kvc: map[string]store.KeyValueStoreOpWithChangelog{
				aggStore.ChangelogTopicName(): aggStore,
			},
			Kvo: nil,
		}
		f = func(ctx context.Context, serdeFormat commtypes.SerdeFormat,
			rs snapshot_store.SnapshotStore,
		) error {
			payloadSerde, err := commtypes.GetPayloadArrSerdeG(serdeFormat)
			if err != nil {
				return err
			}
			stream_task.SetKVStoreWithChangelogSnapshot[K, commtypes.ValueTimestampG[V]](
				ctx,
				rs.(*snapshot_store.RedisSnapshotStore), aggStore, payloadSerde)
			return nil
		}
	}
	debug.Fprintf(os.Stderr, "cached store instance id: %d, parNum: %d\n", cachedStore.GetInstanceId(), mp.ParNum())
	debug.Assert(cachedStore.GetInstanceId() == mp.ParNum(), "instance id should be the same in mp")
	return cachedStore, kvos, f, nil
}

func StreamArgsSetKVStore(
	kvos *store.KVStoreOps,
	builder stream_task.BuildStreamTaskArgs,
	gua exactly_once_intr.GuaranteeMth,
) stream_task.BuildStreamTaskArgs {
	if gua == exactly_once_intr.ALIGN_CHKPT {
		return builder.KVStoreOps(kvos.Kvo)
	} else {
		return builder.KVStoreChangelogs(kvos.Kvc)
	}
}
