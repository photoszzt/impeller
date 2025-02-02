package producer_consumer

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/data_structure"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stats"
	"time"

	"github.com/gammazero/deque"
)

type AlignChkptConsumer struct {
	stream                 *sharedlog_stream.ShardedSharedLogStream
	curReadMsgSeqNum       map[commtypes.ProducerId]data_structure.Uint64Set
	epochMarkerSerde       commtypes.SerdeG[commtypes.EpochMarker]
	msgBuffer              []*deque.Deque[*commtypes.RawMsg]
	producerMarked         data_structure.Uint8Set
	alignTime              stats.StatsCollector[int64]
	alignStart             time.Time
	currentSnapshotId      int64
	firstChkptMarkerSeqNum uint64
	numSrcProducer         uint8
	instanceId             uint8
}

func NewAlignChkptConsumer(
	stream *sharedlog_stream.ShardedSharedLogStream,
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker],
	numSrcProducer uint8,
	instanceId uint8,
) *AlignChkptConsumer {
	msgBuffer := make([]*deque.Deque[*commtypes.RawMsg], stream.NumPartition())
	for i := uint8(0); i < stream.NumPartition(); i++ {
		msgBuffer[i] = deque.New[*commtypes.RawMsg]()
	}
	return &AlignChkptConsumer{
		stream:           stream,
		curReadMsgSeqNum: make(map[commtypes.ProducerId]data_structure.Uint64Set),
		epochMarkerSerde: epochMarkerSerde,
		msgBuffer:        msgBuffer,
		producerMarked:   make(data_structure.Uint8Set),
		alignTime: stats.NewStatsCollector[int64](fmt.Sprintf("WaitAlignTime_%d(ms)", instanceId),
			stats.DEFAULT_COLLECT_DURATION),
		currentSnapshotId: 0,
		numSrcProducer:    numSrcProducer,
		instanceId:        instanceId,
	}
}

func (ndc *AlignChkptConsumer) OutputRemainingStats() {
	ndc.alignTime.PrintRemainingStats()
}

func (ndc *AlignChkptConsumer) ReadNext(ctx context.Context, parNum uint8) (*commtypes.RawMsg, error) {
	msgQueue := ndc.msgBuffer[parNum]
	if msgQueue.Len() != 0 {
		msg := msgQueue.PopFront()
		if !msg.IsControl || msg.Mark == commtypes.STREAM_END || msg.Mark == commtypes.SCALE_FENCE {
			return msg, nil
		} else {
			panic("TODO: handle nested checkpoint")
		}
	}
	for {
		rawMsg, err := ndc.stream.ReadNext(ctx, parNum)
		if err != nil {
			return nil, err
		}
		if rawMsg.IsSyncToRecent {
			continue
		}
		if rawMsg.IsControl {
			epochMark, err := ndc.epochMarkerSerde.Decode(rawMsg.Payload)
			if err != nil {
				return nil, err
			}
			// debug.Fprintf(os.Stderr, "%+v\n", epochMark)
			rawMsg.Mark = epochMark.Mark
			rawMsg.ProdIdx = epochMark.ProdIndex
			if epochMark.Mark == commtypes.SCALE_FENCE {
				rawMsg.ScaleEpoch = epochMark.ScaleEpoch
				if len(ndc.producerMarked) != 0 && ndc.producerMarked.Has(rawMsg.ProdIdx) {
					msgQueue.PushBack(rawMsg)
					continue
				}
				return rawMsg, nil
			} else if epochMark.Mark == commtypes.STREAM_END {
				rawMsg.StartTime = epochMark.StartTime
				if len(ndc.producerMarked) != 0 && ndc.producerMarked.Has(rawMsg.ProdIdx) {
					msgQueue.PushBack(rawMsg)
					continue
				}
				return rawMsg, nil
			} else if epochMark.Mark == commtypes.CHKPT_MARK {
				if ndc.producerMarked.Has(epochMark.ProdIndex) {
					panic("got another epoch mark while current epoch hasn't finished")
					// msgQueue.PushBack(rawMsg)
					// continue
				}
				if len(ndc.producerMarked) == 0 {
					ndc.firstChkptMarkerSeqNum = rawMsg.LogSeqNum
					ndc.alignStart = time.Now()
				}
				ndc.producerMarked.Add(epochMark.ProdIndex)
				if len(ndc.producerMarked) == int(ndc.numSrcProducer) {
					ndc.producerMarked = make(data_structure.Uint8Set)
					var unprocessed []uint64
					for i := 0; i < msgQueue.Len(); i++ {
						msg := msgQueue.At(i)
						unprocessed = append(unprocessed, msg.LogSeqNum)
					}
					rawMsg.UnprocessSeq = unprocessed
					rawMsg.FirstChkptMarkSeq = ndc.firstChkptMarkerSeqNum
					ndc.alignTime.AddSample(time.Since(ndc.alignStart).Milliseconds())
					// got all checkpoint marker
					return rawMsg, nil
				} else {
					continue
				}
			}
		} else {
			if shouldIgnoreThisMsg(ndc.curReadMsgSeqNum, rawMsg) {
				fmt.Fprintf(os.Stderr, "got a duplicate entry; continue\n")
				continue
			}
			if len(ndc.producerMarked) != 0 && ndc.producerMarked.Has(rawMsg.ProdIdx) {
				msgQueue.PushBack(rawMsg)
				continue
			}
			return rawMsg, err
		}
	}
}
