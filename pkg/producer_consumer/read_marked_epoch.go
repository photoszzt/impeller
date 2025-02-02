package producer_consumer

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/data_structure"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/sharedlog_stream"

	"github.com/gammazero/deque"
)

type MsgStatus uint8

const (
	NOT_MARK MsgStatus = iota
	MARKED
	SHOULD_DROP
)

type EpochMarkConsumer struct {
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker]
	stream           *sharedlog_stream.ShardedSharedLogStream
	marked           map[commtypes.ProducerId]map[uint8]commtypes.SeqRangeSet
	curReadMsgSeqNum map[commtypes.ProducerId]data_structure.Uint64Set
	msgBuffer        []*deque.Deque[*commtypes.RawMsg]
	// streamTime       stats.PrintLogStatsCollector[int64]
}

func NewEpochMarkConsumer(
	srcName string,
	stream *sharedlog_stream.ShardedSharedLogStream,
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker],
) *EpochMarkConsumer {
	msgBuffer := make([]*deque.Deque[*commtypes.RawMsg], stream.NumPartition())
	for i := uint8(0); i < stream.NumPartition(); i++ {
		msgBuffer[i] = deque.New[*commtypes.RawMsg]()
	}
	return &EpochMarkConsumer{
		epochMarkerSerde: epochMarkerSerde,
		stream:           stream,
		marked:           make(map[commtypes.ProducerId]map[uint8]commtypes.SeqRangeSet),
		msgBuffer:        msgBuffer,
		curReadMsgSeqNum: make(map[commtypes.ProducerId]data_structure.Uint64Set),
		// streamTime:       stats.NewPrintLogStatsCollector[int64]("streamTime" + srcName),
	}
}

func (emc *EpochMarkConsumer) OutputRemainingStats() {
	// emc.streamTime.PrintRemainingStats()
}

func (emc *EpochMarkConsumer) ReadNextMock(msgs []*commtypes.RawMsg, parNum uint8) (*commtypes.RawMsg, error) {
	msgQueue := emc.msgBuffer[parNum]
	if msgQueue.Len() != 0 {
		retMsg := emc.checkMsgQueue(msgQueue, parNum)
		if retMsg != nil {
			return retMsg, nil
		}
	}
	idx := 0
	for {
		if idx >= len(msgs) {
			return nil, common_errors.ErrStreamEmpty
		}
		rawMsg := msgs[idx]
		idx += 1
		// nowMs := time.Now().UnixMilli()
		// emc.streamTime.AddSample(nowMs - rawMsg.InjTsMs)

		// debug.Fprintf(os.Stderr, "RawMsg\n")
		// debug.Fprintf(os.Stderr, "\tPayload %v\n", string(rawMsg.Payload))
		// debug.Fprintf(os.Stderr, "\tLogSeq 0x%x\n", rawMsg.LogSeqNum)
		// debug.Fprintf(os.Stderr, "\tIsControl: %v\n", rawMsg.IsControl)
		// debug.Fprintf(os.Stderr, "\tIsPayloadArr: %v\n", rawMsg.IsPayloadArr)

		// if !rawMsg.IsControl {
		// 	debug.Fprintf(os.Stderr, "%s RawMsg: Payload %v, LogSeq 0x%x, MsgSeqNum 0x%x, IsControl: %v, IsPayloadArr: %v\n",
		// 		emc.stream.TopicName(), string(rawMsg.Payload), rawMsg.LogSeqNum, rawMsg.MsgSeqNum, rawMsg.IsControl, rawMsg.IsPayloadArr)
		// }

		// control entry's msgseqnum is written for the epoch log;
		// reading from the normal stream, we don't count this entry as a duplicate one
		// control entry itself is idempodent; n commit record with the same content is the
		// same as one commit record
		if !rawMsg.IsControl {
			if shouldIgnoreThisMsg(emc.curReadMsgSeqNum, rawMsg) {
				fmt.Fprintf(os.Stderr, "got a duplicate entry; continue\n")
				continue
			}
			// fmt.Fprintf(os.Stderr, "appending normalMsg logSeq: 0x%x\n", rawMsg.LogSeqNum)
		} else {
			epochMark, err := emc.epochMarkerSerde.Decode(rawMsg.Payload)
			if err != nil {
				return nil, err
			}
			// fmt.Fprintf(os.Stderr, "appending %+v, logSeq: 0x%x\n", epochMark, rawMsg.LogSeqNum)
			rawMsg.Mark = epochMark.Mark
			rawMsg.ProdIdx = epochMark.ProdIndex
			if epochMark.Mark == commtypes.EPOCH_END {
				// debug.Fprintf(os.Stderr, "%+v\n", epochMark)
				ranges, ok := emc.marked[rawMsg.ProdId]
				if !ok {
					ranges = make(map[uint8]commtypes.SeqRangeSet)
				}
				markRanges := epochMark.OutputRanges[emc.stream.TopicName()]
				for _, r := range markRanges {
					if _, ok := ranges[r.SubStreamNum]; !ok {
						ranges[r.SubStreamNum] = commtypes.NewSeqRangeSet()
					}
					ranges[r.SubStreamNum].Add(commtypes.SeqRange{
						Start: r.Start,
						End:   rawMsg.LogSeqNum,
					})
				}
				emc.marked[rawMsg.ProdId] = ranges
				rawMsg.MarkRanges = markRanges
			} else if epochMark.Mark == commtypes.SCALE_FENCE {
				rawMsg.ScaleEpoch = epochMark.ScaleEpoch
			} else if epochMark.Mark == commtypes.STREAM_END {
				rawMsg.StartTime = epochMark.StartTime
			} else if epochMark.Mark == commtypes.CHKPT_MARK {
				panic("checkpoint mark should not appear in epoch mark protocol")
			}
		}

		msgQueue.PushBack(rawMsg)
		retMsg := emc.checkMsgQueue(msgQueue, parNum)
		if retMsg != nil {
			return retMsg, nil
		}
	}
}

