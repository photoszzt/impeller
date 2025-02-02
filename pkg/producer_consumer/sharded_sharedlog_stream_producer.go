package producer_consumer

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/bits"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	eo_intr "sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils"
	"slices"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/rs/zerolog/log"
)

type StreamSinkConfig struct {
	// MsgSerde      commtypes.MessageSerdeG
	FlushDuration time.Duration
	Format        commtypes.SerdeFormat
}

type ShardedSharedLogStreamProducer struct {
	msgSerSerde   commtypes.SerdeG[commtypes.MessageSerialized]
	eom           eo_intr.ReadOnlyExactlyOnceManager
	stream        *sharedlog_stream.ShardedSharedLogStream
	flushCallback eo_intr.FlushCallbackFunc
	name          string
	bufPush       bool
	guarantee     eo_intr.GuaranteeMth
	isFinalOutput bool
}

var _ = Producer(&ShardedSharedLogStreamProducer{})

func NewShardedSharedLogStreamProducer(stream *sharedlog_stream.ShardedSharedLogStream,
	config *StreamSinkConfig,
) *ShardedSharedLogStreamProducer {
	msgSerSerde, err := commtypes.GetMessageSerializedSerdeG(config.Format)
	if err != nil {
		log.Fatal().Err(err).Msg("Fail to get msg serialized serdeG")
	}
	return &ShardedSharedLogStreamProducer{
		// msgSerde:      config.MsgSerde,
		msgSerSerde:   msgSerSerde,
		stream:        stream,
		bufPush:       utils.CheckBufPush(),
		name:          "sink",
		isFinalOutput: false,
		flushCallback: func(ctx context.Context) error { return nil },
	}
}

func (sls *ShardedSharedLogStreamProducer) SetLastMarkerSeq(lastMarkerSeq uint64) {
	sls.stream.SetLastMarkerSeq(lastMarkerSeq)
}

func (sls *ShardedSharedLogStreamProducer) MarkFinalOutput() {
	sls.isFinalOutput = true
}

func (sls *ShardedSharedLogStreamProducer) SetFlushCallback(flushCallback eo_intr.FlushCallbackFunc) {
	sls.flushCallback = flushCallback
}

func (sls *ShardedSharedLogStreamProducer) OutputRemainingStats() {
	sls.stream.OutputRemainingStats()
}

func (sls *ShardedSharedLogStreamProducer) IsFinalOutput() bool {
	return sls.isFinalOutput
}

func (sls *ShardedSharedLogStreamProducer) ResetInitialProd() {
	sls.stream.ResetInitialProd()
}

func (sls *ShardedSharedLogStreamProducer) GetInitialProdSeqNum(substreamNum uint8) uint64 {
	return sls.stream.GetInitialProdSeqNum(substreamNum)
}

// this method is not goroutine safe
func (sls *ShardedSharedLogStreamProducer) SetName(name string) {
	sls.name = name
}

func (sls *ShardedSharedLogStreamProducer) Name() string {
	return sls.name
}

func (sls *ShardedSharedLogStreamProducer) Stream() sharedlog_stream.Stream {
	return sls.stream
}

func (sls *ShardedSharedLogStreamProducer) ConfigExactlyOnce(
	eos eo_intr.ReadOnlyExactlyOnceManager,
	guarantee eo_intr.GuaranteeMth,
) {
	sls.guarantee = guarantee
	sls.stream.ExactlyOnce(guarantee)
	sls.eom = eos
}

