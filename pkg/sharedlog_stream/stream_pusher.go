package sharedlog_stream

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils"
	"sync"
	"time"
)

type PayloadToPush struct {
	Payload    []byte
	Partitions []uint8
	IsControl  bool
	Mark       commtypes.EpochMark
}

type StreamPush struct {
	FlushTimer    time.Time
	MsgChan       chan PayloadToPush
	MsgErrChan    chan error
	Stream        *SizableShardedSharedLogStream
	produceCount  uint64
	ctrlCount     uint64
	FlushDuration time.Duration
	BufPush       bool
}

func NewStreamPush(stream *SizableShardedSharedLogStream) *StreamPush {
	return &StreamPush{
		MsgChan:      make(chan PayloadToPush, MSG_CHAN_SIZE),
		MsgErrChan:   make(chan error, 1),
		BufPush:      utils.CheckBufPush(),
		Stream:       stream,
		produceCount: 0,
		ctrlCount:    0,
	}
}

func (h *StreamPush) InitFlushTimer(duration time.Duration) {
	if h.BufPush {
		h.FlushTimer = time.Now()
		h.FlushDuration = duration
		debug.Fprintf(os.Stderr, "InitFlushTimer: Flush duration %v\n", h.FlushDuration)
	}
}

// msgchan has to close and async pusher has to stop first before calling this function
func (h *StreamPush) Flush(ctx context.Context, producerId commtypes.ProducerId,
) (uint32, error) {
	if h.BufPush {
		f, err := h.Stream.Flush(ctx, producerId)
		if err != nil {
			return 0, err
		}
		h.FlushTimer = time.Now()
		debug.Fprintf(os.Stderr, "F: stream flushed\n")
		return f, nil
	}
	return 0, nil
}

// msgchan has to close and async pusher has to stop first before calling this function
func (h *StreamPush) FlushNoLock(ctx context.Context, producerId commtypes.ProducerId,
) (uint32, error) {
	if h.BufPush {
		f, err := h.Stream.FlushNoLock(ctx, producerId)
		if err != nil {
			return 0, err
		}
		h.FlushTimer = time.Now()
		debug.Fprintf(os.Stderr, "FOL: stream flushed\n")
		return f, nil
	}
	return 0, nil
}

func (h *StreamPush) GetCount() uint64 {
	return h.produceCount
}

func (h *StreamPush) NumCtrlMsgs() uint64 {
	return h.ctrlCount
}

func (h *StreamPush) AsyncStreamPush(ctx context.Context, wg *sync.WaitGroup,
	producerId commtypes.ProducerId, flushCallback exactly_once_intr.FlushCallbackFunc,
) {
	defer wg.Done()
	tpNameHash := h.Stream.TopicNameHash()
	for msg := range h.MsgChan {
		if msg.IsControl {
			if h.BufPush {
				_, err := h.Stream.Flush(ctx, producerId)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] flush err: %v\n", err)
					h.MsgErrChan <- err
					return
				}
				h.FlushTimer = time.Now()
			}
			for _, i := range msg.Partitions {
				nameHashTag := NameHashWithPartition(tpNameHash, i)
				if msg.Mark == commtypes.SCALE_FENCE {
					scale_fence_tag := txn_data.ScaleFenceTag(tpNameHash, i)
					_, err := h.Stream.PushWithTag(ctx, msg.Payload, i, []uint64{nameHashTag, scale_fence_tag}, nil,
						StreamEntryMeta(true, false), producerId)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[ERROR] push err: %v\n", err)
						h.MsgErrChan <- err
						return
					}
					// fmt.Fprintf(os.Stderr, "generate scale fence for partition %d at %x\n", i, off)
					h.produceCount += 1
					h.ctrlCount += 1
				} else if msg.Mark == commtypes.STREAM_END {
					_, err := h.Stream.PushWithTag(ctx, msg.Payload, i, []uint64{nameHashTag}, nil, StreamEntryMeta(true, false), producerId)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[ERROR] push err: %v\n", err)
						h.MsgErrChan <- err
						return
					}
					// fmt.Fprintf(os.Stderr, "generate stream end mark for par %d at %x\n", i, off)
					h.produceCount += 1
					h.ctrlCount += 1
				} else if msg.Mark == commtypes.CHKPT_MARK {
					chkpt_tag := txn_data.ChkptTag(tpNameHash, i)
					_, err := h.Stream.PushWithTag(ctx, msg.Payload, i, []uint64{nameHashTag, chkpt_tag}, nil,
						ControlRecordMeta, producerId)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[ERROR] push err: %v\n", err)
						h.MsgErrChan <- err
						return
					}
					// fmt.Fprintf(os.Stderr, "generate chkpt mark with tag for par %d at %x\n", i, off)
					h.produceCount += 1
					h.ctrlCount += 1
				}
			}
		} else {
			if h.BufPush {
				timeSinceLastFlush := time.Since(h.FlushTimer)
				if timeSinceLastFlush >= h.FlushDuration {
					// debug.Fprintf(os.Stderr, "flush timer: %v\n", timeSinceLastFlush)
					_, err := h.Stream.Flush(ctx, producerId)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[ERROR] flush err: %v\n", err)
						h.MsgErrChan <- err
						return
					}
					h.FlushTimer = time.Now()
				}
				err := h.Stream.BufPush(ctx, msg.Payload, uint8(msg.Partitions[0]), producerId, flushCallback)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] buf push err: %v\n", err)
					h.MsgErrChan <- err
					return
				}
				h.produceCount += 1
			} else {
				debug.Assert(len(msg.Partitions) == 1, "should only have one partition")
				_, err := h.Stream.Push(ctx, msg.Payload, uint8(msg.Partitions[0]), StreamEntryMeta(false, false), producerId)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] push err: %v\n", err)
					h.MsgErrChan <- err
					return
				}
				h.produceCount += 1
			}
		}
	}
}