func (emc *EpochMarkConsumer) ReadNext(ctx context.Context, parNum uint8) (*commtypes.RawMsg, error) {
	msgQueue := emc.msgBuffer[parNum]
	if msgQueue.Len() != 0 {
		retMsg := emc.checkMsgQueue(msgQueue, parNum)
		if retMsg != nil {
			return retMsg, nil
		}
	}
	for {
		rawMsg, err := emc.stream.ReadNext(ctx, parNum)
		if err != nil {
			return nil, err
		}
		if rawMsg.IsSyncToRecent {
			continue
		}
		// nowMs := time.Now().UnixMilli()
		// emc.streamTime.AddSample(nowMs - rawMsg.InjTsMs)

		// debug.Fprintf(os.Stderr, "RawMsg\n")
		// debug.Fprintf(os.Stderr, "\tPayload %v\n", string(rawMsg.Payload))
		// debug.Fprintf(os.Stderr, "\tLogSeq 0x%x\n", rawMsg.LogSeqNum)
		// debug.Fprintf(os.Stderr, "\tIsControl: %v\n", rawMsg.IsControl)
		// debug.Fprintf(os.Stderr, "\tIsPayloadArr: %v\n", rawMsg.IsPayloadArr)

		// if !rawMsg.IsControl {
		// 	debug.Fprintf(os.Stderr, "%s RawMsg: Payload %v, LogSeq 0x%x, MsgSeqNum 0x%x, IsControl: %v, IsPayloadArr: %v\n",
		// 		emc.stream.TopicName(), string(rawMsg.Payload), rawMsg.LogSeqNum, rawMsg.MsgSeqNum, rawMsg.IsControl, rawMsg.IsPayloadArr)
		// }

		// control entry's msgseqnum is written for the epoch log;
		// reading from the normal stream, we don't count this entry as a duplicate one
		// control entry itself is idempodent; n commit record with the same content is the
		// same as one commit record
		if !rawMsg.IsControl {
			if shouldIgnoreThisMsg(emc.curReadMsgSeqNum, rawMsg) {
				fmt.Fprintf(os.Stderr, "got a duplicate entry; continue\n")
				continue
			}
			// fmt.Fprintf(os.Stderr, "appending normalMsg logSeq: 0x%x\n", rawMsg.LogSeqNum)
		} else {
			epochMark, err := emc.epochMarkerSerde.Decode(rawMsg.Payload)
			if err != nil {
				return nil, err
			}
			// fmt.Fprintf(os.Stderr, "appending %+v, logSeq: 0x%x\n", epochMark, rawMsg.LogSeqNum)
			rawMsg.Mark = epochMark.Mark
			rawMsg.ProdIdx = epochMark.ProdIndex
			if epochMark.Mark == commtypes.EPOCH_END {
				// debug.Fprintf(os.Stderr, "%+v\n", epochMark)
				ranges, ok := emc.marked[rawMsg.ProdId]
				if !ok {
					ranges = make(map[uint8]commtypes.SeqRangeSet)
				}
				markRanges := epochMark.OutputRanges[emc.stream.TopicName()]
				// fmt.Fprintf(os.Stderr, "Epoch markedRanges: %v\n", epochMark.OutputRanges)
				for _, r := range markRanges {
					if _, ok := ranges[r.SubStreamNum]; !ok {
						ranges[r.SubStreamNum] = commtypes.NewSeqRangeSet()
					}
					ranges[r.SubStreamNum].Add(commtypes.SeqRange{
						Start: r.Start,
						End:   rawMsg.LogSeqNum,
					})
				}
				emc.marked[rawMsg.ProdId] = ranges
				rawMsg.MarkRanges = markRanges
			} else if epochMark.Mark == commtypes.SCALE_FENCE {
				rawMsg.ScaleEpoch = epochMark.ScaleEpoch
			} else if epochMark.Mark == commtypes.STREAM_END {
				rawMsg.StartTime = epochMark.StartTime
			} else if epochMark.Mark == commtypes.CHKPT_MARK {
				panic("checkpoint mark should not appear in epoch mark protocol")
			}
		}

		msgQueue.PushBack(rawMsg)
		retMsg := emc.checkMsgQueue(msgQueue, parNum)
		if retMsg != nil {
			return retMsg, nil
		}
	}
}

