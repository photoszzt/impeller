package execution

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_with_changelog"
	"sharedlog-stream/pkg/stream_task"
)

type WinStoreParam[K comparable, V any] struct {
	CmpFunc          store.CompareFuncG[K]
	JoinWindow       commtypes.EnumerableWindowDefinition
	RetainDuplicates bool
	CommonStoreParam[K, V]
}

func StreamArgsSetWinStore(
	wsos *store.WinStoreOps,
	builder stream_task.BuildStreamTaskArgs,
	gua exactly_once_intr.GuaranteeMth,
) stream_task.BuildStreamTaskArgs {
	if gua == exactly_once_intr.ALIGN_CHKPT {
		return builder.WindowStoreOps(wsos.Wsos)
	} else {
		return builder.WindowStoreChangelogs(wsos.Wsc)
	}
}

func GetInMemorySkipMapWindowStore[K comparable, V any, VStore any](
	p *WinStoreParam[K, V],
	parNum uint8,
	serdeFormat commtypes.SerdeFormat,
	msgSerde commtypes.MessageGSerdeG[K, VStore],
) (
	*store.InMemorySkipMapWindowStoreG[K, VStore], error,
) {
	cachedStore := store.NewInMemorySkipMapWindowStore[K, VStore](
		p.StoreName,
		p.JoinWindow.MaxSize()+p.JoinWindow.GracePeriodMs(),
		p.JoinWindow.MaxSize(),
		p.RetainDuplicates,
		p.CmpFunc)
	cachedStore.SetInstanceId(parNum)
	keyAndWindowStartTsSerde, err := commtypes.GetKeyAndWindowStartTsSerdeG(serdeFormat, msgSerde.GetKeySerdeG())
	if err != nil {
		return nil, err
	}
	err = cachedStore.SetKVSerde(serdeFormat,
		keyAndWindowStartTsSerde,
		msgSerde.GetKeySerdeG(),
		msgSerde.GetValSerdeG())
	if err != nil {
		return nil, err
	}
	return cachedStore, nil
}

func GetWinStore[K comparable, V any](
	ctx context.Context,
	p *WinStoreParam[K, V],
	mp *store_with_changelog.MaterializeParam[K, commtypes.ValueTimestampG[V]],
) (
	cachedStore store.CachedWindowStateStore[K, commtypes.ValueTimestampG[V]],
	wsos *store.WinStoreOps,
	f stream_task.SetupSnapshotCallbackFunc,
	err error,
) {
	if p.GuaranteeMth == exactly_once_intr.ALIGN_CHKPT || p.GuaranteeMth == exactly_once_intr.NO_GUARANTEE {
		countWindowStore, err := GetInMemorySkipMapWindowStore(p, mp.ParNum(), mp.SerdeFormat(), mp.MessageSerde())
		if err != nil {
			return nil, nil, nil, err
		}
		if p.UseCache {
			sizeOfVTs := commtypes.ValueTimestampGSize[V]{
				ValSizeFunc: p.SizeOfV,
			}
			sizeOfKeyTs := commtypes.KeyAndWindowStartTsGSize[K]{
				KeySizeFunc: p.SizeOfK,
			}
			cachedStore = store.NewCachingWindowStoreG[K, commtypes.ValueTimestampG[V]](
				ctx, p.JoinWindow.MaxSize(), countWindowStore,
				sizeOfKeyTs.SizeOfKeyAndWindowStartTs,
				sizeOfVTs.SizeOfValueTimestamp, p.MaxCacheBytes)
		} else {
			cachedStore = countWindowStore
		}
		f = stream_task.SetupSnapshotCallbackFunc(
			func(ctx context.Context,
				serdeFormat commtypes.SerdeFormat,
				rs snapshot_store.SnapshotStore,
			) error {
				chkptSerde, err := commtypes.GetCheckpointSerdeG(serdeFormat)
				if err != nil {
					return err
				}
				stream_task.SetWinStoreChkpt[K, commtypes.ValueTimestampG[V]](ctx,
					rs.(*snapshot_store.RedisSnapshotStore), cachedStore, chkptSerde)
				return nil
			})
		wsos = &store.WinStoreOps{
			Wsos: []store.WindowStoreOp{cachedStore},
			Wsc:  nil,
		}
	} else {
		countWindowStore, err := store_with_changelog.CreateInMemSkipMapWindowTableWithChangelogG(
			p.JoinWindow, false, p.CmpFunc, mp)
		if err != nil {
			return nil, nil, nil, err
		}
		var aggStore store.CachedWindowStoreBackedByChangelogG[K, commtypes.ValueTimestampG[V]]
		if p.UseCache {
			sizeOfVTs := commtypes.ValueTimestampGSize[V]{
				ValSizeFunc: p.SizeOfV,
			}
			sizeOfKeyTs := commtypes.KeyAndWindowStartTsGSize[K]{
				KeySizeFunc: p.SizeOfK,
			}
			aggStore = store.NewCachingWindowStoreBackedByChangelogG[K, commtypes.ValueTimestampG[V]](
				ctx, p.JoinWindow.MaxSize(), countWindowStore,
				sizeOfKeyTs.SizeOfKeyAndWindowStartTs,
				sizeOfVTs.SizeOfValueTimestamp, p.MaxCacheBytes)
		} else {
			aggStore = countWindowStore
		}
		cachedStore = aggStore
		wsos = &store.WinStoreOps{
			Wsc: map[string]store.WindowStoreOpWithChangelog{
				countWindowStore.ChangelogTopicName(): aggStore,
			},
			Wsos: nil,
		}
		f = stream_task.SetupSnapshotCallbackFunc(func(ctx context.Context,
			serdeFormat commtypes.SerdeFormat,
			rs snapshot_store.SnapshotStore,
		) error {
			payloadSerde, err := commtypes.GetPayloadArrSerdeG(serdeFormat)
			if err != nil {
				return err
			}
			stream_task.SetWinStoreWithChangelogSnapshot[K, commtypes.ValueTimestampG[V]](ctx,
				rs.(*snapshot_store.RedisSnapshotStore), aggStore, payloadSerde)
			return nil
		})
	}
	return cachedStore, wsos, f, nil
}
