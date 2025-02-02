package execution

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_with_changelog"
	"sharedlog-stream/pkg/stream_task"
	"sharedlog-stream/pkg/utils"
)

/*
func SetupStreamStreamJoin[K, VLeft, VRight any](
	mpLeft *store_with_changelog.MaterializeParam[K, VLeft],
	mpRight *store_with_changelog.MaterializeParam[K, VRight],
	compare store.CompareFunc,
	joiner processor.ValueJoinerWithKeyTsFunc,
	jw *processor.JoinWindows,
) (proc_interface.ProcessAndReturnFunc,
	proc_interface.ProcessAndReturnFunc,
	map[string]store.WindowStoreOpWithChangelog,
	error,
) {
	toLeftTab, leftTab, err := store_with_changelog.ToInMemWindowTableWithChangelog(
		jw, true, compare, mpLeft)
	if err != nil {
		return nil, nil, nil, err
	}
	toRightTab, rightTab, err := store_with_changelog.ToInMemWindowTableWithChangelog(
		jw, true, compare, mpRight)
	if err != nil {
		return nil, nil, nil, err
	}
	sharedTimeTracker := processor.NewTimeTracker()
	leftJoinRight := processor.NewMeteredProcessor(
		processor.NewStreamStreamJoinProcessor("leftJoinRight", rightTab, jw, joiner, false, true, sharedTimeTracker),
	)
	rightJoinLeft := processor.NewMeteredProcessor(
		processor.NewStreamStreamJoinProcessor("rightJoinLeft", leftTab, jw,
			processor.ReverseValueJoinerWithKeyTs(joiner), false, false, sharedTimeTracker),
	)
	nullKeyFilter := processor.NewMeteredProcessor(processor.NewStreamFilterProcessor("filterNullKey",
		processor.PredicateFunc(func(key interface{}, value interface{}) (bool, error) {
			return !utils.IsNil(key), nil
		})))
	leftJoinRightFunc := func(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
		// debug.Fprintf(os.Stderr, "before toLeft\n")
		rets, err := toLeftTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		// debug.Fprintf(os.Stderr, "after toLeft\n")
		msgs, err := leftJoinRight.ProcessAndReturn(ctx, rets[0])
		// debug.Fprintf(os.Stderr, "after leftJoinRight\n")
		out := make([]commtypes.Message, 0, len(msgs))
		for _, msg := range msgs {
			ret, err := nullKeyFilter.ProcessAndReturn(ctx, msg)
			if err != nil {
				return nil, err
			}
			out = append(out, ret...)
		}
		return out, err
	}
	rightJoinLeftFunc := func(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
		// debug.Fprintf(os.Stderr, "before toRight\n")
		rets, err := toRightTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		// debug.Fprintf(os.Stderr, "after toRight\n")
		msgs, err := rightJoinLeft.ProcessAndReturn(ctx, rets[0])
		// debug.Fprintf(os.Stderr, "after rightJoinLeft\n")
		out := make([]commtypes.Message, 0, len(msgs))
		for _, msg := range msgs {
			ret, err := nullKeyFilter.ProcessAndReturn(ctx, msg)
			if err != nil {
				return nil, err
			}
			out = append(out, ret...)
		}
		return out, err
	}
	wsc := map[string]store.WindowStoreOpWithChangelog{
		leftTab.ChangelogTopicName():  leftTab,
		rightTab.ChangelogTopicName(): rightTab}
	return leftJoinRightFunc, rightJoinLeftFunc, wsc, nil
}
*/

func ToInMemSkipMapWindowTableG[K comparable, V any](
	p *WinStoreParam[K, V],
	parNum uint8,
	serdeFormat commtypes.SerdeFormat,
	msgSerde commtypes.MessageGSerdeG[K, V],
) (*processor.MeteredProcessorG[K, V, K, V], store.CoreWindowStoreG[K, V], error) {
	winTab, err := GetInMemorySkipMapWindowStore(p, parNum, serdeFormat, msgSerde)
	if err != nil {
		return nil, nil, err
	}
	toTableProc := processor.NewMeteredProcessorG[K, V, K, V](
		processor.NewStoreToWindowTableProcessorG[K, V](winTab))
	return toTableProc, winTab, nil
}

