package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
)

/*
type StreamFilterProcessor struct {
	pred Predicate
	name string
	BaseProcessor
}

var _ = Processor(&StreamFilterProcessor{})

func NewStreamFilterProcessor(name string, pred Predicate) *StreamFilterProcessor {
	p := &StreamFilterProcessor{
		pred: pred,
		name: name,
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamFilterProcessor) Name() string {
	return p.name
}

func (p *StreamFilterProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	ok, err := p.pred.Assert(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	if ok {
		return []commtypes.Message{msg}, nil
	}
	return nil, nil
}
*/

type StreamFilterProcessorG[K, V any] struct {
	pred PredicateG[K, V]
	name string
	BaseProcessorG[K, V, K, V]
}

var _ = ProcessorG[int, int, int, int](&StreamFilterProcessorG[int, int]{})

func NewStreamFilterProcessorG[K, V any](name string, pred PredicateG[K, V]) ProcessorG[K, V, K, V] {
	p := &StreamFilterProcessorG[K, V]{
		pred:           pred,
		name:           name,
		BaseProcessorG: BaseProcessorG[K, V, K, V]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *StreamFilterProcessorG[K, V]) Name() string {
	return p.name
}

func (p *StreamFilterProcessorG[K, V]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) ([]commtypes.MessageG[K, V], error) {
	ok, err := p.pred.Assert(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	if ok {
		return []commtypes.MessageG[K, V]{msg}, nil
	}
	return nil, nil
}

/*
type StreamFilterNotProcessor struct {
	pred Predicate
	name string
	BaseProcessor
}

var _ = Processor(&StreamFilterNotProcessor{})

func NewStreamFilterNotProcessor(name string, pred Predicate) *StreamFilterNotProcessor {
	p := &StreamFilterNotProcessor{
		pred: pred,
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamFilterNotProcessor) Name() string {
	return p.name
}

func (p *StreamFilterNotProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	ok, err := p.pred.Assert(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []commtypes.Message{msg}, nil
	}
	return nil, nil
}
*/

type StreamFilterNotProcessorG[K, V any] struct {
	pred PredicateG[K, V]
	name string
	BaseProcessorG[K, V, K, V]
}

var _ = ProcessorG[int, int, int, int](&StreamFilterNotProcessorG[int, int]{})

func NewStreamFilterNotProcessorG[K, V any](name string, pred PredicateG[K, V]) *StreamFilterNotProcessorG[K, V] {
	p := &StreamFilterNotProcessorG[K, V]{
		pred:           pred,
		name:           name,
		BaseProcessorG: BaseProcessorG[K, V, K, V]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *StreamFilterNotProcessorG[K, V]) Name() string { return p.name }
func (p *StreamFilterNotProcessorG[K, V]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) ([]commtypes.MessageG[K, V], error) {
	ok, err := p.pred.Assert(msg.Key, msg.Value)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []commtypes.MessageG[K, V]{msg}, nil
	}
	return nil, nil
}
