package execution

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_with_changelog"
	"sharedlog-stream/pkg/stream_task"
)

/*
func SetupTableTableJoin[K, VLeft, VRight any](
	mpLeft *store_with_changelog.MaterializeParam[K, VLeft],
	mpRight *store_with_changelog.MaterializeParam[K, VRight],
	compare store.KVStoreLessFunc,
	joiner processor.ValueJoinerWithKeyFunc,
) (proc_interface.ProcessAndReturnFunc,
	proc_interface.ProcessAndReturnFunc,
	[]store.KeyValueStoreOpWithChangelog,
	error,
) {
	toLeftTab, leftTab, err := store_with_changelog.ToInMemKVTableWithChangelog(mpLeft, compare)
	if err != nil {
		return nil, nil, nil, err
	}
	toRightTab, rightTab, err := store_with_changelog.ToInMemKVTableWithChangelog(mpRight, compare)
	if err != nil {
		return nil, nil, nil, err
	}

	leftJoinRight := processor.NewMeteredProcessor(
		processor.NewTableTableJoinProcessor("leftJoinRight", rightTab, joiner))
	rightJoinLeft := processor.NewMeteredProcessor(
		processor.NewTableTableJoinProcessor("rightJoinLeft", leftTab,
			processor.ReverseValueJoinerWithKey(joiner)))
	leftJoinRightFunc := func(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
		ret, err := toLeftTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		if ret != nil {
			return leftJoinRight.ProcessAndReturn(ctx, ret[0])
		} else {
			return nil, nil
		}
	}
	rightJoinLeftFunc := func(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
		ret, err := toRightTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		if ret != nil {
			return rightJoinLeft.ProcessAndReturn(ctx, ret[0])
		} else {
			return nil, nil
		}
	}
	kvc := []store.KeyValueStoreOpWithChangelog{leftTab, rightTab}
	return leftJoinRightFunc, rightJoinLeftFunc, kvc, nil
}
*/

func ToInMemSkipmapKVTable[K comparable, V any](
	storeName string,
	less store.LessFunc[K],
	parNum uint8,
	serdeFormat commtypes.SerdeFormat,
	msgSerde commtypes.MessageGSerdeG[K, commtypes.ValueTimestampG[V]],
	gua exactly_once_intr.GuaranteeMth,
) (*processor.MeteredProcessorG[K, V, K, commtypes.ChangeG[V]],
	store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[V]], error,
) {
	s, err := GetInMemorySkipMapKVStore(
		&KVStoreParam[K, V]{
			Compare: less,
			CommonStoreParam: CommonStoreParam[K, V]{
				StoreName: storeName,
			},
		}, parNum, serdeFormat, msgSerde)
	if err != nil {
		return nil, nil, err
	}
	toTableProc := processor.NewMeteredProcessorG[K, V, K, commtypes.ChangeG[V]](
		processor.NewTableSourceProcessorWithTableG[K, V](s))
	return toTableProc, s, nil
}

