package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/stats"
)

/*
type MeteredProcessor struct {
	proc      Processor
	latencies stats.StatsCollector[int64]
}

func NewMeteredProcessor(proc Processor) *MeteredProcessor {
	return &MeteredProcessor{
		proc:      proc,
		latencies: stats.NewStatsCollector[int64](proc.Name(), stats.DEFAULT_COLLECT_DURATION),
	}
}

func (p *MeteredProcessor) Name() string {
	return p.proc.Name()
}

func (p *MeteredProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	procStart := stats.TimerBegin()
	msgs, err := p.proc.ProcessAndReturn(ctx, msg)
	elapsed := stats.Elapsed(procStart).Microseconds()
	p.latencies.AddSample(elapsed)
	return msgs, err
}

func (p *MeteredProcessor) Process(ctx context.Context, msg commtypes.Message) error {
	return p.proc.Process(ctx, msg)
}

func (p *MeteredProcessor) NextProcessor(nextProcessor IProcess) {
	p.proc.NextProcessor(nextProcessor)
}

type MeteredCachedProcessor struct {
	proc      CachedProcessor
	latencies stats.StatsCollector[int64]
}

func (p *MeteredCachedProcessor) Name() string {
	return p.proc.Name()
}

func (p *MeteredCachedProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	procStart := stats.TimerBegin()
	msgs, err := p.proc.ProcessAndReturn(ctx, msg)
	elapsed := stats.Elapsed(procStart).Microseconds()
	p.latencies.AddSample(elapsed)
	return msgs, err
}

func (p *MeteredCachedProcessor) Flush(ctx context.Context) error {
	return p.proc.Flush(ctx)
}

func (p *MeteredCachedProcessor) Process(ctx context.Context, msg commtypes.Message) error {
	return p.proc.Process(ctx, msg)
}

func (p *MeteredCachedProcessor) NextProcessor(nextProcessor IProcess) {
	p.proc.NextProcessor(nextProcessor)
}
*/

type MeteredProcessorG[KIn, VIn, KOut, VOut any] struct {
	proc      ProcessorG[KIn, VIn, KOut, VOut]
	latencies stats.StatsCollector[int64]
}

func NewMeteredProcessorG[KI, VI, KO, VO any](proc ProcessorG[KI, VI, KO, VO]) *MeteredProcessorG[KI, VI, KO, VO] {
	return &MeteredProcessorG[KI, VI, KO, VO]{
		proc:      proc,
		latencies: stats.NewStatsCollector[int64](proc.Name(), stats.DEFAULT_COLLECT_DURATION),
	}
}

func (p *MeteredProcessorG[KIn, VIn, KOut, VOut]) Name() string { return p.proc.Name() }
func (p *MeteredProcessorG[KIn, VIn, KOut, VOut]) ProcessAndReturn(ctx context.Context,
	msg commtypes.MessageG[KIn, VIn],
) ([]commtypes.MessageG[KOut, VOut], error) {
	procStart := stats.TimerBegin()
	msgs, err := p.proc.ProcessAndReturn(ctx, msg)
	elapsed := stats.Elapsed(procStart).Microseconds()
	p.latencies.AddSample(elapsed)
	return msgs, err
}

func (p *MeteredProcessorG[KIn, VIn, KOut, VOut]) Process(ctx context.Context,
	msg commtypes.MessageG[KIn, VIn],
) error {
	return p.proc.Process(ctx, msg)
}

func (p *MeteredProcessorG[KIn, VIn, KOut, VOut]) NextProcessor(nextProcessor IProcessG[KOut, VOut]) {
	p.proc.NextProcessor(nextProcessor)
}
