package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
)

// type Mapper interface {
// 	Map(key, value interface{}) (interface{} /* key */, interface{} /* value */, error)
// }
// type MapperFunc func(key, value interface{}) (interface{} /* key */, interface{} /* value */, error)
//
// var _ = (Mapper)(MapperFunc(nil))
//
// func (fn MapperFunc) Map(key, value interface{}) (interface{} /* key */, interface{} /* value */, error) {
// 	return fn(key, value)
// }

type MapperG[K, V, KR, VR any] interface {
	Map(key optional.Option[K], value optional.Option[V]) (optional.Option[KR] /* key */, optional.Option[VR] /* value */, error)
}
type MapperFuncG[K, V, KR, VR any] func(key optional.Option[K], value optional.Option[V]) (optional.Option[KR] /* key */, optional.Option[VR] /* value */, error)

var _ = (MapperG[int, int, int, int])(MapperFuncG[int, int, int, int](nil))

func (fn MapperFuncG[K, V, KR, VR]) Map(key optional.Option[K], value optional.Option[V]) (optional.Option[KR] /* key */, optional.Option[VR] /* value */, error) {
	return fn(key, value)
}

/*
type StreamMapProcessor struct {
	mapper Mapper
	name   string
	BaseProcessor
}

var _ = Processor(&StreamMapProcessor{})

func NewStreamMapProcessor(name string, mapper Mapper) *StreamMapProcessor {
	p := &StreamMapProcessor{
		mapper:        mapper,
		name:          name,
		BaseProcessor: BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamMapProcessor) Name() string {
	return p.name
}

func (p *StreamMapProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	newK, newV, err := p.mapper.Map(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{{Key: newK, Value: newV, Timestamp: msg.Timestamp}}, nil
}
*/

type StreamMapProcessorG[K, V, KR, VR any] struct {
	mapper MapperG[K, V, KR, VR]
	name   string
	BaseProcessorG[K, V, KR, VR]
}

var _ = ProcessorG[int, int, string, string](&StreamMapProcessorG[int, int, string, string]{})