func SetupTableTableJoinWithSkipmap[K comparable, VLeft, VRight, VR any](
	mpLeft *store_with_changelog.MaterializeParam[K, commtypes.ValueTimestampG[VLeft]],
	mpRight *store_with_changelog.MaterializeParam[K, commtypes.ValueTimestampG[VRight]],
	less store.LessFunc[K],
	joiner processor.ValueJoinerWithKeyFuncG[K, VLeft, VRight, VR],
	gua exactly_once_intr.GuaranteeMth,
) (
	leftJoinRightFunc proc_interface.ProcessAndReturnFunc[K, VLeft, K, commtypes.ChangeG[VR]],
	rightJoinLeftFunc proc_interface.ProcessAndReturnFunc[K, VRight, K, commtypes.ChangeG[VR]],
	kvos *store.KVStoreOps,
	setupSnapFunc stream_task.SetupSnapshotCallbackFunc,
	err error,
) {
	var toLeftTab *processor.MeteredProcessorG[K, VLeft, K, commtypes.ChangeG[VLeft]]
	var toRightTab *processor.MeteredProcessorG[K, VRight, K, commtypes.ChangeG[VRight]]
	var leftTab store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[VLeft]]
	var rightTab store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[VRight]]

	if gua == exactly_once_intr.ALIGN_CHKPT || gua == exactly_once_intr.NO_GUARANTEE {
		toLeftTab, leftTab, err = ToInMemSkipmapKVTable(mpLeft.StoreName(), less,
			mpLeft.ParNum(), mpLeft.SerdeFormat(), mpLeft.MessageSerde(), gua)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		debug.Assert(leftTab.GetInstanceId() == mpLeft.ParNum(), "instance id should equal to par num")
		debug.Assert(rightTab.GetInstanceId() == mpRight.ParNum(), "instance id should equal to par num")
		toRightTab, rightTab, err = ToInMemSkipmapKVTable(mpRight.StoreName(), less,
			mpRight.ParNum(), mpRight.SerdeFormat(), mpRight.MessageSerde(), gua)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		kvos = &store.KVStoreOps{
			Kvo: []store.KeyValueStoreOp{leftTab, rightTab},
		}
		setupSnapFunc = stream_task.SetupSnapshotCallbackFunc(
			func(ctx context.Context,
				serdeFormat commtypes.SerdeFormat,
				rs snapshot_store.SnapshotStore,
			) error {
				chkptSerde, err := commtypes.GetCheckpointSerdeG(serdeFormat)
				if err != nil {
					return err
				}
				stream_task.SetKVStoreChkpt[K, commtypes.ValueTimestampG[VLeft]](ctx,
					rs.(*snapshot_store.RedisSnapshotStore), leftTab, chkptSerde)
				stream_task.SetKVStoreChkpt[K, commtypes.ValueTimestampG[VRight]](ctx,
					rs.(*snapshot_store.RedisSnapshotStore), rightTab, chkptSerde)
				return nil
			})
	} else {
		var leftTabCl *store_with_changelog.KeyValueStoreWithChangelogG[K, commtypes.ValueTimestampG[VLeft]]
		var rightTabCl *store_with_changelog.KeyValueStoreWithChangelogG[K, commtypes.ValueTimestampG[VRight]]
		toLeftTab, leftTabCl, err = store_with_changelog.ToInMemSkipmapKVTableWithChangelog(mpLeft, less)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		toRightTab, rightTabCl, err = store_with_changelog.ToInMemSkipmapKVTableWithChangelog(mpRight, less)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		kvos = &store.KVStoreOps{
			Kvc: map[string]store.KeyValueStoreOpWithChangelog{
				leftTabCl.ChangelogTopicName():  leftTabCl,
				rightTabCl.ChangelogTopicName(): rightTabCl,
			},
		}
		setupSnapFunc = stream_task.SetupSnapshotCallbackFunc(func(ctx context.Context, serdeFormat commtypes.SerdeFormat,
			rs snapshot_store.SnapshotStore,
		) error {
			payloadSerde, err := commtypes.GetPayloadArrSerdeG(serdeFormat)
			if err != nil {
				return err
			}
			stream_task.SetKVStoreWithChangelogSnapshot[K, commtypes.ValueTimestampG[VLeft]](
				ctx, rs.(*snapshot_store.RedisSnapshotStore), leftTabCl, payloadSerde)
			stream_task.SetKVStoreWithChangelogSnapshot[K, commtypes.ValueTimestampG[VRight]](
				ctx, rs.(*snapshot_store.RedisSnapshotStore), rightTabCl, payloadSerde)
			return nil
		})
		leftTab = leftTabCl
		rightTab = rightTabCl
	}

	leftJoinRight := processor.NewMeteredProcessorG(
		processor.NewTableTableJoinProcessorG[K, VLeft, VRight, VR]("leftJoinRight", rightTab, joiner))
	rightJoinLeft := processor.NewMeteredProcessorG(
		processor.NewTableTableJoinProcessorG[K, VRight, VLeft, VR]("rightJoinLeft", leftTab,
			processor.ReverseValueJoinerWithKeyG(joiner)))
	nullKeyFilter := processor.NewStreamFilterProcessorG[K, commtypes.ChangeG[VR]]("filterNullKey",
		processor.PredicateFuncG[K, commtypes.ChangeG[VR]](func(key optional.Option[K], value optional.Option[commtypes.ChangeG[VR]]) (bool, error) {
			return key.IsSome(), nil
		}))
	leftJoinRightFunc = func(ctx context.Context, msg commtypes.MessageG[K, VLeft]) ([]commtypes.MessageG[K, commtypes.ChangeG[VR]], error) {
		ret, err := toLeftTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		if ret != nil {
			msgs, err := leftJoinRight.ProcessAndReturn(ctx, ret[0])
			if err != nil {
				return nil, err
			}
			if msgs != nil {
				return nullKeyFilter.ProcessAndReturn(ctx, msgs[0])
			}
		}
		return nil, nil
	}
	rightJoinLeftFunc = func(ctx context.Context, msg commtypes.MessageG[K, VRight]) ([]commtypes.MessageG[K, commtypes.ChangeG[VR]], error) {
		ret, err := toRightTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		if ret != nil {
			msgs, err := rightJoinLeft.ProcessAndReturn(ctx, ret[0])
			if err != nil {
				return nil, err
			}
			if msgs != nil {
				return nullKeyFilter.ProcessAndReturn(ctx, msgs[0])
			}
		}
		return nil, nil
	}
	return leftJoinRightFunc, rightJoinLeftFunc, kvos, setupSnapFunc, nil
}
