package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
)

type ProcessorChainIntr interface {
	Via(proc Processor) ProcessorChainIntr
	Processors() []Processor
	RunChains(ctx context.Context, initMsg commtypes.Message) ([]commtypes.Message, error)
}

type ProcessorChains struct {
	procs []Processor
}

func NewProcessorChains() ProcessorChains {
	return ProcessorChains{
		procs: make([]Processor, 0, 4),
	}
}

func (pc *ProcessorChains) Via(proc Processor) ProcessorChainIntr {
	pc.procs = append(pc.procs, proc)
	return pc
}

func (pc *ProcessorChains) Processors() []Processor {
	return pc.procs
}

func (pc *ProcessorChains) RunChains(ctx context.Context, initMsg commtypes.Message) ([]commtypes.Message, error) {
	lastMsgs, err := pc.procs[0].ProcessAndReturn(ctx, initMsg)
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(pc.procs); i++ {
		var outMsgs []commtypes.Message
		for _, msg := range lastMsgs {
			out, err := pc.procs[i].ProcessAndReturn(ctx, msg)
			if err != nil {
				return nil, err
			}
			outMsgs = append(outMsgs, out...)
		}
		lastMsgs = outMsgs
	}
	return lastMsgs, nil
}

/*
func ProcessMsg(ctx context.Context, initMsg commtypes.Message, argsTmp interface{}) error {
	args := argsTmp.(ExecutionContext)
	// the last processor is a output function
	_, err := args.RunChains(ctx, initMsg)
	return err
}
*/
