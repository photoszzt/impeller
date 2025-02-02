package execution

import (
	"context"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/stream_task"
	"sync"
)

type MsgSerdePair[KI, VI, KO, VO any] struct {
	inMsgSerde  commtypes.MessageGSerdeG[KI, VI]
	outMsgSerde commtypes.MessageGSerdeG[KO, VO]
}

func NewMsgSerdePair[KI, VI, KO, VO any](inMsgSerde commtypes.MessageGSerdeG[KI, VI],
	outMsgSerde commtypes.MessageGSerdeG[KO, VO],
) MsgSerdePair[KI, VI, KO, VO] {
	return MsgSerdePair[KI, VI, KO, VO]{
		inMsgSerde:  inMsgSerde,
		outMsgSerde: outMsgSerde,
	}
}

func PrepareTaskWithJoin[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR any](
	ctx context.Context,
	leftJoinWorker JoinWorkerFunc[KInL, VInL, KOutL, VOutL],
	rightJoinWorker JoinWorkerFunc[KInR, VInR, KOutR, VOutR],
	allConsumersProducers proc_interface.BaseConsumersProducers,
	baseProcArgs proc_interface.BaseProcArgs,
	isFinalStage bool,
	leftMsgPair MsgSerdePair[KInL, VInL, KOutL, VOutL],
	rightMsgPair MsgSerdePair[KInR, VInR, KOutR, VOutR],
	stageTag string,
) (*stream_task.StreamTask, *CommonJoinProcArgs[KInL, VInL, KOutL, VOutL, KInR, VInR, KOutR, VOutR]) {
	joinProcLeft, joinProcRight := CreateJoinProcArgsPair(
		leftJoinWorker, rightJoinWorker,
		allConsumersProducers.Consumers(),
		allConsumersProducers.Producers(), baseProcArgs, stageTag)
	var wg sync.WaitGroup
	leftManager := NewJoinProcManager()
	rightManager := NewJoinProcManager()

	procArgs := NewCommonJoinProcArgs(
		joinProcLeft, joinProcRight,
		allConsumersProducers)
	lctx := context.WithValue(ctx, commtypes.CTXID{}, "left")
	rctx := context.WithValue(ctx, commtypes.CTXID{}, "right")

	handleJoinErrReturn := func() *common.FnOutput {
		var out1 *common.FnOutput
		var out2 *common.FnOutput
		select {
		case out2 = <-rightManager.out:
			debug.Fprintf(os.Stderr, "Got out2 out: %v, per out channel len: %d\n", out2, len(rightManager.out))
		case out1 = <-leftManager.out:
			debug.Fprintf(os.Stderr, "Got out1 out: %v, auc out channel len: %d\n", out1, len(leftManager.out))
		default:
		}
		if out1 != nil || out2 != nil {
			debug.Fprintf(os.Stderr, "out1: %v\n", out1)
			debug.Fprintf(os.Stderr, "out2: %v\n", out2)
		}
		if out2 != nil && !out2.Success {
			return out2
		}
		if out1 != nil && !out1.Success {
			return out1
		}
		return nil
	}

	taskBuilder := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(
			func(ctx context.Context, task *stream_task.StreamTask,
				argsTmp processor.ExecutionContext,
			) (*common.FnOutput, []*commtypes.RawMsgAndSeq) {
				if leftManager.GotEndMark() && rightManager.GotEndMark() {
					debug.Fprintf(os.Stderr, "join proc got end mark\n")
					debug.Assert(leftManager.ctrlMsg != nil, "left manager ctrl msg should not be nil")
					debug.Assert(rightManager.ctrlMsg != nil, "right manager ctrl msg should not be nil")
					return nil, []*commtypes.RawMsgAndSeq{leftManager.ctrlMsg, rightManager.ctrlMsg}
				} else if leftManager.GotScaleFence() && rightManager.GotScaleFence() {
					debug.Fprintf(os.Stderr, "join proc got scale fence\n")
					debug.Assert(leftManager.ctrlMsg != nil, "left manager ctrl msg should not be nil")
					debug.Assert(rightManager.ctrlMsg != nil, "right manager ctrl msg should not be nil")
					return nil, []*commtypes.RawMsgAndSeq{leftManager.ctrlMsg, rightManager.ctrlMsg}
				} else if leftManager.GotChkptMark() && rightManager.GotChkptMark() {
					debug.Fprintf(os.Stderr, "join proc got all checkpt mark\n")
					debug.Assert(leftManager.ctrlMsg != nil, "left manager ctrl msg should not be nil")
					debug.Assert(rightManager.ctrlMsg != nil, "right manager ctrl msg should not be nil")
					dur_between := leftManager.gotChkptTime.Sub(rightManager.gotChkptTime).Abs().Milliseconds()
					procArgs.chkPtBtwTime.AddSample(dur_between)
					return nil, []*commtypes.RawMsgAndSeq{leftManager.ctrlMsg, rightManager.ctrlMsg}
				}
				return handleJoinErrReturn(), nil
			}).
		InitFunc(
			func(task *stream_task.StreamTask) {
				// debug.Fprintf(os.Stderr, "init ts=%d launch join proc loops\n", time.Now().UnixMilli())
				LaunchJoinProcLoop(lctx, leftManager, joinProcLeft, &wg, leftMsgPair)
				LaunchJoinProcLoop(rctx, rightManager, joinProcRight, &wg, rightMsgPair)
				// debug.Fprintf(os.Stderr, "init ts=%d done invoke join proc loops\n", time.Now().UnixMilli())
			}).
		PauseFunc(
			func(gua exactly_once_intr.GuaranteeMth) *common.FnOutput {
				// debug.Fprintf(os.Stderr, "in pause func\n")
				if ret := handleJoinErrReturn(); ret != nil {
					return ret
				}
				pStart := stats.TimerBegin()
				if gua != exactly_once_intr.ALIGN_CHKPT {
					leftManager.LockRunlock()
					rightManager.LockRunlock()
				}
				elapsed := stats.Elapsed(pStart)
				procArgs.pauseFuncTime.AddSample(elapsed.Microseconds())
				return nil
			}).
		ResumeFunc(
			func(task *stream_task.StreamTask, gua exactly_once_intr.GuaranteeMth) {
				// debug.Fprintf(os.Stderr, "in resume func\n")
				rStart := stats.TimerBegin()
				if gua == exactly_once_intr.ALIGN_CHKPT {
					leftManager.gotChkptMark.Store(false)
					rightManager.gotChkptMark.Store(false)
					leftManager.ctrlMsg = nil
					rightManager.ctrlMsg = nil
					LaunchJoinProcLoop(lctx, leftManager, joinProcLeft, &wg, leftMsgPair)
					LaunchJoinProcLoop(rctx, rightManager, joinProcRight, &wg, rightMsgPair)
				} else {
					leftManager.UnlockRunlock()
					rightManager.UnlockRunlock()
				}
				elapsed := stats.Elapsed(rStart)
				procArgs.resumeFuncTime.AddSample(elapsed.Microseconds())
				// debug.Fprintf(os.Stderr, "done resume func\n")
			})
	if isFinalStage {
		return taskBuilder.MarkFinalStage().Build(), procArgs
	} else {
		return taskBuilder.Build(), procArgs
	}
}
