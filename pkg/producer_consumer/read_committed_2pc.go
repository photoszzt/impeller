package producer_consumer

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/data_structure"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/sharedlog_stream"

	"github.com/gammazero/deque"
	"github.com/rs/zerolog/log"
)

type LastMarkAndSeqRange struct {
	commit   commtypes.SeqRangeSet
	abort    commtypes.SeqRangeSet
	lastMark uint64
}

type TransactionAwareConsumer struct {
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker]
	stream           *sharedlog_stream.ShardedSharedLogStream
	marked           map[commtypes.ProducerId]map[uint8]LastMarkAndSeqRange
	// check whether producer produces duplicate records
	curReadMsgSeqNum map[commtypes.ProducerId]data_structure.Uint64Set
	msgBuffer        []*deque.Deque[*commtypes.RawMsg]
}

func NewTransactionAwareConsumer(stream *sharedlog_stream.ShardedSharedLogStream,
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker],
) *TransactionAwareConsumer {
	msgBuffer := make([]*deque.Deque[*commtypes.RawMsg], stream.NumPartition())
	for i := uint8(0); i < stream.NumPartition(); i++ {
		msgBuffer[i] = deque.New[*commtypes.RawMsg]()
	}
	return &TransactionAwareConsumer{
		msgBuffer:        msgBuffer,
		stream:           stream,
		epochMarkerSerde: epochMarkerSerde,
		marked:           make(map[commtypes.ProducerId]map[uint8]LastMarkAndSeqRange),
		curReadMsgSeqNum: make(map[commtypes.ProducerId]data_structure.Uint64Set),
	}
}

func (tac *TransactionAwareConsumer) checkMsgQueue(msgQueue *deque.Deque[*commtypes.RawMsg], parNum uint8) *commtypes.RawMsg {
	if msgQueue.Len() > 0 {
		frontMsg := msgQueue.Front()
		// fmt.Fprintf(os.Stderr, "frontMsgMeta: %s\n", frontMsg.FormatMsgMeta())
		if frontMsg.IsControl && frontMsg.Mark == commtypes.EPOCH_END {
			ranges := tac.marked[frontMsg.ProdId][parNum]
			r := commtypes.SeqRange{
				Start: frontMsg.MarkRanges[0].Start,
				End:   frontMsg.LogSeqNum,
			}
			ranges.commit.Remove(r)
			// debug.Fprintf(os.Stderr, "remove commit mark %s with range %#v\n", frontMsg.FormatMsgMeta(), r)
			msgQueue.PopFront()
			return frontMsg
		} else if frontMsg.IsControl && frontMsg.Mark == commtypes.ABORT {
			ranges := tac.marked[frontMsg.ProdId][parNum]
			ranges.abort.Remove(commtypes.SeqRange{
				Start: frontMsg.MarkRanges[0].Start,
				End:   frontMsg.LogSeqNum,
			})
			msgQueue.PopFront()
			return frontMsg
		}
		if frontMsg.Mark == commtypes.CHKPT_MARK {
			panic("checkpoint mark should not appear in 2pc protocol")
		}
		if (frontMsg.Mark == commtypes.SCALE_FENCE && frontMsg.ScaleEpoch != 0) || frontMsg.Mark == commtypes.STREAM_END {
			msgQueue.PopFront()
			return frontMsg
		}
		if tac.HasCommited(frontMsg, parNum) {
			msgQueue.PopFront()
			// fmt.Fprintf(os.Stderr, "returnMsg2: %s\n", frontMsg.FormatMsgMeta())
			return frontMsg
		}
		if tac.HasAborted(frontMsg, parNum) {
			msgQueue.PopFront()
			fmt.Fprintf(os.Stderr, "dropMsg: %s\n", frontMsg.FormatMsgMeta())
			return nil
		}
	}
	return nil
}

