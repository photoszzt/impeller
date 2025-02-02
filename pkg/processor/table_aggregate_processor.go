package processor

import (
	"context"
	"fmt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/utils"
)

/*
type TableAggregateProcessor struct {
	store       store.CoreKeyValueStore
	initializer Initializer
	add         Aggregator
	remove      Aggregator
	name        string
	BaseProcessor
}

var _ = Processor(&TableAggregateProcessor{})

func NewTableAggregateProcessor(name string,
	store store.CoreKeyValueStore,
	initializer Initializer,
	add Aggregator,
	remove Aggregator,
) *TableAggregateProcessor {
	p := &TableAggregateProcessor{
		initializer:   initializer,
		add:           add,
		remove:        remove,
		name:          name,
		store:         store,
		BaseProcessor: BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *TableAggregateProcessor) Name() string {
	return p.name
}

func (p *TableAggregateProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if utils.IsNil(msg.Key) {
		return nil, fmt.Errorf("msg key for table aggregate should not be nil")
	}
	val, ok, err := p.store.Get(ctx, msg.Key)
	if err != nil {
		return nil, err
	}
	var oldAggTs *commtypes.ValueTimestamp = nil
	var oldAgg interface{} = nil
	if ok {
		oldAggTs = val.(*commtypes.ValueTimestamp)
		oldAgg = oldAggTs.Value
	}
	newTs := msg.Timestamp
	var intermediateAgg interface{}

	// first try to remove the old val
	msgVal := msg.Value.(commtypes.Change)
	if !utils.IsNil(msgVal.OldVal) && !utils.IsNil(oldAgg) {
		intermediateAgg = p.remove.Apply(msg.Key, msgVal.OldVal, oldAgg)
		newTs = utils.MaxInt64(msg.Timestamp, oldAggTs.Timestamp)
	} else {
		intermediateAgg = oldAgg
	}

	// then try to add the new val
	var newAgg interface{}
	if !utils.IsNil(msgVal.NewVal) {
		var initAgg interface{}
		if utils.IsNil(intermediateAgg) {
			initAgg = p.initializer.Apply()
		} else {
			initAgg = intermediateAgg
		}
		newAgg = p.add.Apply(msg.Key, msgVal.NewVal, initAgg)
		if !utils.IsNil(oldAggTs) {
			newTs = utils.MaxInt64(msg.Timestamp, oldAggTs.Timestamp)
		}
	} else {
		newAgg = intermediateAgg
	}

	err = p.store.Put(ctx, msg.Key, commtypes.CreateValueTimestamp(newAgg, newTs))
	if err != nil {
		return nil, err
	}
	change := commtypes.Change{
		NewVal: newAgg,
		OldVal: oldAgg,
	}
	return []commtypes.Message{{Key: msg.Key, Value: change, Timestamp: newTs}}, nil
}
*/

type TableAggregateProcessorG[K, V, VA any] struct {
	store       store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[VA]]
	initializer InitializerG[VA]
	add         AggregatorG[K, V, VA]
	remove      AggregatorG[K, V, VA]
	name        string
	BaseProcessorG[K, commtypes.ChangeG[V], K, commtypes.ChangeG[VA]]
}

var _ = ProcessorG[int, commtypes.ChangeG[int], int, commtypes.ChangeG[int]](&TableAggregateProcessorG[int, int, int]{})

func NewTableAggregateProcessorG[K, V, VA any](name string,
	store store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[VA]],
	initializer InitializerG[VA],
	add AggregatorG[K, V, VA],
	remove AggregatorG[K, V, VA],
) ProcessorG[K, commtypes.ChangeG[V], K, commtypes.ChangeG[VA]] {
	p := &TableAggregateProcessorG[K, V, VA]{
		initializer: initializer,
		add:         add,
		remove:      remove,
		name:        name,
		store:       store,
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *TableAggregateProcessorG[K, V, VA]) Name() string {
	return p.name
}

func (p *TableAggregateProcessorG[K, V, VA]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, commtypes.ChangeG[V]]) (
	[]commtypes.MessageG[K, commtypes.ChangeG[VA]], error,
) {
	if msg.Key.IsNone() {
		return nil, fmt.Errorf("msg key for table aggregate should not be nil")
	}
	key := msg.Key.Unwrap()
	oldAggTs, ok, err := p.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	oldAgg := optional.None[VA]()
	if ok {
		oldAgg = optional.Some(oldAggTs.Value)
	}
	newTs := msg.TimestampMs
	var intermediateAgg optional.Option[VA]

	// first try to remove the old val
	msgVal := msg.Value.Unwrap()
	msgOldVal, hasMsgOldVal := msgVal.OldVal.Take()
	if hasMsgOldVal && oldAgg.IsSome() {
		intermediateAgg = p.remove.Apply(key, msgOldVal, oldAgg)
		newTs = utils.MaxInt64(msg.TimestampMs, oldAggTs.Timestamp)
	} else {
		intermediateAgg = oldAgg
	}

	// then try to add the new val
	var newAgg optional.Option[VA]
	msgNewVal, hasMsgNewVal := msgVal.NewVal.Take()
	if hasMsgNewVal {
		var initAgg optional.Option[VA]
		if intermediateAgg.IsSome() {
			initAgg = intermediateAgg
		} else {
			initAgg = p.initializer.Apply()
		}
		newAgg = p.add.Apply(key, msgNewVal, initAgg)
		if !utils.IsNil(oldAggTs) {
			newTs = utils.MaxInt64(msg.TimestampMs, oldAggTs.Timestamp)
		}
	} else {
		newAgg = intermediateAgg
	}

	err = p.store.Put(ctx, key, commtypes.CreateValueTimestampGOptional(newAgg, newTs),
		store.TimeMeta{RecordTsMs: newTs, StartProcTs: msg.StartProcTime})
	if err != nil {
		return nil, err
	}
	change := optional.Some(commtypes.ChangeG[VA]{
		NewVal: newAgg,
		OldVal: oldAgg,
	})
	return []commtypes.MessageG[K, commtypes.ChangeG[VA]]{{
		Key: msg.Key, Value: change,
		TimestampMs: newTs, StartProcTime: msg.StartProcTime,
	}}, nil
}