func SetupWinStoreWithChanglogSnap[K, VLeft, VRight any](
	leftTab store.WindowStoreBackedByChangelogG[K, VLeft],
	rightTab store.WindowStoreBackedByChangelogG[K, VRight],
) (
	*store.WinStoreOps,
	stream_task.SetupSnapshotCallbackFunc,
) {
	setupSnapFunc := stream_task.SetupSnapshotCallbackFunc(func(ctx context.Context,
		serdeFormat commtypes.SerdeFormat, rs snapshot_store.SnapshotStore,
	) error {
		payloadSerde, err := commtypes.GetPayloadArrSerdeG(serdeFormat)
		if err != nil {
			return err
		}
		stream_task.SetWinStoreWithChangelogSnapshot(ctx, rs.(*snapshot_store.RedisSnapshotStore), leftTab, payloadSerde)
		stream_task.SetWinStoreWithChangelogSnapshot(ctx, rs.(*snapshot_store.RedisSnapshotStore), rightTab, payloadSerde)
		return nil
	})
	return &store.WinStoreOps{
		Wsc: map[string]store.WindowStoreOpWithChangelog{
			leftTab.ChangelogTopicName():  leftTab,
			rightTab.ChangelogTopicName(): rightTab,
		},
	}, setupSnapFunc
}

func SetupWinStore[K, VLeft, VRight any](
	leftTab store.CoreWindowStoreG[K, VLeft],
	rightTab store.CoreWindowStoreG[K, VRight],
) (
	*store.WinStoreOps,
	stream_task.SetupSnapshotCallbackFunc,
) {
	setupSnapFunc := stream_task.SetupSnapshotCallbackFunc(func(ctx context.Context,
		serdeFormat commtypes.SerdeFormat, rs snapshot_store.SnapshotStore,
	) error {
		chkptSerde, err := commtypes.GetCheckpointSerdeG(serdeFormat)
		if err != nil {
			return err
		}
		stream_task.SetWinStoreChkpt(ctx, rs.(*snapshot_store.RedisSnapshotStore), leftTab, chkptSerde)
		stream_task.SetWinStoreChkpt(ctx, rs.(*snapshot_store.RedisSnapshotStore), rightTab, chkptSerde)
		return nil
	})
	return &store.WinStoreOps{
		Wsos: []store.WindowStoreOp{
			leftTab,
			rightTab,
		},
	}, setupSnapFunc
}

func SetupStreamStreamJoinG[K, VLeft, VRight, VR any](
	toLeftTab *processor.MeteredProcessorG[K, VLeft, K, VLeft],
	leftTab store.CoreWindowStoreG[K, VLeft],
	toRightTab *processor.MeteredProcessorG[K, VRight, K, VRight],
	rightTab store.CoreWindowStoreG[K, VRight],
	joiner processor.ValueJoinerWithKeyTsFuncG[K, VLeft, VRight, VR],
	jw *commtypes.JoinWindows,
) (leftJoinRightFunc proc_interface.ProcessAndReturnFunc[K, VLeft, K, VR],
	rightJoinLeftFunc proc_interface.ProcessAndReturnFunc[K, VRight, K, VR],
	err error,
) {
	sharedTimeTracker := processor.NewTimeTracker()
	leftJoinRight := processor.NewMeteredProcessorG[K, VLeft, K, VR](
		processor.NewStreamStreamJoinProcessorG[K, VLeft, VRight, VR]("leftJoinRight", rightTab, jw,
			joiner, false, true, sharedTimeTracker))
	rightJoinLeft := processor.NewMeteredProcessorG[K, VRight, K, VR](
		processor.NewStreamStreamJoinProcessorG[K, VRight, VLeft, VR]("rightJoinLeft", leftTab, jw,
			processor.ReverseValueJoinerWithKeyTsG(joiner), false, false, sharedTimeTracker))
	nullKeyFilter := processor.NewStreamFilterProcessorG[K, VR]("filterNullKey",
		processor.PredicateFuncG[K, VR](func(key optional.Option[K], value optional.Option[VR]) (bool, error) {
			return !utils.IsNil(key), nil
		}))
	leftJoinRightFunc = func(ctx context.Context, msg commtypes.MessageG[K, VLeft]) ([]commtypes.MessageG[K, VR], error) {
		// debug.Fprintf(os.Stderr, "before toLeft\n")
		rets, err := toLeftTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		// debug.Fprintf(os.Stderr, "after toLeft\n")
		msgs, err := leftJoinRight.ProcessAndReturn(ctx, rets[0])
		// debug.Fprintf(os.Stderr, "after leftJoinRight\n")
		out := make([]commtypes.MessageG[K, VR], 0, len(msgs))
		for _, msg := range msgs {
			ret, err := nullKeyFilter.ProcessAndReturn(ctx, msg)
			if err != nil {
				return nil, err
			}
			out = append(out, ret...)
		}
		return out, err
	}
	rightJoinLeftFunc = func(ctx context.Context, msg commtypes.MessageG[K, VRight]) ([]commtypes.MessageG[K, VR], error) {
		// debug.Fprintf(os.Stderr, "before toRight\n")
		rets, err := toRightTab.ProcessAndReturn(ctx, msg)
		if err != nil {
			return nil, err
		}
		// debug.Fprintf(os.Stderr, "after toRight\n")
		msgs, err := rightJoinLeft.ProcessAndReturn(ctx, rets[0])
		// debug.Fprintf(os.Stderr, "after rightJoinLeft\n")
		out := make([]commtypes.MessageG[K, VR], 0, len(msgs))
		for _, msg := range msgs {
			ret, err := nullKeyFilter.ProcessAndReturn(ctx, msg)
			if err != nil {
				return nil, err
			}
			out = append(out, ret...)
		}
		return out, err
	}
	return leftJoinRightFunc, rightJoinLeftFunc, nil
}