func (sls *ShardedSharedLogStreamProducer) ProduceCtrlMsg(ctx context.Context, msg *commtypes.RawMsgAndSeq, parNums []uint8) (int, error) {
	if sls.bufPush {
		_, err := sls.flush(ctx)
		if err != nil {
			return 0, err
		}
	}
	tpNameHash := sls.Stream().TopicNameHash()
	if msg.Mark == commtypes.SCALE_FENCE {
		for _, parNum := range parNums {
			scale_fence_tag := txn_data.ScaleFenceTag(tpNameHash, parNum)
			nameTag := sharedlog_stream.NameHashWithPartition(tpNameHash, parNum)
			_, err := sls.pushWithTag(ctx, msg.Payload, parNum, []uint64{scale_fence_tag, nameTag},
				nil, sharedlog_stream.ControlRecordMeta)
			if err != nil {
				return 0, err
			}
			// fmt.Fprintf(os.Stderr, "produce scale fence %s(%d): offset 0x%x\n", sls.TopicName(), parNum, off)
		}
		return len(parNums), nil
	} else if msg.Mark == commtypes.STREAM_END {
		for _, parNum := range parNums {
			off, err := sls.push(ctx, msg.Payload, parNum, sharedlog_stream.ControlRecordMeta)
			debug.Fprintf(os.Stderr, "Producer %v: push end of stream to %s %d at 0x%x\n", sls.eom.GetProducerId(), sls.Stream().TopicName(), parNum, off)
			if err != nil {
				return 0, err
			}
			env := ctx.Value(commtypes.ENVID{}).(types.Environment)
			tag := sharedlog_stream.NameHashWithPartition(sls.stream.TopicNameHash(), parNum)
			logEntry, err := env.SharedLogReadNext(ctx, tag, off)
			if err != nil {
				return 0, err
			}
			sEntry := sharedlog_stream.DecodeStreamLogEntry(logEntry)
			isControl := bits.Has(bits.Bits(sEntry.Meta), sharedlog_stream.Control)
			debug.Assert(sls.eom.GetCurrentEpoch() == sEntry.TaskEpoch, "task epoch should match")
			debug.Assert(sls.eom.GetCurrentTaskId() == sEntry.TaskId, "task id should match")
			debug.Assert(isControl, "should read back the stream end")
			debug.Assert(logEntry.SeqNum == off, "should read back the stream end")
			debug.Assert(sEntry.TopicName[0] == sls.stream.TopicName(), "topic name should match")
			debug.Assert(slices.Equal(sEntry.Payload, msg.Payload), "payload should be the same")
		}
		return len(parNums), nil
	} else if msg.Mark == commtypes.CHKPT_MARK {
		for _, parNum := range parNums {
			chkpt_tag := txn_data.ChkptTag(tpNameHash, parNum)
			nameTag := sharedlog_stream.NameHashWithPartition(tpNameHash, parNum)
			_, err := sls.pushWithTag(ctx, msg.Payload, parNum, []uint64{chkpt_tag, nameTag},
				nil, sharedlog_stream.ControlRecordMeta)
			if err != nil {
				return 0, err
			}
			// fmt.Fprintf(os.Stderr, "produce scale fence %s(%d): offset 0x%x\n", sls.TopicName(), parNum, off)
		}
		return len(parNums), nil
	} else {
		return 0, fmt.Errorf("unknown control message with mark: %s", msg.Mark)
	}
}

func (sls *ShardedSharedLogStreamProducer) ProduceData(ctx context.Context, msgSer commtypes.MessageSerialized, parNum uint8) error {
	bytes, _, err := sls.msgSerSerde.Encode(msgSer)
	if err != nil {
		return err
	}
	if bytes != nil {
		if sls.bufPush {
			return sls.bufpush(ctx, bytes, parNum)
		} else {
			err = sls.flushCallback(ctx)
			if err != nil {
				return err
			}
			_, err := sls.push(ctx, bytes, parNum, sharedlog_stream.StreamEntryMeta(false, false))
			// if sls.msgSerSerde.UsedBufferPool() && b != nil {
			// 	*b = bytes
			// 	commtypes.PushBuffer(b)
			// }
			return err
		}
	}
	return nil
}

func (sls *ShardedSharedLogStreamProducer) ProduceDataNoLock(ctx context.Context, msgSer commtypes.MessageSerialized, parNum uint8) error {
	bytes, b, err := sls.msgSerSerde.Encode(msgSer)
	if err != nil {
		return err
	}
	if bytes != nil {
		// vret, err := sls.msgSerSerde.Decode(bytes)
		// if err != nil {
		// 	log.Fatal().Err(err).Msg("fail to decode encoded bytes")
		// }
		// if !msgSer.Equal(&vret) {
		// 	rdbg.PrintStack()
		// 	log.Fatal().Err(err).
		// 		Bytes("inKeyEnc", msgSer.KeyEnc).
		// 		Bytes("inValEnc", msgSer.ValueEnc).
		// 		Int64("inInjTms", msgSer.InjTMs).
		// 		Int64("inTs", msgSer.TimestampMs).
		// 		Bytes("retKeyEnc", vret.KeyEnc).
		// 		Bytes("retValEnc", vret.ValueEnc).
		// 		Int64("retInjTms", vret.InjTMs).
		// 		Int64("retTs", vret.TimestampMs).
		// 		Msg("encode and redecode doesn't get back same input")
		// }
		if sls.bufPush {
			return sls.bufpushNoLock(ctx, bytes, parNum)
		} else {
			err = sls.flushCallback(ctx)
			if err != nil {
				return err
			}
			_, err := sls.push(ctx, bytes, parNum, sharedlog_stream.StreamEntryMeta(false, false))
			if sls.msgSerSerde.UsedBufferPool() && b != nil {
				*b = bytes
				commtypes.PushBuffer(b)
			}
			return err
		}
	}
	return nil
}

