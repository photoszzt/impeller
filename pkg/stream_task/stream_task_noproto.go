package stream_task

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/stats"
	"time"
)

func processNoProto(ctx context.Context, t *StreamTask, args *StreamTaskArgs) *common.FnOutput {
	debug.Assert(len(args.ectx.Consumers()) >= 1, "Srcs should be filled")
	debug.Assert(args.ectx != nil, "program args should be filled")
	args.ectx.StartWarmup()
	if t.initFunc != nil {
		t.initFunc(t)
	}

	debug.Fprintf(os.Stderr, "warmup time: %v, flush every: %v, waitEndMark: %v\n",
		args.warmup, args.flushEvery, args.waitEndMark)
	warmupCheck := stats.NewWarmupChecker(args.warmup)
	warmupCheck.StartWarmup()
	flushTimer := time.Now()
	for {
		timeSinceLastFlush := time.Since(flushTimer)
		if timeSinceLastFlush >= args.flushEvery {
			ret_err := timedFlushStreams(ctx, t, args)
			if ret_err != nil {
				return ret_err
			}
			flushTimer = time.Now()
		}
		warmupCheck.Check()
		if !args.waitEndMark && args.duration != 0 && warmupCheck.ElapsedSinceInitial() >= args.duration {
			break
		}
		ret, ctrlRawMsgArr := t.appProcessFunc(ctx, t, args.ectx)
		if ret != nil {
			if ret.Success {
				continue
			}
			return ret
		}
		if ctrlRawMsgArr != nil {
			fmt.Fprintf(os.Stderr, "exit due to ctrlMsg\n")
			if ret_err := pauseTimedFlushStreams(ctx, t, args); ret_err != nil {
				return ret_err
			}
			return handleCtrlMsg(ctx, ctrlRawMsgArr, t, args, &warmupCheck, nil)
		}
	}
	if ret_err := pauseTimedFlushStreams(ctx, t, args); ret_err != nil {
		return ret_err
	}
	ret := &common.FnOutput{Success: true}
	updateReturnMetric(ret, &warmupCheck,
		args.waitEndMark, t.GetEndDuration(), args.ectx.SubstreamNum())
	return ret
}
