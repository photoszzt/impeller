package consume_seq_num_manager

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/txn_data"

	"cs.utexas.edu/zjia/faas/protocol"
)

const (
	CONSUMER_OFFSET_LOG_TOPIC_NAME = "__con_offset_log"
)

type ConsumeSeqManager struct {
	offsetMarkerSerde  commtypes.SerdeG[commtypes.OffsetMarker]
	offsetLog          *sharedlog_stream.ShardedSharedLogStream
	offsetLogMarkerTag uint64
	useBuf             bool
}

func NewConsumeSeqManager(serdeFormat commtypes.SerdeFormat,
	conSeqMngrName string, bufMaxSize uint32,
) (*ConsumeSeqManager, error) {
	offsetMarkerSerde, err := commtypes.GetOffsetMarkerSerdeG(serdeFormat)
	if err != nil {
		return nil, err
	}
	off, err := sharedlog_stream.NewShardedSharedLogStream(
		CONSUMER_OFFSET_LOG_TOPIC_NAME+"_"+conSeqMngrName, 1, serdeFormat, bufMaxSize)
	if err != nil {
		return nil, err
	}
	return &ConsumeSeqManager{
		offsetMarkerSerde:  offsetMarkerSerde,
		offsetLog:          off,
		offsetLogMarkerTag: txn_data.MarkerTag(off.TopicNameHash(), 0),
		useBuf:             offsetMarkerSerde.UsedBufferPool(),
	}, nil
}

func (cm *ConsumeSeqManager) FindLastConsumedSeqNum(ctx context.Context) (map[string]uint64, error) {
	// find the most recent transaction marker
	var txnMkRawMsg *commtypes.RawMsg
	var err error

	txnMkRawMsg, err = cm.offsetLog.ReadBackwardWithTag(ctx, protocol.MaxLogSeqnum, 0, cm.offsetLogMarkerTag)
	if err != nil {
		return nil, err
	}
	// debug.Fprintf(os.Stderr, "CM: offlog got entry off %x, control %v\n",
	// 	txnMkRawMsg.LogSeqNum, txnMkRawMsg.IsControl)
	mark, err := cm.offsetMarkerSerde.Decode(txnMkRawMsg.Payload)
	if err != nil {
		return nil, err
	}
	return mark.ConSeqNums, nil
}

func CollectOffsetMarker(consumers []*producer_consumer.MeteredConsumer) map[string]uint64 {
	ret := make(map[string]uint64)
	for _, consumer := range consumers {
		topic := consumer.TopicName()
		offset := consumer.CurrentConsumedSeqNum()
		ret[topic] = offset
	}
	return ret
}

func (cm *ConsumeSeqManager) Track(ctx context.Context, offsetMarkers map[string]uint64) error {
	offsetMarker := commtypes.OffsetMarker{
		ConSeqNums: offsetMarkers,
	}
	encoded, b, err := cm.offsetMarkerSerde.Encode(offsetMarker)
	defer func() {
		if cm.useBuf && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return err
	}
	_, err = cm.offsetLog.PushWithTag(ctx, encoded, 0, []uint64{cm.offsetLogMarkerTag}, nil,
		sharedlog_stream.ControlRecordMeta, commtypes.EmptyProducerId)
	if err != nil {
		return err
	}
	// debug.Fprintf(os.Stderr, "append marker %d to stream %s off %x\n",
	// 	commtypes.EPOCH_END, stream.TopicName(), off)
	return nil
}