func (tac *TransactionAwareConsumer) ReadNext(ctx context.Context, parNum uint8) (*commtypes.RawMsg, error) {
	msgQueue := tac.msgBuffer[parNum]
	// debug.Fprintf(os.Stderr, "reading from sub %d, msg queue len: %d\n", parNum, msgQueue.Len())
	if msgQueue.Len() != 0 {
		retMsg := tac.checkMsgQueue(msgQueue, parNum)
		if retMsg != nil {
			// debug.Fprintf(os.Stderr, "return msg in check 1\n")
			return retMsg, nil
		}
	}
	ctxid := commtypes.GetCtxId(ctx)
	for {
		rawMsg, err := tac.stream.ReadNext(ctx, parNum)
		if err != nil {
			// debug.Fprintf(os.Stderr, "[ERROR] return err: %v\n", err)
			return nil, err
		}
		if rawMsg.IsSyncToRecent {
			continue
		}
		// debug.Fprintf(os.Stderr, "RawMsg\n")
		// debug.Fprintf(os.Stderr, "\tPayload %v\n", string(rawMsg.Payload))
		// debug.Fprintf(os.Stderr, "\tLogSeq 0x%x\n", rawMsg.LogSeqNum)
		// debug.Fprintf(os.Stderr, "\tIsControl: %v\n", rawMsg.IsControl)
		// debug.Fprintf(os.Stderr, "\tIsPayloadArr: %v\n", rawMsg.IsPayloadArr)

		// debug.Fprintf(os.Stderr, "%s RawMsg: LogSeq 0x%x, MsgSeqNum 0x%x, IsControl: %#v\n",
		// 	tac.stream.TopicName(), rawMsg.LogSeqNum, rawMsg.MsgSeqNum, rawMsg.IsControl)
		if !rawMsg.IsControl {
			if shouldIgnoreThisMsg(tac.curReadMsgSeqNum, rawMsg) {
				fmt.Fprintf(os.Stderr, "got a duplicate entry; continue\n")
				continue
			}
		} else {
			txnMark, err := tac.epochMarkerSerde.Decode(rawMsg.Payload)
			if err != nil {
				log.Err(err).Str("epoch marker payload", commtypes.StringFromImmutableBytes(rawMsg.Payload)).Msg("[ERROR] fail to decode epochMarker")
				return nil, err
			}
			if txnMark.Mark == commtypes.SCALE_FENCE || txnMark.Mark == commtypes.STREAM_END {
				debug.Fprintf(os.Stderr, "[%d, id=%s] Got %+v, logSeq: 0x%x\n", parNum, ctxid, txnMark, rawMsg.LogSeqNum)
			}
			if txnMark.Mark == commtypes.EPOCH_END {
				rangesAndLastMark := tac.createMarkedMapIfNotExists(rawMsg.ProdId, parNum)
				start := rangesAndLastMark[parNum].lastMark + 1
				seqRangeTmp := rangesAndLastMark[parNum]
				r := commtypes.SeqRange{
					Start: start,
					End:   rawMsg.LogSeqNum,
				}
				// debug.Fprintf(os.Stderr, "appending %+v, producer %#v, logSeq: 0x%x, range: %#v\n",
				// 	txnMark.Mark, rawMsg.ProdId, rawMsg.LogSeqNum, r)
				seqRangeTmp.commit.Add(r)
				seqRangeTmp.lastMark = rawMsg.LogSeqNum
				rangesAndLastMark[parNum] = seqRangeTmp
				tac.marked[rawMsg.ProdId] = rangesAndLastMark
				rawMsg.Mark = commtypes.EPOCH_END
				rawMsg.MarkRanges = []commtypes.ProduceRange{
					{Start: start},
				}
			} else if txnMark.Mark == commtypes.ABORT {
				rangesAndLastMark := tac.createMarkedMapIfNotExists(rawMsg.ProdId, parNum)
				lastMark := rangesAndLastMark[parNum].lastMark
				seqRangeTmp := rangesAndLastMark[parNum]
				r := commtypes.SeqRange{
					Start: lastMark + 1,
					End:   rawMsg.LogSeqNum,
				}
				seqRangeTmp.abort.Add(r)
				// debug.Fprintf(os.Stderr, "appending %+v, producer %#v, logSeq: 0x%x, range: %#v\n",
				// 	txnMark.Mark, rawMsg.ProdId, rawMsg.LogSeqNum, r)
				seqRangeTmp.lastMark = rawMsg.LogSeqNum
				rangesAndLastMark[parNum] = seqRangeTmp
				tac.marked[rawMsg.ProdId] = rangesAndLastMark
				rawMsg.MarkRanges = []commtypes.ProduceRange{
					{Start: lastMark + 1},
				}
				rawMsg.Mark = commtypes.ABORT
			} else if txnMark.Mark == commtypes.SCALE_FENCE {
				rawMsg.ScaleEpoch = txnMark.ScaleEpoch
				rawMsg.ProdIdx = txnMark.ProdIndex
				rawMsg.Mark = txnMark.Mark
			} else if txnMark.Mark == commtypes.STREAM_END {
				rawMsg.Mark = txnMark.Mark
				rawMsg.StartTime = txnMark.StartTime
				rawMsg.ProdIdx = txnMark.ProdIndex
			}
		}
		msgQueue.PushBack(rawMsg)
		retMsg := tac.checkMsgQueue(msgQueue, parNum)
		if retMsg != nil {
			// debug.Fprintf(os.Stderr, "return msg in check 2\n")
			return retMsg, nil
		}
	}
}

func (tac *TransactionAwareConsumer) createMarkedMapIfNotExists(prodId commtypes.ProducerId, parNum uint8) map[uint8]LastMarkAndSeqRange {
	rangesAndLastMark, ok := tac.marked[prodId]
	if !ok {
		rangesAndLastMark = make(map[uint8]LastMarkAndSeqRange)
	}
	if _, ok := rangesAndLastMark[parNum]; !ok {
		rangesAndLastMark[parNum] = LastMarkAndSeqRange{
			lastMark: 0,
			commit:   commtypes.NewSeqRangeSet(),
			abort:    commtypes.NewSeqRangeSet(),
		}
	}
	return rangesAndLastMark
}

func (tac *TransactionAwareConsumer) HasCommited(rawMsg *commtypes.RawMsg, parNum uint8) bool {
	ranges, ok := tac.marked[rawMsg.ProdId]
	if !ok {
		return false
	}
	// debug.Fprintf(os.Stderr, "committed ranges: %+v\n", ranges[parNum].commit)
	for r := range ranges[parNum].commit {
		if rawMsg.LogSeqNum >= r.Start && rawMsg.LogSeqNum <= r.End {
			return true
		}
	}
	return false
}

func (tac *TransactionAwareConsumer) HasAborted(rawMsg *commtypes.RawMsg, parNum uint8) bool {
	ranges, ok := tac.marked[rawMsg.ProdId]
	if !ok {
		return false
	}
	for r := range ranges[parNum].abort {
		if rawMsg.LogSeqNum >= r.Start && rawMsg.LogSeqNum <= r.End {
			return true
		}
	}
	return false
}
