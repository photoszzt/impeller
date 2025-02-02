//go:generate msgp
//msgp:ignore SharedLogStream
package sharedlog_stream

import (
	"context"
	"fmt"
	"sharedlog-stream/pkg/bits"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils/syncutils"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

	"cs.utexas.edu/zjia/faas/protocol"
	"cs.utexas.edu/zjia/faas/types"
)

const (
	kBlockingReadTimeout = 5 * time.Millisecond
)

type LogEntryMeta uint8

const (
	Control bits.Bits = 1 << iota
	PayloadArr
	SyncToRecent
)

func StreamEntryMeta(isControl bool, payloadIsArr bool) LogEntryMeta {
	var meta bits.Bits
	if isControl {
		meta = bits.Set(meta, Control)
	}
	if payloadIsArr {
		meta = bits.Set(meta, PayloadArr)
	}
	return LogEntryMeta(meta)
}

func SyncToRecentMeta() LogEntryMeta {
	var meta bits.Bits
	meta = bits.Set(meta, SyncToRecent)
	return LogEntryMeta(meta)
}

var (
	SingleDataRecordMeta = StreamEntryMeta(false, false)
	ControlRecordMeta    = StreamEntryMeta(true, false)
	ArrRecordMeta        = StreamEntryMeta(false, true)
)

type SharedLogStream struct {
	mux syncutils.RWMutex
	// txnMarkerSerde commtypes.Serde
	topicName     string
	topicNameHash uint64
	// current read position in forward direction
	cursor             uint64
	tail               uint64 // protected by mux
	curAppendMsgSeqNum atomic.Uint64
}

func NameHashWithPartition(nameHash uint64, par uint8) uint64 {
	return (nameHash & txn_data.PartitionMask) + uint64(par)
}

type StreamLogEntry struct {
	TopicName []string `msg:"topicName"`
	Payload   []byte   `msg:"payload,omitempty"`
	InjTsMs   int64    `msg:"injTsMs,omitempty"`
	TaskId    uint64   `msg:"tid,omitempty"`
	MsgSeqNum uint64   `msg:"mseq,omitempty"`
	TaskEpoch uint32   `msg:"te,omitempty"`
	Meta      uint8    `msg:"meta,omitempty"`
}

func (e *StreamLogEntry) BelongsToTopic(topicName string) bool {
	// for common case where an entry only belongs to one topic
	if e.TopicName[0] == topicName {
		return true
	} else {
		// for mark record, it can be read by multiple topics
		for i := 1; i < len(e.TopicName); i++ {
			if e.TopicName[i] == topicName {
				return true
			}
		}
		return false
	}
}

func DecodeStreamLogEntry(logEntry *types.LogEntry) *StreamLogEntry {
	streamLogEntry := StreamLogEntry{}
	_, err := streamLogEntry.UnmarshalMsg(logEntry.Data)
	if err != nil {
		panic(err)
	}
	return &streamLogEntry
}

func NewSharedLogStream(topicName string, serdeFormat commtypes.SerdeFormat) (*SharedLogStream, error) {
	s := &SharedLogStream{
		topicName:     topicName,
		topicNameHash: hashfuncs.NameHash(topicName),
		cursor:        0,
		tail:          0,
	}
	s.curAppendMsgSeqNum.Store(0)
	return s, nil
}

func (s *SharedLogStream) NumPartition() uint8 {
	return 1
}

func (s *SharedLogStream) SetAppendMsgSeqNum(val uint64) {
	s.curAppendMsgSeqNum.Store(val)
}

func (s *SharedLogStream) TopicNameHash() uint64 {
	return s.topicNameHash
}

func (s *SharedLogStream) TopicName() string {
	return s.topicName
}

func (s *SharedLogStream) SetCursor(cursor uint64, parNum uint8) {
	s.cursor = cursor
}

func (s *SharedLogStream) GetCursor() uint64 {
	return s.cursor
}

