package execution

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sync"
	"time"

	// "sharedlog-stream/pkg/producer_consumer"

	"golang.org/x/xerrors"
)

func joinProcLoop[KIn, VIn, KOut, VOut any](
	ctx context.Context,
	jm *JoinProcManager,
	procArgs *JoinProcArgs[KIn, VIn, KOut, VOut],
	wg *sync.WaitGroup,
	msgSerdePair MsgSerdePair[KIn, VIn, KOut, VOut],
) {
	id := ctx.Value(commtypes.CTXID{}).(string)
	debug.Assert(id != "", "id should not be empty")
	defer wg.Done()
	// debug.Fprintf(os.Stderr, "[id=%s, ts=%d] joinProc start running\n",
	// 	id, time.Now().UnixMilli())
	// lockAcqTime := stats.NewStatsCollector[int64](fmt.Sprintf("%s_lockAcq", id), stats.DEFAULT_COLLECT_DURATION)
	jWStart := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-jm.done:
			// debug.Fprintf(os.Stderr, "[id=%s, ts=%d] got done msg\n",
			// 	id, time.Now().UnixMilli())
			return
		case <-jm.flushAndCollect:
			for _, sink := range procArgs.Producers() {
				if _, err := sink.Flush(ctx); err != nil {
					jm.out <- common.GenErrFnOutput(err)
					return
				}
			}
		default:
		}
		// lSt := stats.TimerBegin()
		jm.runLock.Lock()
		// lelapsed := stats.Elapsed(lSt).Microseconds()
		// lockAcqTime.AddSample(lelapsed)
		// debug.Fprintf(os.Stderr, "[id=%s] before consume, stream name %s\n", id, procArgs.Consumers()[0].Stream().TopicName())
		consumer := procArgs.Consumers()[0]
		rawMsgSeq, err := consumer.Consume(ctx, procArgs.SubstreamNum())
		if err != nil {
			if xerrors.Is(err, common_errors.ErrStreamSourceTimeout) {
				// debug.Fprintf(os.Stderr, "[TIMEOUT] %s %s timeout, out chan len: %d\n",
				// 	id, procArgs.Consumers()[0].TopicName(), len(jm.out))
				// out <- &common.FnOutput{Success: true, Message: err.Error()}
				// debug.Fprintf(os.Stderr, "%s done sending msg\n", id)
				// return
				jm.runLock.Unlock()
				// count := consumer.GetCount()
				// debug.Fprintf(os.Stderr, "consume %s %d timeout, count %d\n",
				// 	consumer.TopicName(), procArgs.SubstreamNum(), count)
				continue
			}
			fmt.Fprintf(os.Stderr, "[ERROR] consume: %v, out chan len: %d\n", err, len(jm.out))
			jm.out <- common.GenErrFnOutput(err)
			fmt.Fprintf(os.Stderr, "%s done sending msg2\n", id)
			jm.runLock.Unlock()
			return
		}
		// fmt.Fprintf(os.Stderr, "read 0x%x from %s(%d)\n", rawMsgSeq.LogSeqNum,
		// 	consumer.TopicName(), procArgs.SubstreamNum())
		if rawMsgSeq.IsControl {
			procArgs.Consumers()[0].RecordCurrentConsumedSeqNum(rawMsgSeq.LogSeqNum)
			if rawMsgSeq.Mark == commtypes.SCALE_FENCE {
				// fmt.Fprintf(os.Stderr, "%s,%s(%d) scale fence, cur epoch %d, got epoch %d\n",
				// 	id, consumer.TopicName(), procArgs.SubstreamNum(), procArgs.CurEpoch(), rawMsgSeq.ScaleEpoch)
				if procArgs.CurEpoch() < rawMsgSeq.ScaleEpoch {
					consumer.SrcProducerGotScaleFence(rawMsgSeq.ProdIdx)
					if consumer.AllProducerScaleFenced() {
						jm.ctrlMsg = rawMsgSeq
						jm.gotScaleFence.Store(true)
						jm.runLock.Unlock()
						return
					} else {
						jm.runLock.Unlock()
						continue
					}
				} else {
					jm.runLock.Unlock()
					continue
				}
			} else if rawMsgSeq.Mark == commtypes.STREAM_END {
				consumer.SrcProducerEnd(rawMsgSeq.ProdIdx)
				elapsed := time.Since(jWStart)
				fmt.Fprintf(os.Stderr, "[id=%s] joinProc done, elapsed %v\n", id, elapsed)
				if consumer.AllProducerEnded() {
					jm.ctrlMsg = rawMsgSeq
					jm.gotEndMark.Store(true)
					// debug.Fprintf(os.Stderr, "[id=%s] %s %d ends, start time: %d\n",
					// 	id, consumer.TopicName(), procArgs.SubstreamNum(), jm.startTimeMs)
					// debug.Fprintf(os.Stderr, "[id=%s] %s(%d) ends, elapsed: %v\n", id, consumer.TopicName(), procArgs.SubstreamNum(), elapsed)
					jm.runLock.Unlock()
					return
				} else {
					jm.runLock.Unlock()
					continue
				}
			} else if rawMsgSeq.Mark == commtypes.EPOCH_END {
				jm.runLock.Unlock()
				continue
			} else if rawMsgSeq.Mark == commtypes.CHKPT_MARK {
				jm.ctrlMsg = rawMsgSeq
				jm.gotChkptTime = time.Now()
				jm.gotChkptMark.Store(true)
				jm.runLock.Unlock()
				return
			} else {
				jm.out <- common.GenErrFnOutput(
					fmt.Errorf("unrecognized mark: %v", rawMsgSeq.Mark))
				jm.runLock.Unlock()
				return
			}
		}
		msgs, err := commtypes.DecodeRawMsgSeqG(rawMsgSeq, msgSerdePair.inMsgSerde)
		if err != nil {
			jm.out <- common.GenErrFnOutput(err)
			jm.runLock.Unlock()
			return
		}
		if msgs.MsgArr == nil && msgs.Msg.Value.IsNone() && msgs.Msg.Key.IsNone() {
			jm.runLock.Unlock()
			continue
		}
		consumer.RecordCurrentConsumedSeqNum(rawMsgSeq.LogSeqNum)
		// debug.Fprintf(os.Stderr, "[id=%s] after consume\n", id)
		if msgs.MsgArr != nil {
			// debug.Fprintf(os.Stderr, "[id=%s] got msgarr\n", id)
			for _, subMsg := range msgs.MsgArr {
				if subMsg.Key.IsNone() && subMsg.Value.IsNone() {
					continue
				}
				subMsg.StartProcTime = time.Now()
				// producer_consumer.ExtractProduceToConsumeTimeMsgG(consumer, &subMsg)
				// batchTime := rawMsgSeq.InjTsMs - subMsg.InjTMs
				// consumer.CollectBatchTime(batchTime)
				// debug.Fprintf(os.Stderr, "[id=%s] before proc msg with sink1\n", id)
				err = procMsgWithSink(ctx, subMsg, msgSerdePair.outMsgSerde, procArgs, id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s progMsgWithSink: %v, out chan len: %d\n", id, err, len(jm.out))
					jm.out <- common.GenErrFnOutput(err)
					fmt.Fprintf(os.Stderr, "%s done send msg3\n", id)
					jm.runLock.Unlock()
					return
				}
			}
		} else {
			// debug.Fprintf(os.Stderr, "[id=%s] got single msg\n", id)
			if msgs.Msg.Key.IsNone() && msgs.Msg.Value.IsNone() {
				jm.runLock.Unlock()
				continue
			}
			msgs.Msg.StartProcTime = time.Now()
			// producer_consumer.ExtractProduceToConsumeTimeMsgG(consumer, &msgs.Msg)
			err = procMsgWithSink(ctx, msgs.Msg, msgSerdePair.outMsgSerde, procArgs, id)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] %s progMsgWithSink2: %v, out chan len: %d\n", id, err, len(jm.out))
				jm.out <- common.GenErrFnOutput(err)
				fmt.Fprintf(os.Stderr, "[id=%s] done send msg4\n", id)
				jm.runLock.Unlock()
				return
			}
		}
		jm.runLock.Unlock()
		// debug.Fprintf(os.Stderr, "[id=%s] after for loop\n", id)
	}
}

