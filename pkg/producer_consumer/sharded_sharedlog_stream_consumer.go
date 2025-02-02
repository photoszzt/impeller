package producer_consumer

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/data_structure"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sync/atomic"
	"time"
)

type ScaleEpochAndBytes struct {
	Payload    []byte
	ScaleEpoch uint16
}

type StartTimeAndProdIdx struct {
	StartTime int64
	ProdIdx   uint8
}

/*
type StreamConsumerConfigG[K, V any] struct {
	// MsgSerde    commtypes.MessageSerdeG
	Timeout     time.Duration
	SerdeFormat commtypes.SerdeFormat
}
*/

type StreamConsumerConfig struct {
	// MsgSerde    commtypes.MessageSerde
	Timeout     time.Duration
	SerdeFormat commtypes.SerdeFormat
}

type ShardedSharedLogStreamConsumer struct {
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker]
	payloadArrSerde  commtypes.SerdeG[commtypes.PayloadArr]
	// msgSerde         commtypes.MessageSerdeG
	producerNotEnd        data_structure.Uint8Set
	producerNotScaleFence data_structure.Uint8Set
	stream                *sharedlog_stream.ShardedSharedLogStream
	tac                   *TransactionAwareConsumer
	emc                   *EpochMarkConsumer
	acc                   *AlignChkptConsumer
	name                  string
	currentSeqNum         uint64
	timeout               time.Duration
	guarantee             exactly_once_intr.GuaranteeMth
	initialSource         bool
	serdeFormat           commtypes.SerdeFormat
	numSrcProducer        uint8
	instanceId            uint8
}

var _ = Consumer(&ShardedSharedLogStreamConsumer{})

func NewShardedSharedLogStreamConsumer(stream *sharedlog_stream.ShardedSharedLogStream,
	config *StreamConsumerConfig, numSrcProducer uint8, instanceId uint8,
) (*ShardedSharedLogStreamConsumer, error) {
	// debug.Fprintf(os.Stderr, "%s timeout: %v\n", stream.TopicName(), config.Timeout)
	epochMarkSerde, err := commtypes.GetEpochMarkerSerdeG(config.SerdeFormat)
	if err != nil {
		return nil, err
	}
	notEnded := make(data_structure.Uint8Set)
	notScaleFenced := make(data_structure.Uint8Set)
	if numSrcProducer != 1 {
		for i := uint8(0); i < numSrcProducer; i++ {
			notEnded.Add(i)
			notScaleFenced.Add(i)
		}
	} else {
		notEnded.Add(instanceId)
		notScaleFenced.Add(instanceId)
	}
	fmt.Fprintf(os.Stderr, "%s notEnded: %v, notScaleFence: %v, epochMarker: %s\n",
		stream.TopicName(), notEnded, notScaleFenced, epochMarkSerde.String())
	return &ShardedSharedLogStreamConsumer{
		epochMarkerSerde: epochMarkSerde,
		stream:           stream,
		timeout:          config.Timeout,
		// msgSerde:         config.MsgSerde,
		payloadArrSerde:       sharedlog_stream.DEFAULT_PAYLOAD_ARR_SERDEG,
		guarantee:             exactly_once_intr.AT_LEAST_ONCE,
		serdeFormat:           config.SerdeFormat,
		numSrcProducer:        numSrcProducer,
		producerNotEnd:        notEnded,
		producerNotScaleFence: notScaleFenced,
	}, nil
}

func (s *ShardedSharedLogStreamConsumer) SetInitialSource(initial bool) {
	s.initialSource = initial
}

func (s *ShardedSharedLogStreamConsumer) IsInitialSource() bool {
	return s.initialSource
}

func (s *ShardedSharedLogStreamConsumer) NumSubstreamProducer() uint8 {
	return s.numSrcProducer
}

func (s *ShardedSharedLogStreamConsumer) SrcProducerEnd(prodIdx uint8) {
	s.producerNotEnd.Remove(prodIdx)
	fmt.Fprintf(os.Stderr, "%d producer ended, %v remain\n", prodIdx, s.producerNotEnd)
}

func (s *ShardedSharedLogStreamConsumer) SrcProducerGotScaleFence(prodIdx uint8) {
	s.producerNotScaleFence.Remove(prodIdx)
	fmt.Fprintf(os.Stderr, "%d producer got scale fence, %v remain\n", prodIdx, s.producerNotScaleFence)
}

func (s *ShardedSharedLogStreamConsumer) AllProducerEnded() bool {
	return len(s.producerNotEnd) == 0
}

func (s *ShardedSharedLogStreamConsumer) AllProducerScaleFenced() bool {
	return len(s.producerNotScaleFence) == 0
}

func (s *ShardedSharedLogStreamConsumer) ConfigExactlyOnce(
	guarantee exactly_once_intr.GuaranteeMth,
) {
	debug.Assert(guarantee == exactly_once_intr.TWO_PHASE_COMMIT ||
		guarantee == exactly_once_intr.EPOCH_MARK ||
		guarantee == exactly_once_intr.REMOTE_2PC ||
		guarantee == exactly_once_intr.ALIGN_CHKPT,
		"configure exactly once should specify 2pc or epoch mark or align checkpoint or remote_2pc")
	s.guarantee = guarantee
	if s.guarantee == exactly_once_intr.TWO_PHASE_COMMIT || s.guarantee == exactly_once_intr.REMOTE_2PC {
		s.tac = NewTransactionAwareConsumer(s.stream, s.epochMarkerSerde)
	} else if s.guarantee == exactly_once_intr.EPOCH_MARK {
		s.emc = NewEpochMarkConsumer(s.TopicName(), s.stream, s.epochMarkerSerde)
	} else if s.guarantee == exactly_once_intr.ALIGN_CHKPT {
		s.acc = NewAlignChkptConsumer(s.stream, s.epochMarkerSerde,
			s.numSrcProducer, s.instanceId)
	}
}