// multiple thread could push to the same stream but only one reader
func (s *SharedLogStream) PushWithTag(ctx context.Context,
	payload []byte, parNum uint8, tags []uint64, additionalTopicNames []string,
	meta LogEntryMeta, producerId commtypes.ProducerId,
) (uint64, error) {
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	topics := []string{s.topicName}
	if additionalTopicNames != nil {
		topics = append(topics, additionalTopicNames...)
	}
	nowMs := time.Now().UnixMilli()
	logEntry := &StreamLogEntry{
		TopicName: topics,
		Payload:   payload,
		Meta:      uint8(meta),
		TaskId:    producerId.TaskId,
		TaskEpoch: producerId.TaskEpoch,
		InjTsMs:   nowMs,
	}
	// TODO: need to deal with sequence number overflow
	msgSeq := s.curAppendMsgSeqNum.Add(1)
	logEntry.MsgSeqNum = msgSeq
	// b := commtypes.PopBuffer(logEntry.Msgsize())
	// buf := *b
	// encoded, err := logEntry.MarshalMsg(buf[:0])
	encoded, err := logEntry.MarshalMsg(nil)
	if err != nil {
		return 0, err
	}

	s.mux.Lock()
	seqNum, err := env.SharedLogAppend(ctx, tags, encoded)
	if err != nil {
		// commtypes.PushBuffer(&encoded)
		return 0, err
	}
	s.tail = seqNum + 1
	s.mux.Unlock()

	// *b = encoded
	// commtypes.PushBuffer(b)
	// verify that push is successful

	// debug.Fprintf(os.Stderr, "push to %s tag: ", s.topicName)
	// for _, t := range tags {
	// 	debug.Fprintf(os.Stderr, "0x%x ", t)
	// }
	// debug.Fprintf(os.Stderr, "seqNum 0x%x, tail 0x%x\n", seqNum, s.tail)
	/*
		// debug.Fprintf(os.Stderr, "append val %s with tag: %x to topic %s par %d, seqNum: %x\n",
		// 	string(payload), tags[0], s.topicName, parNum, seqNum)
		logEntryRead, err := env.SharedLogReadNext(ctx, tags[0], seqNum)
		if err != nil {
			return 0, err
		}
		if logEntryRead == nil || logEntryRead.SeqNum != seqNum {
			return 0, fmt.Errorf("fail to read the log just appended")
		}
		if !bytes.Equal(encoded, logEntryRead.Data) {
			return 0, fmt.Errorf("log data mismatch")
		}

		logEntryRead, err = env.SharedLogReadPrev(ctx, tags[0], seqNum+1)
		if err != nil {
			return 0, err
		}
		if logEntryRead == nil || logEntryRead.SeqNum != seqNum {
			return 0, fmt.Errorf("fail to read the log just appended from backward")
		}

		if !bytes.Equal(encoded, logEntryRead.Data) {
			return 0, fmt.Errorf("log data mismatch")
		}
	*/

	return seqNum, nil
}

func (s *SharedLogStream) Push(ctx context.Context, payload []byte, parNum uint8, meta LogEntryMeta, producerId commtypes.ProducerId,
) (uint64, error) {
	tags := []uint64{NameHashWithPartition(s.topicNameHash, parNum)}
	return s.PushWithTag(ctx, payload, parNum, tags, nil, meta, producerId)
}

func (s *SharedLogStream) isEmpty() bool {
	s.mux.RLock()
	ret := s.cursor >= s.tail
	s.mux.RUnlock()
	return ret
}

func (s *SharedLogStream) ReadBackwardWithTag(ctx context.Context, tailSeqNum uint64, parNum uint8, tag uint64) (*commtypes.RawMsg, error) {
	seqNum := tailSeqNum
	for {
		logEntry, err := s.readPrevWithTimeout(ctx, tag, seqNum)
		if err != nil {
			return nil, err
		}
		if logEntry == nil {
			return nil, common_errors.ErrStreamEmpty
		}
		seqNum = logEntry.SeqNum
		streamLogEntry := DecodeStreamLogEntry(logEntry)
		if !streamLogEntry.BelongsToTopic(s.topicName) {
			continue
		} else {
			isControl := bits.Has(bits.Bits(streamLogEntry.Meta), Control)
			isPayloadArr := bits.Has(bits.Bits(streamLogEntry.Meta), PayloadArr)
			isSyncToRecent := bits.Has(bits.Bits(streamLogEntry.Meta), SyncToRecent)
			return &commtypes.RawMsg{
				Payload:        streamLogEntry.Payload,
				AuxData:        logEntry.AuxData,
				MsgSeqNum:      streamLogEntry.MsgSeqNum,
				LogSeqNum:      logEntry.SeqNum,
				IsControl:      isControl,
				IsPayloadArr:   isPayloadArr,
				IsSyncToRecent: isSyncToRecent,
				ProdId: commtypes.ProducerId{
					TaskId:    streamLogEntry.TaskId,
					TaskEpoch: streamLogEntry.TaskEpoch,
				},
			}, nil
		}
	}
}

func (s *SharedLogStream) ReadNext(ctx context.Context, parNum uint8) (*commtypes.RawMsg, error) {
	tag := NameHashWithPartition(s.topicNameHash, parNum)
	return s.ReadNextWithTag(ctx, parNum, tag)
}

