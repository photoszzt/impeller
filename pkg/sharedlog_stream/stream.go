package sharedlog_stream

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
)

type Stream interface {
	Push(ctx context.Context, payload []byte, parNum uint8, meta LogEntryMeta, producerId commtypes.ProducerId) (uint64, error)
	PushWithTag(ctx context.Context, payload []byte, parNumber uint8, tags []uint64,
		additionalTopic []string, meta LogEntryMeta, producerId commtypes.ProducerId) (uint64, error)
	ReadNext(ctx context.Context, parNum uint8) (*commtypes.RawMsg /* payload */, error)
	ReadNextWithTag(ctx context.Context, parNumber uint8, tag uint64) (*commtypes.RawMsg, error)
	ReadBackwardWithTag(ctx context.Context, tailSeqNum uint64, parNum uint8, tag uint64) (*commtypes.RawMsg, error)
	TopicName() string
	TopicNameHash() uint64
	SetCursor(cursor uint64, parNum uint8)
	NumPartition() uint8
	Flush(ctx context.Context, producerId commtypes.ProducerId) (uint32, error)
	BufPush(ctx context.Context, payload []byte, parNum uint8, producerId commtypes.ProducerId,
		flushCallback exactly_once_intr.FlushCallbackFunc) error
}
