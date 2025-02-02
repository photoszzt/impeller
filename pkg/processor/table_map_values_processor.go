package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
)

/*
type TableMapValuesProcessor struct {
	valueMapperWithKey ValueMapperWithKey
	store              store.CoreKeyValueStore
	name               string
	BaseProcessor
}

func NewTableMapValuesProcessor(name string, mapper ValueMapperWithKey) Processor {
	p := &TableMapValuesProcessor{
		valueMapperWithKey: mapper,
		name:               name,
		store:              nil,
		BaseProcessor:      BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func NewTableMapValuesProcessorQueriable(name string, mapper ValueMapperWithKey, store store.CoreKeyValueStore) Processor {
	return &TableMapValuesProcessor{
		valueMapperWithKey: mapper,
		name:               name,
		store:              store,
	}
}

func (p *TableMapValuesProcessor) Name() string {
	return p.name
}

func (p *TableMapValuesProcessor) computeValue(key interface{}, value interface{}) (interface{}, error) {
	var newVal interface{} = nil
	var err error
	if value != nil {
		newVal, err = p.valueMapperWithKey.MapValue(key, value)
		if err != nil {
			return nil, err
		}
	}
	return newVal, nil
}

func (p *TableMapValuesProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	change, ok := msg.Value.(*commtypes.Change)
	if !ok {
		changeTmp := msg.Value.(commtypes.Change)
		change = &changeTmp
	}
	newV, err := p.computeValue(msg.Key, change.NewVal)
	if err != nil {
		return nil, err
	}
	oldV, err := p.computeValue(msg.Key, change.OldVal)
	if err != nil {
		return nil, err
	}
	if p.store != nil {
		err := p.store.Put(ctx, msg.Key, commtypes.CreateValueTimestamp(newV, msg.Timestamp))
		if err != nil {
			return nil, err
		}
	}
	newMsg := commtypes.Message{
		Key: msg.Key,
		Value: commtypes.Change{
			OldVal: oldV,
			NewVal: newV,
		},
		Timestamp: msg.Timestamp,
	}
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{newMsg}, nil
}
*/

type TableMapValuesProcessorG[K, V, VR any] struct {
	valueMapperWithKey ValueMapperWithKeyG[K, V, VR]
	name               string
	BaseProcessorG[K, commtypes.ChangeG[V], K, commtypes.ChangeG[VR]]
}

func NewTableMapValuesProcessorG[K, V, VR any](name string, mapper ValueMapperWithKeyG[K, V, VR]) ProcessorG[K, commtypes.ChangeG[V], K, commtypes.ChangeG[VR]] {
	p := &TableMapValuesProcessorG[K, V, VR]{
		valueMapperWithKey: mapper,
		name:               name,
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *TableMapValuesProcessorG[K, V, VR]) Name() string {
	return p.name
}

func (p *TableMapValuesProcessorG[K, V, VR]) computeValue(key optional.Option[K], value optional.Option[V]) (optional.Option[VR], error) {
	newVal := optional.None[VR]()
	var err error
	if value.IsSome() {
		newVal, err = p.valueMapperWithKey.MapValue(key, value)
		if err != nil {
			return newVal, err
		}
	}
	return newVal, nil
}

func (p *TableMapValuesProcessorG[K, V, VR]) ProcessAndReturn(ctx context.Context,
	msg commtypes.MessageG[K, commtypes.ChangeG[V]],
) ([]commtypes.MessageG[K, commtypes.ChangeG[VR]], error) {
	change := msg.Value.Unwrap()
	newV, err := p.computeValue(msg.Key, change.NewVal)
	if err != nil {
		return nil, err
	}
	oldV, err := p.computeValue(msg.Key, change.OldVal)
	if err != nil {
		return nil, err
	}
	newMsg := commtypes.MessageG[K, commtypes.ChangeG[VR]]{
		Key: msg.Key,
		Value: optional.Some(commtypes.ChangeG[VR]{
			OldVal: oldV,
			NewVal: newV,
		}),
		TimestampMs:   msg.TimestampMs,
		StartProcTime: msg.StartProcTime,
	}
	return []commtypes.MessageG[K, commtypes.ChangeG[VR]]{newMsg}, nil
}