func procMsgWithSink[KIn, VIn, KOut, VOut any](ctx context.Context,
	msg commtypes.MessageG[KIn, VIn],
	outMsgSerde commtypes.MessageGSerdeG[KOut, VOut],
	procArgs *JoinProcArgs[KIn, VIn, KOut, VOut], id string,
) error {
	// st := msg.Value.(commtypes.EventTimeExtractor)
	// ts, err := st.ExtractEventTime()
	// if err != nil {
	// 	debug.Fprintf(os.Stderr, "[ERROR] %s return extract ts: %v\n", ctx.Value("id"), err)
	// 	return fmt.Errorf("fail to extract timestamp: %v", err)
	// }
	// msg.Timestamp = ts
	_ = id

	// debug.Fprintf(os.Stderr, "[id=%s] before runner\n", id)
	msgs, err := procArgs.runner(ctx, msg)
	if err != nil {
		debug.Fprintf(os.Stderr, "[ERROR] %s return runner: %v\n", ctx.Value("id"), err)
		return err
	}
	kUseBuf := outMsgSerde.GetKeySerdeG().UsedBufferPool()
	vUseBuf := outMsgSerde.GetValSerdeG().UsedBufferPool()
	// debug.Fprintf(os.Stderr, "[id=%s] after runner\n", id)
	for _, msg := range msgs {
		// procTime := time.Since(msg.StartProcTime)
		// procArgs.procLat.AddSample(procTime.Nanoseconds())
		// debug.Fprintf(os.Stderr, "k %v, v %v, ts %d\n", msg.Key, msg.Value, msg.Timestamp)
		msgSerOp, kbuf, vbuf, err := commtypes.MsgGToMsgSer(msg, outMsgSerde)
		if err != nil {
			return err
		}
		msgSer, ok := msgSerOp.Take()
		if ok {
			err = procArgs.Producers()[0].ProduceData(ctx, msgSer, procArgs.SubstreamNum())
			if err != nil {
				debug.Fprintf(os.Stderr, "[ERROR] %s return push to sink: %v\n", ctx.Value("id"), err)
				return err
			}
			if kUseBuf && kbuf != nil {
				*kbuf = msgSer.KeyEnc
				commtypes.PushBuffer(kbuf)
			}
			if vUseBuf && vbuf != nil {
				*vbuf = msgSer.ValueEnc
				commtypes.PushBuffer(vbuf)
			}
		}
	}
	// debug.Fprintf(os.Stderr, "[id=%s] after produce\n", id)
	return nil
}
