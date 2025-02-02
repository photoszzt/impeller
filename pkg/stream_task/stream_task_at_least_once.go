package stream_task

import (
	"context"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/consume_seq_num_manager"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/stats"
	"time"
)

func process(ctx context.Context, t *StreamTask, args *StreamTaskArgs) *common.FnOutput {
	checkStreamArgs(args)
	// latencies := stats.NewInt64Collector("latPerIter", stats.DEFAULT_COLLECT_DURATION)
	cm, err := consume_seq_num_manager.NewConsumeSeqManager(args.serdeFormat,
		args.transactionalId, args.bufMaxSize)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	debug.Fprint(os.Stderr, "start restore\n")
	offsetMap, err := cm.FindLastConsumedSeqNum(ctx)
	if err != nil && !common_errors.IsStreamEmptyError(err) {
		return common.GenErrFnOutput(err)
	}
	err = restoreStateStore(ctx, args)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	setOffsetOnStream(offsetMap, args)
	debug.Fprint(os.Stderr, "done restore\n")
	args.ectx.StartWarmup()
	if t.initFunc != nil {
		t.initFunc(t)
	}
	hasUntrackedConsume := false

	debug.Fprintf(os.Stderr, "warmup time: %v, flush every: %v, waitEndMark: %v, sinkBufMaxSize: %v\n",
		args.warmup, args.flushEvery, args.waitEndMark, args.bufMaxSize)
	warmupCheck := stats.NewWarmupChecker(args.warmup)
	warmupCheck.StartWarmup()
	commitTimer := time.Now()
	flushTimer := time.Now()
	gotEndMark := false
	for {
		timeSinceLastFlush := time.Since(flushTimer)
		if timeSinceLastFlush >= args.flushEvery {
			ret_err := timedFlushStreams(ctx, t, args)
			if ret_err != nil {
				return ret_err
			}
			flushTimer = time.Now()
		}
		timeSinceLastTrack := time.Since(commitTimer)
		if timeSinceLastTrack >= args.trackEveryForAtLeastOnce {
			if ret_err := track(ctx, cm, args.ectx.Consumers()); ret_err != nil {
				return ret_err
			}
			hasUntrackedConsume = false
			commitTimer = time.Now()
		}
		warmupCheck.Check()
		if (!args.waitEndMark && args.duration != 0 && warmupCheck.ElapsedSinceInitial() >= args.duration) || gotEndMark {
			break
		}
		// procStart := time.Now()
		ret, ctrlRawMsgArr := t.appProcessFunc(ctx, t, args.ectx)
		if ret != nil {
			if ret.Success {
				continue
			}
			return ret
		}
		if !hasUntrackedConsume {
			hasUntrackedConsume = true
		}
		if ctrlRawMsgArr != nil {
			if ret_err := pauseTimedFlushStreams(ctx, t, args); ret_err != nil {
				return ret_err
			}
			if ret_err := track(ctx, cm, args.ectx.Consumers()); ret_err != nil {
				return ret_err
			}
			return handleCtrlMsg(ctx, ctrlRawMsgArr, t, args, &warmupCheck, nil)
		}
		// if warmupCheck.AfterWarmup() {
		// 	elapsed := time.Since(procStart)
		// 	latencies.AddSample(elapsed.Microseconds())
		// }
	}
	pauseTrackFlush(ctx, t, args, cm, hasUntrackedConsume)
	ret := &common.FnOutput{Success: true}
	updateReturnMetric(ret, &warmupCheck,
		args.waitEndMark, t.GetEndDuration(), args.ectx.SubstreamNum())
	return ret
}

func track(ctx context.Context, cm *consume_seq_num_manager.ConsumeSeqManager,
	consumers []*producer_consumer.MeteredConsumer,
) *common.FnOutput {
	markers := consume_seq_num_manager.CollectOffsetMarker(consumers)
	err := cm.Track(ctx, markers)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func pauseTrackFlush(ctx context.Context, t *StreamTask, args *StreamTaskArgs, cm *consume_seq_num_manager.ConsumeSeqManager, hasUncommitted bool) *common.FnOutput {
	alreadyPaused := false
	if hasUncommitted {
		if t.pauseFunc != nil {
			if ret := t.pauseFunc(args.guarantee); ret != nil {
				return ret
			}
		}
		markers := consume_seq_num_manager.CollectOffsetMarker(args.ectx.Consumers())
		err := cm.Track(ctx, markers)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		alreadyPaused = true
	}
	if !alreadyPaused && t.pauseFunc != nil {
		if ret := t.pauseFunc(args.guarantee); ret != nil {
			return ret
		}
	}
	ret_err := timedFlushStreams(ctx, t, args)
	if ret_err != nil {
		return ret_err
	}
	return nil
}