func NewStreamMapProcessorG[K, V, KR, VR any](name string, mapper MapperG[K, V, KR, VR]) ProcessorG[K, V, KR, VR] {
	p := &StreamMapProcessorG[K, V, KR, VR]{
		mapper:         mapper,
		name:           name,
		BaseProcessorG: BaseProcessorG[K, V, KR, VR]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}
func (p *StreamMapProcessorG[K, V, KR, VR]) Name() string { return p.name }
func (p *StreamMapProcessorG[K, V, KR, VR]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) ([]commtypes.MessageG[KR, VR], error) {
	newK, newV, err := p.mapper.Map(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.MessageG[KR, VR]{{Key: newK, Value: newV,
		TimestampMs: msg.TimestampMs, StartProcTime: msg.StartProcTime}}, nil
}

// type ValueMapperWithKey interface {
// 	MapValue(key interface{}, value interface{}) (interface{}, error)
// }
// type ValueMapperWithKeyFunc func(key interface{}, value interface{}) (interface{}, error)

// var _ = ValueMapperWithKey(ValueMapperWithKeyFunc(nil))

// func (fn ValueMapperWithKeyFunc) MapValue(key interface{}, value interface{}) (interface{}, error) {
// 	return fn(key, value)
// }

type ValueMapperWithKeyG[K, V, VR any] interface {
	MapValue(key optional.Option[K], value optional.Option[V]) (optional.Option[VR], error)
}
type ValueMapperWithKeyFuncG[K, V, VR any] func(key optional.Option[K], value optional.Option[V]) (optional.Option[VR], error)

var _ = ValueMapperWithKeyG[int, int, string](ValueMapperWithKeyFuncG[int, int, string](nil))

func (fn ValueMapperWithKeyFuncG[K, V, VR]) MapValue(key optional.Option[K], value optional.Option[V]) (optional.Option[VR], error) {
	return fn(key, value)
}

/*
type StreamMapValuesProcessor struct {
	valueMapperWithKey ValueMapperWithKey
	name               string
	BaseProcessor
}

var _ = Processor(&StreamMapValuesProcessor{})

func NewStreamMapValuesProcessor(name string, mapper ValueMapperWithKey) *StreamMapValuesProcessor {
	p := &StreamMapValuesProcessor{
		valueMapperWithKey: mapper,
		name:               name,
		BaseProcessor:      BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamMapValuesProcessor) Name() string {
	return p.name
}

func (p *StreamMapValuesProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	newV, err := p.valueMapperWithKey.MapValue(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{{Key: msg.Key, Value: newV, Timestamp: msg.Timestamp}}, nil
}
*/

type StreamMapValuesProcessorG[K, V, VR any] struct {
	valueMapperWithKey ValueMapperWithKeyG[K, V, VR]
	name               string
	BaseProcessorG[K, V, K, VR]
}

var _ = ProcessorG[int, string, int, string](&StreamMapValuesProcessorG[int, string, string]{})

func NewStreamMapValuesProcessorG[K, V, VR any](name string, mapper ValueMapperWithKeyG[K, V, VR]) *StreamMapValuesProcessorG[K, V, VR] {
	p := &StreamMapValuesProcessorG[K, V, VR]{
		valueMapperWithKey: mapper,
		name:               name,
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *StreamMapValuesProcessorG[K, V, VR]) Name() string {
	return p.name
}

func (p *StreamMapValuesProcessorG[K, V, VR]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) ([]commtypes.MessageG[K, VR], error) {
	newV, err := p.valueMapperWithKey.MapValue(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.MessageG[K, VR]{{Key: msg.Key, Value: newV,
		TimestampMs: msg.TimestampMs, StartProcTime: msg.StartProcTime}}, nil
}

/*
type SelectKeyMapper interface {
	SelectKey(key, value interface{}) (interface{}, error)
}
type SelectKeyFunc func(key, value interface{}) (interface{}, error)

var _ = SelectKeyMapper(SelectKeyFunc(nil))

func (fn SelectKeyFunc) SelectKey(key, value interface{}) (interface{}, error) {
	return fn(key, value)
}
*/

type SelectKeyMapperG[K, V, KR any] interface {
	SelectKey(key optional.Option[K], value optional.Option[V]) (optional.Option[KR], error)
}
type SelectKeyFuncG[K, V, KR any] func(key optional.Option[K], value optional.Option[V]) (optional.Option[KR], error)

var _ = SelectKeyMapperG[int, int, string](SelectKeyFuncG[int, int, string](nil))

func (fn SelectKeyFuncG[K, V, KR]) SelectKey(key optional.Option[K], value optional.Option[V]) (optional.Option[KR], error) {
	return fn(key, value)
}

/*
type StreamSelectKeyProcessor struct {
	selectKey SelectKeyMapper
	name      string
	BaseProcessor
}

func NewStreamSelectKeyProcessor(name string, keySelector SelectKeyMapper) Processor {
	p := &StreamSelectKeyProcessor{
		name:      name,
		selectKey: keySelector,
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamSelectKeyProcessor) Name() string {
	return p.name
}

func (p *StreamSelectKeyProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	newKey, err := p.selectKey.SelectKey(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{{Key: newKey, Value: msg.Value,
		Timestamp: msg.Timestamp}}, nil
}
*/

type StreamSelectKeyProcessorG[K, V, KR any] struct {
	selectKey SelectKeyMapperG[K, V, KR]
	name      string
	BaseProcessorG[K, V, KR, V]
}

func NewStreamSelectKeyProcessorG[K, V, KR any](name string, keySelector SelectKeyMapperG[K, V, KR]) ProcessorG[K, V, KR, V] {
	p := &StreamSelectKeyProcessorG[K, V, KR]{
		name:      name,
		selectKey: keySelector,
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *StreamSelectKeyProcessorG[K, V, KR]) Name() string {
	return p.name
}

func (p *StreamSelectKeyProcessorG[K, V, KR]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) ([]commtypes.MessageG[KR, V], error) {
	newKey, err := p.selectKey.SelectKey(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	return []commtypes.MessageG[KR, V]{{Key: newKey, Value: msg.Value,
		TimestampMs: msg.TimestampMs, StartProcTime: msg.StartProcTime}}, nil
}
