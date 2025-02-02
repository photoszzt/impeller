package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
)

type Processor interface {
	Name() string
	// Process processes the stream commtypes.Message.
	ProcessAndReturn(context.Context, commtypes.Message) ([]commtypes.Message, error)
	IProcess
	NextProcessor(nextProcessor IProcess)
}

// type CachedProcessor interface {
// 	Processor
// 	Flush(ctx context.Context) error
// }

type IProcessG[KIn, VIn any] interface {
	Process(ctx context.Context, msg commtypes.MessageG[KIn, VIn]) error
}

type IProcess interface {
	Process(ctx context.Context, msg commtypes.Message) error
}

type IProcessFuncG[KIn, VIn any] func(ctx context.Context, msg commtypes.MessageG[KIn, VIn]) error

func (fn IProcessFuncG[KIn, VIn]) Process(ctx context.Context, msg commtypes.MessageG[KIn, VIn]) error {
	return fn(ctx, msg)
}

var _ IProcessG[int, int] = IProcessG[int, int](IProcessFuncG[int, int](nil))

type ProcessorG[KIn, VIn, KOut, VOut any] interface {
	Name() string
	ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[KIn, VIn]) ([]commtypes.MessageG[KOut, VOut], error)
	IProcessG[KIn, VIn]
	NextProcessor(nextProcessor IProcessG[KOut, VOut])
}

// type CachedProcessorG[KIn, VIn, KOut, VOut any] interface {
// 	ProcessorG[KIn, VIn, KOut, VOut]
// 	Flush(ctx context.Context) error
// }

type BaseProcessorG[KIn, VIn, KOut, VOut any] struct {
	ProcessingFuncG func(ctx context.Context, msg commtypes.MessageG[KIn,
		VIn]) ([]commtypes.MessageG[KOut, VOut], error)
	nextProcessors []IProcessG[KOut, VOut]
}

func (b *BaseProcessorG[KIn, VIn, KOut, VOut]) NextProcessor(nextProcessor IProcessG[KOut, VOut]) {
	b.nextProcessors = append(b.nextProcessors, nextProcessor)
}

func (b *BaseProcessorG[KIn, VIn, KOut, VOut]) Process(ctx context.Context,
	msg commtypes.MessageG[KIn, VIn],
) error {
	result, err := b.ProcessingFuncG(ctx, msg)
	if err != nil {
		return err
	}
	if result != nil {
		for _, nextProcessor := range b.nextProcessors {
			for _, nextMsg := range result {
				err = nextProcessor.Process(ctx, nextMsg)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type BaseProcessor struct {
	ProcessingFunc func(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error)
	nextProcessors []IProcess
}

func (b *BaseProcessor) NextProcessor(nextProcessor IProcess) {
	b.nextProcessors = append(b.nextProcessors, nextProcessor)
}

func (b *BaseProcessor) Process(ctx context.Context, msg commtypes.Message) error {
	result, err := b.ProcessingFunc(ctx, msg)
	if err != nil {
		return err
	}
	for _, nextProcessor := range b.nextProcessors {
		for _, nextMsg := range result {
			err = nextProcessor.Process(ctx, nextMsg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
