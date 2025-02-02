package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/producer_consumer"
)

type RoundRobinOutputProcessorG[KIn, VIn any] struct {
	producer   producer_consumer.MeteredProducerIntr
	ectx       ExecutionContext
	msgGSerdeG commtypes.MessageGSerdeG[KIn, VIn]
	name       string
	BaseProcessorG[KIn, VIn, any, any]
	curSubstream uint8
	kUseBuf      bool
	vUseBuf      bool
}

var _ ProcessorG[int, int, any, any] = &RoundRobinOutputProcessorG[int, int]{}

func NewRoundRobinOutputProcessorG[KIn, VIn any](processTimeTag string,
	ectx ExecutionContext,
	producer producer_consumer.MeteredProducerIntr,
	msgGSerdeG commtypes.MessageGSerdeG[KIn, VIn],
) *RoundRobinOutputProcessorG[KIn, VIn] {
	r := &RoundRobinOutputProcessorG[KIn, VIn]{
		producer:       producer,
		ectx:           ectx,
		msgGSerdeG:     msgGSerdeG,
		name:           "to" + producer.TopicName(),
		BaseProcessorG: BaseProcessorG[KIn, VIn, any, any]{},
		curSubstream:   0,
		kUseBuf:        msgGSerdeG.GetKeySerdeG().UsedBufferPool(),
		vUseBuf:        msgGSerdeG.GetValSerdeG().UsedBufferPool(),
	}
	r.BaseProcessorG.ProcessingFuncG = r.ProcessAndReturn
	return r
}

func (p *RoundRobinOutputProcessorG[KIn, VIn]) OutputRemainingStats() {
	// p.procTimeStats.PrintRemainingStats()
}

func (p *RoundRobinOutputProcessorG[KIn, VIn]) Name() string {
	return p.name
}

func (p *RoundRobinOutputProcessorG[KIn, VIn]) ProcessAndReturn(ctx context.Context,
	msg commtypes.MessageG[KIn, VIn],
) ([]commtypes.MessageG[any, any], error) {
	msgSerOp, kbuf, vbuf, err := commtypes.MsgGToMsgSer(msg, p.msgGSerdeG)
	if err != nil {
		return nil, err
	}
	msgSer, ok := msgSerOp.Take()
	if ok {
		par := p.curSubstream
		p.curSubstream = (p.curSubstream + 1) % p.producer.Stream().NumPartition()
		p.ectx.TrackParFunc()(p.producer.TopicName(), par)
		err = p.producer.ProduceData(ctx, msgSer, par)
		if p.kUseBuf && msgSer.KeyEnc != nil && kbuf != nil {
			*kbuf = msgSer.KeyEnc
			commtypes.PushBuffer(kbuf)
		}
		if p.vUseBuf && msgSer.ValueEnc != nil && vbuf != nil {
			*vbuf = msgSer.ValueEnc
			commtypes.PushBuffer(vbuf)
		}
		return nil, err
	} else {
		return nil, nil
	}
}
