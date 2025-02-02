package producer_consumer

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
)

type Consumer interface {
	// Consume gets the next commtypes.Message from the source
	Consume(ctx context.Context, parNum uint8) (*commtypes.RawMsgAndSeq, error)
	SetCursor(cursor uint64, parNum uint8)
	TopicName() string
	Name() string
	SetName(string)
	NumSubstreamProducer() uint8
	Stream() sharedlog_stream.Stream
	ConfigExactlyOnce(guarantee exactly_once_intr.GuaranteeMth)
	SetInitialSource(initial bool)
	IsInitialSource() bool
	RecordCurrentConsumedSeqNum(seqNum uint64)
	CurrentConsumedSeqNum() uint64
	SrcProducerEnd(prodIdx uint8)
	AllProducerEnded() bool
}

type MeteredConsumerIntr interface {
	Consumer
	StartWarmup()
	GetCount() uint64
	NumCtrlMsg() uint32
}
