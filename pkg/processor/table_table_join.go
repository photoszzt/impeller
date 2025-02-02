package processor

import (
	"context"
	"fmt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/store"

	"github.com/rs/zerolog/log"
)

/*
type TableTableJoinProcessor struct {
	store             store.CoreKeyValueStore
	joiner            ValueJoinerWithKey
	streamTimeTracker commtypes.StreamTimeTracker
	name              string
	BaseProcessor
}

var _ = Processor(&TableTableJoinProcessor{})

func NewTableTableJoinProcessor(name string, store store.CoreKeyValueStore, joiner ValueJoinerWithKey) *TableTableJoinProcessor {
	p := &TableTableJoinProcessor{
		joiner:            joiner,
		store:             store,
		streamTimeTracker: commtypes.NewStreamTimeTracker(),
		name:              name,
		BaseProcessor:     BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *TableTableJoinProcessor) Name() string {
	return p.name
}

func (p *TableTableJoinProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Key == nil {
		log.Warn().Msgf("Skipping record due to null join key. key=%v", msg.Key)
		return nil, nil
	}
	val2, ok, err := p.store.Get(ctx, msg.Key)
	if err != nil {
		return nil, fmt.Errorf("get err: %v", err)
	}
	if ok {
		rvTs := val2.(*commtypes.ValueTimestamp)
		if rvTs.Value == nil {
			return nil, nil
		}
		ts := msg.Timestamp
		if ts < rvTs.Timestamp {
			ts = rvTs.Timestamp
		}
		var newVal interface{}
		var oldVal interface{}
		change := commtypes.CastToChangePtr(msg.Value)
		if change.NewVal != nil {
			newVal = p.joiner.Apply(msg.Key, change.NewVal, rvTs.Value)
		}
		if change.OldVal != nil {
			oldVal = p.joiner.Apply(msg.Key, change.OldVal, rvTs.Value)
		}
		return []commtypes.Message{{Key: msg.Key, Value: commtypes.Change{NewVal: newVal, OldVal: oldVal}, Timestamp: ts}}, nil
	}
	return nil, nil
}
*/

type TableTableJoinProcessorG[K, V1, V2, VR any] struct {
	store             store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[V2]]
	joiner            ValueJoinerWithKeyG[K, V1, V2, VR]
	streamTimeTracker commtypes.StreamTimeTracker
	name              string
	BaseProcessorG[K, commtypes.ChangeG[V1], K, commtypes.ChangeG[VR]]
}

var _ = ProcessorG[int, commtypes.ChangeG[string], int, commtypes.ChangeG[string]](&TableTableJoinProcessorG[int, string, string, string]{})

func NewTableTableJoinProcessorG[K, V1, V2, VR any](name string, store store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[V2]],
	joiner ValueJoinerWithKeyG[K, V1, V2, VR]) ProcessorG[K, commtypes.ChangeG[V1], K, commtypes.ChangeG[VR]] {
	p := &TableTableJoinProcessorG[K, V1, V2, VR]{
		joiner:            joiner,
		store:             store,
		streamTimeTracker: commtypes.NewStreamTimeTracker(),
		name:              name,
		BaseProcessorG:    BaseProcessorG[K, commtypes.ChangeG[V1], K, commtypes.ChangeG[VR]]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *TableTableJoinProcessorG[K, V1, V2, VR]) Name() string {
	return p.name
}

func (p *TableTableJoinProcessorG[K, V1, V2, VR]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, commtypes.ChangeG[V1]]) (
	[]commtypes.MessageG[K, commtypes.ChangeG[VR]], error,
) {
	if msg.Key.IsNone() {
		log.Warn().Msgf("Skipping record due to null join key. key=%v", msg.Key)
		return nil, nil
	}
	key := msg.Key.Unwrap()
	rvTs, ok, err := p.store.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get err: %v", err)
	}
	if ok {
		ts := msg.TimestampMs
		if ts < rvTs.Timestamp {
			ts = rvTs.Timestamp
		}
		newVal := optional.None[VR]()
		oldVal := optional.None[VR]()
		change := msg.Value.Unwrap()
		msgNewVal, hasMsgNewVal := change.NewVal.Take()
		if hasMsgNewVal {
			newVal = p.joiner.Apply(key, msgNewVal, rvTs.Value)
		}
		msgOldVal, hasMsgOldVal := change.OldVal.Take()
		if hasMsgOldVal {
			oldVal = p.joiner.Apply(key, msgOldVal, rvTs.Value)
		}
		return []commtypes.MessageG[K, commtypes.ChangeG[VR]]{{
			Key:           msg.Key,
			Value:         optional.Some(commtypes.ChangeG[VR]{NewVal: newVal, OldVal: oldVal}),
			TimestampMs:   ts,
			StartProcTime: msg.StartProcTime}}, nil
	}
	return nil, nil
}
