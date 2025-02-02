package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
)

/*
type TableToStreamProcessor struct {
	valueMapperWithKey ValueMapperWithKey
	name               string
	BaseProcessor
}

var _ = Processor(&TableToStreamProcessor{})

func NewTableToStreamProcessor() *TableToStreamProcessor {
	p := &TableToStreamProcessor{
		name: "toStream",
		valueMapperWithKey: ValueMapperWithKeyFunc(func(key, value interface{}) (interface{}, error) {
			val := commtypes.CastToChangePtr(value)
			return val.NewVal, nil
		}),
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *TableToStreamProcessor) Name() string {
	return p.name
}

func (p *TableToStreamProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	newV, err := p.valueMapperWithKey.MapValue(ctx, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{{Key: msg.Key, Value: newV, Timestamp: msg.Timestamp}}, nil
}
*/

type TableToStreamProcessorG[K, V any] struct {
	valueMapperWithKey ValueMapperWithKeyG[K, commtypes.ChangeG[V], V]
	name               string
	BaseProcessorG[K, commtypes.ChangeG[V], K, V]
}

func NewTableToStreamProcessorG[K, V any]() ProcessorG[K, commtypes.ChangeG[V], K, V] {
	p := &TableToStreamProcessorG[K, V]{
		name: "toStream",
		valueMapperWithKey: ValueMapperWithKeyFuncG[K, commtypes.ChangeG[V], V](func(key optional.Option[K], value optional.Option[commtypes.ChangeG[V]]) (optional.Option[V], error) {
			c := value.Unwrap()
			return c.NewVal, nil
		}),
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *TableToStreamProcessorG[K, V]) Name() string {
	return p.name
}

func (p *TableToStreamProcessorG[K, V]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, commtypes.ChangeG[V]]) ([]commtypes.MessageG[K, V], error) {
	newV, err := p.valueMapperWithKey.MapValue(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.MessageG[K, V]{{Key: msg.Key, Value: newV,
		TimestampMs: msg.TimestampMs, StartProcTime: msg.StartProcTime}}, nil
}