func winStoreParamForJoin[K comparable, V any](
	name string,
	compare store.CompareFuncG[K],
	jw *commtypes.JoinWindows,
	gua exactly_once_intr.GuaranteeMth,
) *WinStoreParam[K, V] {
	return &WinStoreParam[K, V]{
		CmpFunc:          compare,
		JoinWindow:       jw,
		RetainDuplicates: true,
		CommonStoreParam: CommonStoreParam[K, V]{
			StoreName:     name,
			SizeOfK:       nil,
			SizeOfV:       nil,
			UseCache:      false,
			MaxCacheBytes: 0,
			GuaranteeMth:  gua,
		},
	}
}

func SetupSkipMapStreamStreamJoin[K comparable, VLeft, VRight, VR any](
	mpLeft *store_with_changelog.MaterializeParam[K, VLeft],
	mpRight *store_with_changelog.MaterializeParam[K, VRight],
	compare store.CompareFuncG[K],
	joiner processor.ValueJoinerWithKeyTsFuncG[K, VLeft, VRight, VR],
	jw *commtypes.JoinWindows,
	gua exactly_once_intr.GuaranteeMth,
) (proc_interface.ProcessAndReturnFunc[K, VLeft, K, VR],
	proc_interface.ProcessAndReturnFunc[K, VRight, K, VR],
	*store.WinStoreOps,
	stream_task.SetupSnapshotCallbackFunc,
	error,
) {
	var leftTab store.CoreWindowStoreG[K, VLeft]
	var rightTab store.CoreWindowStoreG[K, VRight]
	var toLeftTab *processor.MeteredProcessorG[K, VLeft, K, VLeft]
	var toRightTab *processor.MeteredProcessorG[K, VRight, K, VRight]
	var setupSnapFunc stream_task.SetupSnapshotCallbackFunc
	var err error
	var wsos *store.WinStoreOps
	if gua == exactly_once_intr.ALIGN_CHKPT || gua == exactly_once_intr.NO_GUARANTEE {
		toLeftTab, leftTab, err = ToInMemSkipMapWindowTableG(
			winStoreParamForJoin[K, VLeft](mpLeft.StoreName(), compare, jw, gua),
			mpLeft.ParNum(), mpLeft.SerdeFormat(), mpLeft.MessageSerde())
		if err != nil {
			return nil, nil, nil, nil, err
		}
		toRightTab, rightTab, err = ToInMemSkipMapWindowTableG(
			winStoreParamForJoin[K, VRight](mpRight.StoreName(), compare, jw, gua),
			mpRight.ParNum(), mpRight.SerdeFormat(), mpRight.MessageSerde())
		if err != nil {
			return nil, nil, nil, nil, err
		}
		wsos, setupSnapFunc = SetupWinStore(leftTab, rightTab)
	} else {
		var leftTabCl *store_with_changelog.InMemoryWindowStoreWithChangelogG[K, VLeft]
		var rightTabCl *store_with_changelog.InMemoryWindowStoreWithChangelogG[K, VRight]
		toLeftTab, leftTabCl, err = store_with_changelog.ToInMemSkipMapWindowTableWithChangelogG(
			jw, true, compare, mpLeft)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		toRightTab, rightTabCl, err = store_with_changelog.ToInMemSkipMapWindowTableWithChangelogG(
			jw, true, compare, mpRight)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		wsos, setupSnapFunc = SetupWinStoreWithChanglogSnap[K, VLeft, VRight](leftTabCl, rightTabCl)
		leftTab = leftTabCl
		rightTab = rightTabCl
	}
	lJoinR, rJoinL, err := SetupStreamStreamJoinG[K, VLeft, VRight](
		toLeftTab, leftTab, toRightTab, rightTab, joiner, jw)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return lJoinR, rJoinL, wsos, setupSnapFunc, nil
}