func (s *ShardedSharedLogStreamConsumer) Stream() sharedlog_stream.Stream {
	return s.stream
}

func (s *ShardedSharedLogStreamConsumer) SetName(name string) {
	s.name = name
}

func (s *ShardedSharedLogStreamConsumer) Name() string {
	return s.name
}

func (s *ShardedSharedLogStreamConsumer) TopicName() string {
	return s.stream.TopicName()
}

func (s *ShardedSharedLogStreamConsumer) SetCursor(cursor uint64, parNum uint8) {
	s.stream.SetCursor(cursor, parNum)
}

func (s *ShardedSharedLogStreamConsumer) readNext(ctx context.Context, parNum uint8) (*commtypes.RawMsg, error) {
	if s.guarantee == exactly_once_intr.TWO_PHASE_COMMIT || s.guarantee == exactly_once_intr.REMOTE_2PC {
		return s.tac.ReadNext(ctx, parNum)
	} else if s.guarantee == exactly_once_intr.EPOCH_MARK {
		return s.emc.ReadNext(ctx, parNum)
	} else if s.guarantee == exactly_once_intr.ALIGN_CHKPT {
		return s.acc.ReadNext(ctx, parNum)
	} else {
		rawMsg, err := s.stream.ReadNext(ctx, parNum)
		if err != nil {
			return nil, err
		}
		if rawMsg.IsControl {
			epochMark, err := s.epochMarkerSerde.Decode(rawMsg.Payload)
			if err != nil {
				return nil, err
			}
			// debug.Fprintf(os.Stderr, "%+v\n", epochMark)
			rawMsg.Mark = epochMark.Mark
			rawMsg.ProdIdx = epochMark.ProdIndex
			if epochMark.Mark == commtypes.SCALE_FENCE {
				rawMsg.ScaleEpoch = epochMark.ScaleEpoch
			} else if epochMark.Mark == commtypes.STREAM_END {
				rawMsg.StartTime = epochMark.StartTime
			}
		}
		return rawMsg, err
	}
}

func (s *ShardedSharedLogStreamConsumer) RecordCurrentConsumedSeqNum(seqNum uint64) {
	atomic.StoreUint64(&s.currentSeqNum, seqNum)
}

func (s *ShardedSharedLogStreamConsumer) CurrentConsumedSeqNum() uint64 {
	return atomic.LoadUint64(&s.currentSeqNum)
}

func (s *ShardedSharedLogStreamConsumer) Consume(ctx context.Context, parNum uint8) (*commtypes.RawMsgAndSeq, error) {
	startTime := time.Now()
L:
	for {
		select {
		case <-ctx.Done():
			break L
		default:
		}
		duration := time.Since(startTime)
		// debug.Fprintf(os.Stderr, "consume dur: %v\n", duration)
		if s.timeout != 0 && duration >= s.timeout {
			break
		}
		rawMsg, err := s.readNext(ctx, parNum)
		if err != nil {
			if common_errors.IsStreamEmptyError(err) {
				// debug.Fprintf(os.Stderr, "stream is empty\n")
				time.Sleep(time.Duration(100) * time.Microsecond)
				continue
			} else if common_errors.IsStreamTimeoutError(err) {
				// debug.Fprintf(os.Stderr, "stream time out\n")
				continue
			} else {
				return nil, err
			}
		}
		if len(rawMsg.Payload) == 0 || rawMsg.IsSyncToRecent {
			continue
		}
		if rawMsg.IsControl {
			return &commtypes.RawMsgAndSeq{
				Payload:           rawMsg.Payload,
				PayloadArr:        nil,
				AuxData:           rawMsg.AuxData,
				MsgSeqNum:         rawMsg.MsgSeqNum,
				LogSeqNum:         rawMsg.LogSeqNum,
				IsControl:         true,
				StartTime:         rawMsg.StartTime,
				Mark:              rawMsg.Mark,
				ScaleEpoch:        rawMsg.ScaleEpoch,
				ProdIdx:           rawMsg.ProdIdx,
				FirstChkptMarkSeq: rawMsg.FirstChkptMarkSeq,
				UnprocessSeq:      rawMsg.UnprocessSeq,
				InjTsMs:           rawMsg.InjTsMs,
			}, nil
		} else {
			if rawMsg.IsPayloadArr {
				payloadArr, err := s.payloadArrSerde.Decode(rawMsg.Payload)
				if err != nil {
					return nil, fmt.Errorf("fail to decode payload arr: %v", err)
				}
				return &commtypes.RawMsgAndSeq{
					Payload:    nil,
					PayloadArr: payloadArr.Payloads,
					AuxData:    rawMsg.AuxData,
					MsgSeqNum:  rawMsg.MsgSeqNum,
					LogSeqNum:  rawMsg.LogSeqNum,
					IsControl:  false,
					InjTsMs:    rawMsg.InjTsMs,
					ProdIdx:    rawMsg.ProdIdx,
				}, nil
			} else {
				return &commtypes.RawMsgAndSeq{
					Payload:    rawMsg.Payload,
					PayloadArr: nil,
					AuxData:    rawMsg.AuxData,
					MsgSeqNum:  rawMsg.MsgSeqNum,
					LogSeqNum:  rawMsg.LogSeqNum,
					IsControl:  false,
					InjTsMs:    rawMsg.InjTsMs,
					ProdIdx:    rawMsg.ProdIdx,
				}, nil
			}
		}
	}
	return nil, common_errors.ErrStreamSourceTimeout
}
