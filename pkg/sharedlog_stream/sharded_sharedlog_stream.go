package sharedlog_stream

import (
	"context"
	"fmt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type PartitionFunc func(interface{}) uint32

type SizableShardedSharedLogStream struct {
	topicName string

	mux                 sync.RWMutex // this mux is for modify number of entries of subSharedLogStreams
	subSharedLogStreams []*BufferedSinkStream
	numPartitions       uint8
	serdeFormat         commtypes.SerdeFormat
	bufMaxSize          uint32
}

func NewSizableShardedSharedLogStream(topicName string, numPartitions uint8,
	serdeFormat commtypes.SerdeFormat, bufMaxSize uint32,
) (*SizableShardedSharedLogStream, error) {
	if numPartitions == 0 {
		panic(ErrZeroParNum)
	}
	streams := make([]*BufferedSinkStream, 0, numPartitions)
	for i := uint8(0); i < numPartitions; i++ {
		s, err := NewSharedLogStream(topicName, serdeFormat)
		if err != nil {
			return nil, err
		}
		buf := NewBufferedSinkStream(s, i, bufMaxSize)
		streams = append(streams, buf)
	}
	return &SizableShardedSharedLogStream{
		subSharedLogStreams: streams,
		numPartitions:       numPartitions,
		topicName:           topicName,
		serdeFormat:         serdeFormat,
		bufMaxSize:          bufMaxSize,
	}, nil
}

func (s *SizableShardedSharedLogStream) SetLastMarkerSeq(lastMarkerSeq uint64) {
	for _, stream := range s.subSharedLogStreams {
		stream.SetLastMarkerSeq(lastMarkerSeq)
	}
}

func (s *SizableShardedSharedLogStream) NumPartition() uint8 {
	s.mux.RLock()
	ret := s.numPartitions
	s.mux.RUnlock()
	return ret
}

func (s *SizableShardedSharedLogStream) Push(ctx context.Context, payload []byte, parNum uint8,
	meta LogEntryMeta, producerId commtypes.ProducerId,
) (uint64, error) {
	s.mux.RLock()
	off, err := s.subSharedLogStreams[parNum].Push(ctx, payload, parNum, meta, producerId)
	s.mux.RUnlock()
	return off, err
}

func (s *SizableShardedSharedLogStream) BufPush(ctx context.Context, payload []byte, parNum uint8,
	producerId commtypes.ProducerId, flushCallback exactly_once_intr.FlushCallbackFunc,
) error {
	// idTmp := ctx.Value(commtypes.CTXID{})
	// id := ""
	// if idTmp != nil {
	// 	id = idTmp.(string)
	// }
	// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) before bufpush\n", id, s.topicName, parNum)
	s.mux.RLock()
	err := s.subSharedLogStreams[parNum].BufPushAutoFlushGoroutineSafe(ctx, payload, producerId, flushCallback)
	s.mux.RUnlock()
	// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) after bufpush\n", id, s.topicName, parNum)
	return err
}

func (s *SizableShardedSharedLogStream) Flush(ctx context.Context,
	producerId commtypes.ProducerId,
) (uint32, error) {
	// idTmp := ctx.Value(commtypes.CTXID{})
	// id := ""
	// if idTmp != nil {
	// 	id = idTmp.(string)
	// }
	flushed := uint32(0)
	s.mux.RLock()
	for i := uint8(0); i < s.numPartitions; i++ {
		// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) before flush\n", id, s.topicName, i)
		f, err := s.subSharedLogStreams[i].FlushGoroutineSafe(ctx, producerId)
		if err != nil {
			s.mux.RUnlock()
			return 0, err
		}
		flushed += f
		// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) after flush\n", id, s.topicName, i)
	}
	s.mux.RUnlock()
	return flushed, nil
}

func (s *SizableShardedSharedLogStream) BufPushNoLock(ctx context.Context, payload []byte,
	parNum uint8, producerId commtypes.ProducerId, flushCallback exactly_once_intr.FlushCallbackFunc,
) error {
	s.mux.RLock()
	err := s.subSharedLogStreams[parNum].BufPushAutoFlushNoLock(ctx, payload, producerId, flushCallback)
	s.mux.RUnlock()
	return err
}

func (s *SizableShardedSharedLogStream) FlushNoLock(ctx context.Context,
	producerId commtypes.ProducerId,
) (uint32, error) {
	s.mux.RLock()
	flushed := uint32(0)
	for i := uint8(0); i < s.numPartitions; i++ {
		f, err := s.subSharedLogStreams[i].FlushNoLock(ctx, producerId)
		if err != nil {
			s.mux.RLock()
			return 0, err
		}
		flushed += f
	}
	s.mux.RLock()
	return 0, nil
}

func (s *SizableShardedSharedLogStream) PushWithTag(ctx context.Context, payload []byte, parNum uint8, tags []uint64,
	additionalTopic []string, meta LogEntryMeta, producerId commtypes.ProducerId,
) (uint64, error) {
	s.mux.RLock()
	off, err := s.subSharedLogStreams[parNum].PushWithTag(ctx, payload, parNum, tags,
		additionalTopic, meta, producerId)
	s.mux.RUnlock()
	return off, err
}

func (s *SizableShardedSharedLogStream) ReadNext(ctx context.Context, parNumber uint8) (*commtypes.RawMsg, error) {
	s.mux.RLock()
	if parNumber >= s.numPartitions {
		s.mux.RUnlock()
		return nil, xerrors.Errorf("Invalid partition number: %d", parNumber)
	}
	par := parNumber
	shard := s.subSharedLogStreams[par]
	msg, err := shard.Stream.ReadNext(ctx, parNumber)
	s.mux.RUnlock()
	return msg, err
}

func (s *SizableShardedSharedLogStream) ReadNextWithTag(
	ctx context.Context, parNumber uint8, tag uint64,
) (*commtypes.RawMsg, error) {
	s.mux.RLock()
	if parNumber >= s.numPartitions {
		s.mux.RUnlock()
		return nil, xerrors.Errorf("Invalid partition number: %d", parNumber)
	}
	par := parNumber
	shard := s.subSharedLogStreams[par]
	msg, err := shard.Stream.ReadNextWithTag(ctx, parNumber, tag)
	s.mux.RUnlock()
	return msg, err
}

func (s *SizableShardedSharedLogStream) ReadBackwardWithTag(ctx context.Context, tailSeqNum uint64, parNum uint8, tag uint64) (*commtypes.RawMsg, error) {
	s.mux.RLock()
	if parNum < s.numPartitions {
		par := parNum
		shard := s.subSharedLogStreams[par]
		msg, err := shard.Stream.ReadBackwardWithTag(ctx, tailSeqNum, parNum, tag)
		s.mux.RUnlock()
		return msg, err
	} else {
		s.mux.RUnlock()
		return nil, xerrors.Errorf("Invalid partition number: %d", parNum)
	}
}

func (s *SizableShardedSharedLogStream) TopicNameHash() uint64 {
	return s.subSharedLogStreams[0].Stream.topicNameHash
}

func (s *SizableShardedSharedLogStream) TopicName() string {
	return s.topicName
}

// subSharedLogStreams array should only be modified in one thread
type ShardedSharedLogStream struct {
	topicName string

	subSharedLogStreams []*BufferedSinkStream
	bufMaxSize          uint32
	numPartitions       uint8
	serdeFormat         commtypes.SerdeFormat
}

var _ = Stream(&ShardedSharedLogStream{})

var ErrZeroParNum = xerrors.New("Shards must be positive")

func NewShardedSharedLogStream(topicName string, numPartitions uint8,
	serdeFormat commtypes.SerdeFormat, bufMaxSize uint32,
) (*ShardedSharedLogStream, error) {
	if numPartitions == 0 {
		panic(ErrZeroParNum)
	}
	streams := make([]*BufferedSinkStream, 0, numPartitions)
	for i := uint8(0); i < numPartitions; i++ {
		s, err := NewSharedLogStream(topicName, serdeFormat)
		if err != nil {
			return nil, err
		}
		buf := NewBufferedSinkStream(s, i, bufMaxSize)
		streams = append(streams, buf)
	}
	return &ShardedSharedLogStream{
		subSharedLogStreams: streams,
		numPartitions:       numPartitions,
		topicName:           topicName,
		serdeFormat:         serdeFormat,
		bufMaxSize:          bufMaxSize,
	}, nil
}

func (s *ShardedSharedLogStream) SetLastMarkerSeq(lastMarkerSeq uint64) {
	for _, stream := range s.subSharedLogStreams {
		stream.SetLastMarkerSeq(lastMarkerSeq)
	}
}

func (s *ShardedSharedLogStream) OutputRemainingStats() {
	for _, shard := range s.subSharedLogStreams {
		shard.OutputRemainingStats()
	}
}

func (s *ShardedSharedLogStream) ExactlyOnce(gua exactly_once_intr.GuaranteeMth) {
	for _, substream := range s.subSharedLogStreams {
		substream.ExactlyOnce(gua)
	}
}

func (s *ShardedSharedLogStream) ResetInitialProd() {
	for _, substream := range s.subSharedLogStreams {
		substream.ResetInitialProd()
	}
}

func (s *ShardedSharedLogStream) GetInitialProdSeqNum(substreamNum uint8) uint64 {
	return s.subSharedLogStreams[substreamNum].GetInitialProdSeqNum()
}

func (s *SizableShardedSharedLogStream) ScaleSubstreams(scaleTo uint8) error {
	if scaleTo == 0 {
		return fmt.Errorf("updated number of substreams should be larger and equal to one")
	}
	s.mux.Lock()
	if scaleTo > s.numPartitions {
		numAdd := scaleTo - s.numPartitions
		for i := uint8(0); i < numAdd; i++ {
			subs, err := NewSharedLogStream(s.topicName, s.serdeFormat)
			if err != nil {
				s.mux.Unlock()
				return err
			}
			buf := NewBufferedSinkStream(subs, i+s.numPartitions, s.bufMaxSize)
			s.subSharedLogStreams = append(s.subSharedLogStreams, buf)
		}
	}
	s.numPartitions = scaleTo
	s.mux.Unlock()
	return nil
}

func (s *ShardedSharedLogStream) NumPartition() uint8 {
	ret := s.numPartitions
	return ret
}

func (s *ShardedSharedLogStream) Push(ctx context.Context, payload []byte, parNum uint8,
	meta LogEntryMeta, producerId commtypes.ProducerId,
) (uint64, error) {
	off, err := s.subSharedLogStreams[parNum].Push(ctx, payload, parNum, meta, producerId)
	return off, err
}

func (s *ShardedSharedLogStream) BufPush(ctx context.Context, payload []byte, parNum uint8,
	producerId commtypes.ProducerId, flushCallback exactly_once_intr.FlushCallbackFunc,
) error {
	id := commtypes.GetCtxId(ctx)
	_ = id
	// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) before bufpush\n", id, s.topicName, parNum)
	err := s.subSharedLogStreams[parNum].BufPushAutoFlushGoroutineSafe(ctx, payload, producerId, flushCallback)
	// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) after bufpush\n", id, s.topicName, parNum)
	return err
}

func (s *ShardedSharedLogStream) Flush(ctx context.Context,
	producerId commtypes.ProducerId,
) (uint32, error) {
	flushed := uint32(0)
	for i := uint8(0); i < s.numPartitions; i++ {
		f, err := s.subSharedLogStreams[i].FlushGoroutineSafe(ctx, producerId)
		if err != nil {
			return 0, err
		}
		flushed += f
	}
	return flushed, nil
}

func (s *ShardedSharedLogStream) BufPushNoLock(ctx context.Context, payload []byte, parNum uint8,
	producerId commtypes.ProducerId, flushCallback exactly_once_intr.FlushCallbackFunc,
) error {
	err := s.subSharedLogStreams[parNum].BufPushAutoFlushNoLock(ctx, payload, producerId, flushCallback)
	return err
}

func (s *ShardedSharedLogStream) ParallelFlushNoLock(ctx context.Context,
	producerId commtypes.ProducerId,
) (uint32, error) {
	var flushed atomic.Uint32
	flushed.Store(0)
	bgGrp, bgCtx := errgroup.WithContext(ctx)
	for i := uint8(0); i < s.numPartitions; i++ {
		idx := i
		bgGrp.Go(func() error {
			f, err := s.subSharedLogStreams[idx].FlushNoLock(bgCtx, producerId)
			if err != nil {
				return err
			}
			flushed.Add(f)
			return nil
		})
	}
	err := bgGrp.Wait()
	if err != nil {
		return 0, err
	}
	return flushed.Load(), nil
}

func (s *ShardedSharedLogStream) FlushNoLock(ctx context.Context,
	producerId commtypes.ProducerId,
) (uint32, error) {
	// var flushed atomic.Uint32
	// flushed.Store(0)
	// bgGrp, bgCtx := errgroup.WithContext(ctx)
	// for i := uint8(0); i < s.numPartitions; i++ {
	// 	idx := i
	// 	bgGrp.Go(func() error {
	// 		f, err := s.subSharedLogStreams[idx].FlushNoLock(bgCtx, producerId)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		flushed.Add(f)
	// 		return nil
	// 	})
	// }
	// err := bgGrp.Wait()
	// if err != nil {
	// 	return 0, err
	// }
	// return flushed.Load(), nil

	flushed := uint32(0)
	for i := uint8(0); i < s.numPartitions; i++ {
		f, err := s.subSharedLogStreams[i].FlushNoLock(ctx, producerId)
		if err != nil {
			return 0, err
		}
		flushed += f
	}
	return flushed, nil
}

func (s *ShardedSharedLogStream) PushWithTag(ctx context.Context, payload []byte, parNum uint8, tags []uint64,
	additionalTopic []string, meta LogEntryMeta, producerId commtypes.ProducerId,
) (uint64, error) {
	off, err := s.subSharedLogStreams[parNum].PushWithTag(ctx, payload, parNum, tags,
		additionalTopic, meta, producerId)
	return off, err
}

func (s *ShardedSharedLogStream) ReadNext(ctx context.Context, parNumber uint8) (*commtypes.RawMsg, error) {
	if parNumber >= s.numPartitions {
		return nil, xerrors.Errorf("Invalid partition number: %d", parNumber)
	}
	par := parNumber
	shard := s.subSharedLogStreams[par]
	msg, err := shard.Stream.ReadNext(ctx, parNumber)
	return msg, err
}

func (s *ShardedSharedLogStream) SyncToRecent(ctx context.Context, parNumber uint8, producerId commtypes.ProducerId) (uint64, error) {
	if parNumber >= s.numPartitions {
		return 0, xerrors.Errorf("Invalid partition number: %d", parNumber)
	}
	par := parNumber
	shard := s.subSharedLogStreams[par]
	meta := SyncToRecentMeta()
	tags := []uint64{
		NameHashWithPartition(shard.Stream.topicNameHash, parNumber),
	}
	off, err := shard.Stream.PushWithTag(ctx, nil, 0, tags, nil,
		meta, producerId)
	if err != nil {
		return 0, fmt.Errorf("PushWithTag: %v", err)
	}
	return off, nil
}

func (s *ShardedSharedLogStream) ReadNextWithTag(
	ctx context.Context, parNum uint8, tag uint64,
) (*commtypes.RawMsg, error) {
	if parNum >= s.numPartitions {
		return nil, xerrors.Errorf("Invalid partition number: %d", parNum)
	}
	shard := s.subSharedLogStreams[parNum]
	msg, err := shard.Stream.ReadNextWithTag(ctx, parNum, tag)
	return msg, err
}

func (s *ShardedSharedLogStream) ReadBackwardWithTag(ctx context.Context, tailSeqNum uint64, parNum uint8, tag uint64) (*commtypes.RawMsg, error) {
	if parNum >= s.numPartitions {
		return nil, xerrors.Errorf("Invalid partition number: %d", parNum)
	}
	par := parNum
	shard := s.subSharedLogStreams[par]
	msg, err := shard.Stream.ReadBackwardWithTag(ctx, tailSeqNum, parNum, tag)
	return msg, err
}

func (s *ShardedSharedLogStream) TopicName() string {
	return s.topicName
}

func (s *ShardedSharedLogStream) TopicNameHash() uint64 {
	return s.subSharedLogStreams[0].Stream.topicNameHash
}

// not threadsafe for modify cursor; readlock is protecting the subSharedLogStreams
func (s *ShardedSharedLogStream) SetCursor(cursor uint64, parNum uint8) {
	s.subSharedLogStreams[parNum].Stream.cursor = cursor
}

func (s *ShardedSharedLogStream) GetCuror(parNum uint8) uint64 {
	cur := s.subSharedLogStreams[parNum].Stream.cursor
	return cur
}