// here the maxSeqNum is the seqNum of the most recent appended log entry
func (s *SharedLogStream) ReadNextWithTagUntil(ctx context.Context, parNum uint8, tag uint64, maxSeqNum uint64) ([]commtypes.RawMsg, error) {
	if s.tail < maxSeqNum {
		s.mux.Lock()
		s.tail = maxSeqNum + 1
		s.mux.Unlock()
	}
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	seqNum := s.cursor
	logEntries := make([]commtypes.RawMsg, 0, 3)
	for seqNum <= maxSeqNum {
		newCtx, cancel := context.WithTimeout(ctx, kBlockingReadTimeout)
		defer cancel()
		logEntry, err := env.SharedLogReadNextBlock(newCtx, tag, seqNum)
		// debug.Fprintf(os.Stderr, "after read next block\n")
		if err != nil {
			return nil, err
		}
		if logEntry == nil {
			return logEntries, nil
		}
		streamLogEntry := DecodeStreamLogEntry(logEntry)
		isControl := bits.Has(bits.Bits(streamLogEntry.Meta), Control)
		isSyncToRecent := bits.Has(bits.Bits(streamLogEntry.Meta), SyncToRecent)
		if streamLogEntry.BelongsToTopic(s.topicName) || isControl {
			isPayloadArr := bits.Has(bits.Bits(streamLogEntry.Meta), PayloadArr)
			s.cursor = logEntry.SeqNum + 1
			logEntries = append(logEntries, commtypes.RawMsg{
				Payload:        streamLogEntry.Payload,
				AuxData:        logEntry.AuxData,
				MsgSeqNum:      streamLogEntry.MsgSeqNum,
				LogSeqNum:      logEntry.SeqNum,
				IsControl:      isControl,
				IsPayloadArr:   isPayloadArr,
				IsSyncToRecent: isSyncToRecent,
				ProdId: commtypes.ProducerId{
					TaskId:    streamLogEntry.TaskId,
					TaskEpoch: streamLogEntry.TaskEpoch,
				},
				InjTsMs: streamLogEntry.InjTsMs,
			})
		}
		seqNum = logEntry.SeqNum + 1
		s.cursor = seqNum
	}
	return logEntries, nil
}

// ReadNextWithTag reads the log entry with the given tag from seqNum produced by prodId
func (s *SharedLogStream) ReadFromSeqNumWithTag(ctx context.Context, from uint64, appendedSeq uint64,
	parNum uint8, tag uint64, prodId commtypes.ProducerId,
) (*commtypes.RawMsg, error) {
	// fmt.Fprintf(os.Stderr, "ReadFromSeqNumWithTag %s[%d] tag %#x, from %#x, prodId %s\n",
	// 	s.topicName, parNum, tag, from, prodId.String())
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	for from <= appendedSeq {
		newCtx, cancel := context.WithTimeout(ctx, kBlockingReadTimeout)
		defer cancel()
		logEntry, err := env.SharedLogReadNextBlock(newCtx, tag, from)
		if err != nil {
			return nil, err
		}
		if logEntry == nil {
			continue
		}
		streamLogEntry := DecodeStreamLogEntry(logEntry)
		// fmt.Fprintf(os.Stderr, "logEntry %#x, tp %v, taskId %#x, taskEpoch %#x\n",
		// 	logEntry.SeqNum, streamLogEntry.TopicName, streamLogEntry.TaskId, streamLogEntry.TaskEpoch)
		if streamLogEntry.BelongsToTopic(s.topicName) &&
			streamLogEntry.TaskId == prodId.TaskId &&
			streamLogEntry.TaskEpoch == prodId.TaskEpoch {

			isPayloadArr := bits.Has(bits.Bits(streamLogEntry.Meta), PayloadArr)
			isControl := bits.Has(bits.Bits(streamLogEntry.Meta), Control)
			isSyncToRecent := bits.Has(bits.Bits(streamLogEntry.Meta), SyncToRecent)
			return &commtypes.RawMsg{
				Payload:        streamLogEntry.Payload,
				AuxData:        logEntry.AuxData,
				MsgSeqNum:      streamLogEntry.MsgSeqNum,
				LogSeqNum:      logEntry.SeqNum,
				IsControl:      isControl,
				IsPayloadArr:   isPayloadArr,
				IsSyncToRecent: isSyncToRecent,
				ProdId: commtypes.ProducerId{
					TaskId:    streamLogEntry.TaskId,
					TaskEpoch: streamLogEntry.TaskEpoch,
				},
				InjTsMs: streamLogEntry.InjTsMs,
			}, nil
		}
		from = logEntry.SeqNum + 1
	}
	return nil, fmt.Errorf("can't find log entry %s[%d] from %#x to %#x", s.topicName, parNum, from, appendedSeq)
}

