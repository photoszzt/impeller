package producer_consumer

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	exactly_once_intr "sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
)

type Producer interface {
	ProduceData(ctx context.Context, msgSer commtypes.MessageSerialized, parNum uint8) error
	ProduceCtrlMsg(ctx context.Context, msg *commtypes.RawMsgAndSeq, parNums []uint8) (int, error)
	TopicName() string
	Name() string
	SetName(string)
	Flush(ctx context.Context) (uint32, error)
	ConfigExactlyOnce(rem exactly_once_intr.ReadOnlyExactlyOnceManager,
		guarantee exactly_once_intr.GuaranteeMth)
	Stream() sharedlog_stream.Stream
	GetInitialProdSeqNum(substreamNum uint8) uint64
	ResetInitialProd()
	SetLastMarkerSeq(lastMarkerSeq uint64)
	SetFlushCallback(cb exactly_once_intr.FlushCallbackFunc)
}

type MeteredProducerIntr interface {
	Producer
	MarkFinalOutput()
	IsFinalOutput() bool
	StartWarmup()
	GetEventTimeLatency() []int
	GetEventTs() []int64
	GetCount() uint64
	NumCtrlMsg() uint32
	OutputRemainingStats()
}
