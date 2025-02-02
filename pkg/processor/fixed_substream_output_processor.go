package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/producer_consumer"
)

type FixedSubstreamOutputProcessorG[KIn, VIn any] struct {
	producer   producer_consumer.MeteredProducerIntr
	msgGSerdeG commtypes.MessageGSerdeG[KIn, VIn]
	name       string
	// procTimeStats stats.PrintLogStatsCollector[int64]
	BaseProcessorG[KIn, VIn, any, any]
	fixedSubstream uint8
	kUseBuf        bool
	vUseBuf        bool
}

var _ ProcessorG[int, int, any, any] = &FixedSubstreamOutputProcessorG[int, int]{}

func NewFixedSubstreamOutputProcessorG[KIn, VIn any](processTimeTag string,
	producer producer_consumer.MeteredProducerIntr, fixedSubNum uint8,
	msgGSerdeG commtypes.MessageGSerdeG[KIn, VIn],
) *FixedSubstreamOutputProcessorG[KIn, VIn] {
	r := &FixedSubstreamOutputProcessorG[KIn, VIn]{
		name:           "to" + producer.TopicName(),
		producer:       producer,
		fixedSubstream: fixedSubNum,
		BaseProcessorG: BaseProcessorG[KIn, VIn, any, any]{},
		msgGSerdeG:     msgGSerdeG,
		// procTimeStats:  stats.NewPrintLogStatsCollector[int64](processTimeTag),
		kUseBuf: msgGSerdeG.GetKeySerdeG().UsedBufferPool(),
		vUseBuf: msgGSerdeG.GetValSerdeG().UsedBufferPool(),
	}
	r.BaseProcessorG.ProcessingFuncG = r.ProcessAndReturn
	return r
}

func (p *FixedSubstreamOutputProcessorG[KIn, VIn]) OutputRemainingStats() {
	// p.procTimeStats.PrintRemainingStats()
}

func (p *FixedSubstreamOutputProcessorG[KIn, VIn]) Name() string {
	return p.name
}

func (p *FixedSubstreamOutputProcessorG[KIn, VIn]) ProcessAndReturn(ctx context.Context,
	msg commtypes.MessageG[KIn, VIn],
) ([]commtypes.MessageG[any, any], error) {
	// procTime := time.Since(msg.StartProcTime)
	// p.procTimeStats.AddSample(procTime.Nanoseconds())
	msgSerOp, kbuf, vbuf, err := commtypes.MsgGToMsgSer(msg, p.msgGSerdeG)
	if err != nil {
		return nil, err
	}
	msgSer, ok := msgSerOp.Take()
	if ok {
		err := p.producer.ProduceData(ctx, msgSer, p.fixedSubstream)
		if err != nil {
			return nil, err
		}
		if p.kUseBuf && msgSer.KeyEnc != nil && kbuf != nil {
			*kbuf = msgSer.KeyEnc
			commtypes.PushBuffer(kbuf)
		}
		if p.vUseBuf && msgSer.ValueEnc != nil && vbuf != nil {
			*vbuf = msgSer.ValueEnc
			commtypes.PushBuffer(vbuf)
		}
	}
	return nil, nil
}