func (s *SharedLogStream) ReadNextWithTag(ctx context.Context, parNum uint8, tag uint64) (*commtypes.RawMsg, error) {
	// debug.Fprintf(os.Stderr, "read topic %s with parNum %d tag %x\n", s.topicName, parNum, tag)
	if s.isEmpty() {
		if err := s.findLastEntryBackward(ctx, protocol.MaxLogSeqnum, parNum); err != nil {
			return nil, err
		}
		if s.isEmpty() {
			return nil, common_errors.ErrStreamEmpty
		}
	}
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	seqNumInSharedLog := s.cursor
	// debug.Fprintf(os.Stderr, "cursor: 0x%x, tail: 0x%x\n", s.cursor, s.tail)
	for seqNumInSharedLog < s.tail {
		// debug.Fprintf(os.Stderr, "read tag: 0x%x, seqNum: 0x%x, tail: 0x%x\n",
		// 	tag, seqNumInSharedLog, s.tail)
		newCtx, cancel := context.WithTimeout(ctx, kBlockingReadTimeout)
		defer cancel()
		logEntry, err := env.SharedLogReadNextBlock(newCtx, tag, seqNumInSharedLog)
		// debug.Fprintf(os.Stderr, "after read next block\n")
		if err != nil {
			return nil, err
		}
		if logEntry == nil {
			return nil, common_errors.ErrStreamEmpty
		}
		streamLogEntry := DecodeStreamLogEntry(logEntry)
		isControl := bits.Has(bits.Bits(streamLogEntry.Meta), Control)
		isSyncToRecent := bits.Has(bits.Bits(streamLogEntry.Meta), SyncToRecent)
		if streamLogEntry.BelongsToTopic(s.topicName) || isControl {
			isPayloadArr := bits.Has(bits.Bits(streamLogEntry.Meta), PayloadArr)
			s.cursor = logEntry.SeqNum + 1
			return &commtypes.RawMsg{
				Payload:        streamLogEntry.Payload,
				AuxData:        logEntry.AuxData,
				MsgSeqNum:      streamLogEntry.MsgSeqNum,
				LogSeqNum:      logEntry.SeqNum,
				IsControl:      isControl,
				IsPayloadArr:   isPayloadArr,
				IsSyncToRecent: isSyncToRecent,
				ProdId: commtypes.ProducerId{
					TaskId:    streamLogEntry.TaskId,
					TaskEpoch: streamLogEntry.TaskEpoch,
				},
				InjTsMs: streamLogEntry.InjTsMs,
			}, nil
		}
		seqNumInSharedLog = logEntry.SeqNum + 1
		s.cursor = seqNumInSharedLog
	}
	return nil, common_errors.ErrStreamEmpty
}

func (s *SharedLogStream) readPrevWithTimeout(ctx context.Context, tag uint64, seqNum uint64) (*types.LogEntry, error) {
	maxRetryTimes := 100
	idx := 0
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	for {
		newCtx, cancel := context.WithTimeout(ctx, kBlockingReadTimeout)
		defer cancel()
		logEntry, err := env.SharedLogReadPrev(newCtx, tag, seqNum)
		if err != nil {
			return nil, err
		}
		if logEntry != nil {
			return logEntry, nil
		} else {
			idx += 1
			if idx >= maxRetryTimes {
				return logEntry, nil
			}
			time.Sleep(100 * time.Microsecond)
		}
	}
}

func (s *SharedLogStream) findLastEntryBackward(ctx context.Context, tailSeqNum uint64, parNum uint8) error {
	if tailSeqNum < s.cursor {
		log.Fatal().
			Uint64("Current seq", s.cursor).
			Uint64("Request seq", tailSeqNum).
			Msg("Cannot sync to request")
		return fmt.Errorf("cannot sync to request")
	}

	/*
		if tailSeqNum == s.cursor+1 {
			return nil
		}
	*/

	tag := NameHashWithPartition(s.topicNameHash, parNum)

	seqNum := tailSeqNum
	// debug.Fprintf(os.Stderr, "find tail for topic: %s, par: %d\n", s.topicName, parNum)
	for seqNum >= s.cursor {
		logEntry, err := s.readPrevWithTimeout(ctx, tag, seqNum)
		if err != nil {
			return err
		}

		if logEntry == nil || logEntry.SeqNum < s.cursor {
			// we are already at the tail
			break
		}
		seqNum = logEntry.SeqNum
		streamLogEntry := DecodeStreamLogEntry(logEntry)
		if !streamLogEntry.BelongsToTopic(s.topicName) {
			seqNum -= 1
			continue
		}
		s.mux.Lock()
		s.tail = logEntry.SeqNum + 1
		s.mux.Unlock()
		// debug.Fprintf(os.Stderr, "%s(%d) current tail is %x\n", s.topicName, parNum, s.tail)
		break
	}
	return nil
}