func (emc *EpochMarkConsumer) checkMsg(msgQueue *deque.Deque[*commtypes.RawMsg], parNum uint8, frontMsg *commtypes.RawMsg) *commtypes.RawMsg {
	if frontMsg.IsControl && frontMsg.Mark == commtypes.EPOCH_END {
		ranges := emc.marked[frontMsg.ProdId]
		for _, r := range frontMsg.MarkRanges {
			ranges[parNum].Remove(commtypes.SeqRange{
				Start: r.Start,
				End:   frontMsg.LogSeqNum,
			})
		}
		msgQueue.PopFront()
		return frontMsg
	}
	if (frontMsg.Mark == commtypes.SCALE_FENCE && frontMsg.ScaleEpoch != 0) || frontMsg.Mark == commtypes.STREAM_END {
		msgQueue.PopFront()
		return frontMsg
	}
	return nil
}

func (emc *EpochMarkConsumer) checkMsgQueue(msgQueue *deque.Deque[*commtypes.RawMsg], parNum uint8) *commtypes.RawMsg {
	if msgQueue.Len() > 0 {
		frontMsg := msgQueue.Front()
		// fmt.Fprintf(os.Stderr, "frontMsgMeta: %s\n", frontMsg.FormatMsgMeta())
		readyMsg := emc.checkMsg(msgQueue, parNum, frontMsg)
		if readyMsg != nil {
			// fmt.Fprintf(os.Stderr, "returnMsg1: %s\n", readyMsg.FormatMsgMeta())
			return readyMsg
		}
		msgStatus := emc.checkMsgStatus(frontMsg, parNum)
		if msgStatus == MARKED {
			msgQueue.PopFront()
			// fmt.Fprintf(os.Stderr, "returnMsg2: %s\n", frontMsg.FormatMsgMeta())
			return frontMsg
		}
		if msgStatus == SHOULD_DROP {
			frontMsg = msgQueue.PopFront()
			fmt.Fprintf(os.Stderr, "dropMsg: %s\n", frontMsg.FormatMsgMeta())
			return nil
		}
	}
	return nil
}

func (emc *EpochMarkConsumer) checkMsgStatus(rawMsg *commtypes.RawMsg, parNum uint8) MsgStatus {
	ranges, ok := emc.marked[rawMsg.ProdId]
	if !ok || len(ranges[parNum]) == 0 {
		// fmt.Fprintf(os.Stderr, "no ranges for rawMsg %+v\n", rawMsg.FormatMsgMeta())
		return NOT_MARK
	}
	markedRange := ranges[parNum]
	// fmt.Fprintf(os.Stderr, "ranges: %v, marked ranges: %v, parNum: %d\n", ranges, markedRange, parNum)
	minStart := optional.None[uint64]()
	for r := range markedRange {
		if minStart.IsNone() {
			minStart = optional.Some(r.Start)
		} else {
			if r.Start < minStart.Unwrap() {
				minStart = optional.Some(r.Start)
			}
		}
		if rawMsg.LogSeqNum >= r.Start && rawMsg.LogSeqNum <= r.End {
			return MARKED
		}
	}
	debug.Assert(minStart.IsSome(), "should have a min start")
	mStart := minStart.Unwrap()
	if rawMsg.LogSeqNum < mStart {
		fmt.Fprintf(os.Stderr, "rawMsg logSeq %#x < markedRange start %#x, parnum %d\n",
			rawMsg.LogSeqNum, mStart, parNum)
		return SHOULD_DROP
	} else {
		return NOT_MARK
	}
}

func shouldIgnoreThisMsg(curReadMsgSeqNum map[commtypes.ProducerId]data_structure.Uint64Set, rawMsg *commtypes.RawMsg) bool {
	prodId := rawMsg.ProdId
	msgSeqNumSet, ok := curReadMsgSeqNum[prodId]
	if ok && msgSeqNumSet != nil && msgSeqNumSet.Has(rawMsg.MsgSeqNum) {
		return true
	}

	if msgSeqNumSet == nil {
		msgSeqNumSet = data_structure.NewUint64Set()
		curReadMsgSeqNum[prodId] = msgSeqNumSet
	}
	curReadMsgSeqNum[prodId].Add(rawMsg.MsgSeqNum)
	return false
}