func (s *ShardedSharedLogStreamProducer) TopicName() string { return s.stream.TopicName() }
func (s *ShardedSharedLogStreamProducer) Flush(ctx context.Context) (uint32, error) {
	if s.bufPush {
		return s.flush(ctx)
	}
	return 0, nil
}

func (s *ShardedSharedLogStreamProducer) FlushNoLock(ctx context.Context) (uint32, error) {
	if s.bufPush {
		return s.flushNoLock(ctx)
	}
	return 0, nil
}

func (s *ShardedSharedLogStreamProducer) flushNoLock(ctx context.Context) (uint32, error) {
	if s.guarantee == eo_intr.TWO_PHASE_COMMIT || s.guarantee == eo_intr.EPOCH_MARK ||
		s.guarantee == eo_intr.ALIGN_CHKPT || s.guarantee == eo_intr.REMOTE_2PC {
		return s.stream.FlushNoLock(ctx, s.eom.GetProducerId())
	} else {
		return s.stream.FlushNoLock(ctx, commtypes.EmptyProducerId)
	}
}

func (s *ShardedSharedLogStreamProducer) flush(ctx context.Context) (uint32, error) {
	if s.guarantee == eo_intr.TWO_PHASE_COMMIT || s.guarantee == eo_intr.EPOCH_MARK ||
		s.guarantee == eo_intr.ALIGN_CHKPT || s.guarantee == eo_intr.REMOTE_2PC {
		return s.stream.Flush(ctx, s.eom.GetProducerId())
	} else {
		return s.stream.Flush(ctx, commtypes.EmptyProducerId)
	}
}

func (s *ShardedSharedLogStreamProducer) push(ctx context.Context, payload []byte, parNumber uint8, meta sharedlog_stream.LogEntryMeta) (uint64, error) {
	if s.guarantee == eo_intr.TWO_PHASE_COMMIT || s.guarantee == eo_intr.EPOCH_MARK ||
		s.guarantee == eo_intr.ALIGN_CHKPT || s.guarantee == eo_intr.REMOTE_2PC {
		return s.stream.Push(ctx, payload, parNumber, meta, s.eom.GetProducerId())
	} else {
		return s.stream.Push(ctx, payload, parNumber, meta, commtypes.EmptyProducerId)
	}
}

func (s *ShardedSharedLogStreamProducer) pushWithTag(ctx context.Context, payload []byte, parNumber uint8, tag []uint64,
	additionalTopic []string, meta sharedlog_stream.LogEntryMeta,
) (uint64, error) {
	if s.guarantee == eo_intr.TWO_PHASE_COMMIT || s.guarantee == eo_intr.EPOCH_MARK ||
		s.guarantee == eo_intr.ALIGN_CHKPT || s.guarantee == eo_intr.REMOTE_2PC {
		return s.stream.PushWithTag(ctx, payload, parNumber, tag, additionalTopic, meta, s.eom.GetProducerId())
	} else {
		return s.stream.PushWithTag(ctx, payload, parNumber, tag, additionalTopic, meta, commtypes.EmptyProducerId)
	}
}

func (s *ShardedSharedLogStreamProducer) bufpush(ctx context.Context, payload []byte, parNum uint8) error {
	if s.guarantee == eo_intr.TWO_PHASE_COMMIT || s.guarantee == eo_intr.EPOCH_MARK ||
		s.guarantee == eo_intr.ALIGN_CHKPT || s.guarantee == eo_intr.REMOTE_2PC {
		return s.stream.BufPush(ctx, payload, parNum, s.eom.GetProducerId(), s.flushCallback)
	} else {
		return s.stream.BufPush(ctx, payload, parNum, commtypes.EmptyProducerId, s.flushCallback)
	}
}

func (s *ShardedSharedLogStreamProducer) bufpushNoLock(ctx context.Context, payload []byte, parNum uint8) error {
	if s.guarantee == eo_intr.TWO_PHASE_COMMIT || s.guarantee == eo_intr.EPOCH_MARK || s.guarantee == eo_intr.ALIGN_CHKPT ||
		s.guarantee == eo_intr.REMOTE_2PC {
		return s.stream.BufPushNoLock(ctx, payload, parNum, s.eom.GetProducerId(), s.flushCallback)
	} else {
		return s.stream.BufPushNoLock(ctx, payload, parNum, commtypes.EmptyProducerId, s.flushCallback)
	}
}
