package stream_task

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/control_channel"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/env_config"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/transaction"
	"sync"
	"time"
)

func SetupManagersFor2pcTest(ctx context.Context, streamTaskArgs *StreamTaskArgs) (*transaction.TransactionManager, error) {
	tm, err := transaction.NewTransactionManager(ctx,
		streamTaskArgs.transactionalId, streamTaskArgs.serdeFormat)
	if err != nil {
		return nil, err
	}
	initRet, err := tm.InitTransaction(ctx)
	if err != nil {
		return nil, err
	}
	trackStream := func(name string, stream *sharedlog_stream.ShardedSharedLogStream) {
		tm.RecordTopicStreams(name, stream)
	}
	configChangelogExactlyOnce(tm, streamTaskArgs, trackStream)
	err = createOffsetTopicTrackSource(tm, streamTaskArgs) // txn mngr
	if err != nil {
		return nil, err
	}
	if initRet.HasRecentTxnMeta {
		offsetMap, err := getOffsetMap(ctx, tm, streamTaskArgs)
		if err != nil {
			return nil, err
		}
		err = restoreStateStore(ctx, streamTaskArgs)
		if err != nil {
			return nil, err
		}
		setOffsetOnStream(offsetMap, streamTaskArgs)
	}
	setupFuncsFor2pcTest(tm, streamTaskArgs)
	return tm, nil
}

func SetupManagersFor2pc(ctx context.Context, t *StreamTask,
	streamTaskArgs *StreamTaskArgs, rs *snapshot_store.RedisSnapshotStore,
) (*transaction.TransactionManager, *control_channel.ControlChannelManager, error) {
	checkStreamArgs(streamTaskArgs)
	debug.Fprint(os.Stderr, "setup transaction and control manager\n")
	tm, err := transaction.NewTransactionManager(ctx,
		streamTaskArgs.transactionalId, streamTaskArgs.serdeFormat)
	if err != nil {
		return nil, nil, fmt.Errorf("NewTransactionManager failed: %v", err)
	}
	initRet, err := tm.InitTransaction(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("InitTransaction failed: %v", err)
	}
	trackStream := func(name string, stream *sharedlog_stream.ShardedSharedLogStream) {
		tm.RecordTopicStreams(name, stream)
	}
	configChangelogExactlyOnce(tm, streamTaskArgs, trackStream)
	err = createOffsetTopicTrackSource(tm, streamTaskArgs) // txn mngr
	if err != nil {
		return nil, nil, err
	}
	if initRet.HasRecentTxnMeta {
		debug.Fprint(os.Stderr, "start restore\n")
		restoreBeg := time.Now()
		offsetMap, err := getOffsetMap(ctx, tm, streamTaskArgs)
		if err != nil {
			return nil, nil, err
		}
		debug.Fprintf(os.Stderr, "got offset map: %v\n", offsetMap)
		// TODO: the most recent completed txn might not have a snapshot
		// if env_config.CREATE_SNAPSHOT {
		// 	loadSnapBeg := time.Now()
		// 	err = loadSnapshot(ctx, streamTaskArgs, initRet.RecentTxnAuxData, initRet.RecentTxnLogSeq, rs)
		// 	if err != nil {
		// 		return nil, nil, err
		// 	}
		// 	loadSnapElapsed := time.Since(loadSnapBeg)
		// 	fmt.Fprintf(os.Stderr, "load snapshot took %v\n", loadSnapElapsed)
		// }
		restoreSsBeg := time.Now()
		err = restoreStateStore(ctx, streamTaskArgs)
		if err != nil {
			return nil, nil, err
		}
		restoreStateStoreElapsed := time.Since(restoreSsBeg)
		fmt.Fprintf(os.Stderr, "restore state store took %v\n", restoreStateStoreElapsed)
		setOffsetOnStream(offsetMap, streamTaskArgs)
		debug.Fprintf(os.Stderr, "down restore\n")
		restoreElapsed := time.Since(restoreBeg)
		fmt.Fprintf(os.Stderr, "down restore, elapsed: %v\n", restoreElapsed)
	}

	cmm, err := control_channel.NewControlChannelManager(streamTaskArgs.appId,
		commtypes.SerdeFormat(streamTaskArgs.serdeFormat), streamTaskArgs.bufMaxSize,
		streamTaskArgs.ectx.CurEpoch(), streamTaskArgs.ectx.SubstreamNum())
	if err != nil {
		return nil, nil, err
	}
	err = cmm.RestoreMappingAndWaitForPrevTask(ctx, streamTaskArgs.ectx.FuncName(),
		env_config.CREATE_SNAPSHOT, streamTaskArgs.serdeFormat,
		streamTaskArgs.kvChangelogs, streamTaskArgs.windowStoreChangelogs, rs)
	if err != nil {
		return nil, nil, err
	}
	trackParFunc := func(topicName string, substreamId uint8) {
		tm.AddTopicSubstream(topicName, substreamId)
	}
	recordFinish := func(ctx context.Context, funcName string, instanceID uint8) error {
		return cmm.RecordPrevInstanceFinish(ctx, funcName, instanceID, streamTaskArgs.ectx.CurEpoch())
	}
	flushCallbackFunc := func(ctx context.Context) error {
		waitPrev := stats.TimerBegin()
		waited, err := tm.EnsurePrevTxnFinAndAppendMeta(ctx)
		waitAndStart := stats.Elapsed(waitPrev).Microseconds()
		if waited == transaction.WaitPrev {
			t.waitedInPushCounter.Tick(1)
		} else if waited == transaction.AppendedNewPar {
			t.appendedMetaInPushCounter.Tick(1)
		} else if waited == transaction.WaitAndAppend {
			t.waitAndAppendInPushCounter.Tick(1)
		}
		if waited != transaction.None {
			t.waitPrevTxnInPush.AddSample(waitAndStart)
		}
		return err
	}
	updateFuncs(streamTaskArgs,
		trackParFunc, recordFinish, flushCallbackFunc)
	for _, p := range streamTaskArgs.ectx.Producers() {
		p.ConfigExactlyOnce(tm, streamTaskArgs.guarantee)
		tm.RecordTopicStreams(p.TopicName(), p.Stream().(*sharedlog_stream.ShardedSharedLogStream))
		p.SetFlushCallback(flushCallbackFunc)
	}
	return tm, cmm, nil
}

func setupFuncsFor2pcTest(tm *transaction.TransactionManager, args *StreamTaskArgs) {
	trackParFunc := func(topicName string, substreamId uint8) {
		tm.AddTopicSubstream(topicName, substreamId)
	}
	recordFinish := func(ctx context.Context, funcName string, instanceID uint8) error {
		return nil
	}
	flushCallbackFunc := func(ctx context.Context) error {
		_, err := tm.EnsurePrevTxnFinAndAppendMeta(ctx)
		return err
	}
	updateFuncs(args,
		trackParFunc, recordFinish, flushCallbackFunc)
	for _, p := range args.ectx.Producers() {
		p.ConfigExactlyOnce(tm, args.guarantee)
		tm.RecordTopicStreams(p.TopicName(), p.Stream().(*sharedlog_stream.ShardedSharedLogStream))
		p.SetFlushCallback(flushCallbackFunc)
	}
}

type txnProcessMeta struct {
	t    *StreamTask
	tm   *transaction.TransactionManager
	cmm  *control_channel.ControlChannelManager
	args *StreamTaskArgs
}

func processWithTransaction(
	ctx context.Context,
	meta *txnProcessMeta,
) *common.FnOutput {
	// trackStreamAndConfigureExactlyOnce(meta.args, meta.tm,
	// 	func(name string, stream *sharedlog_stream.ShardedSharedLogStream) {
	// 		meta.tm.RecordTopicStreams(name, stream)
	// 	})
	// latencies := stats.NewStatsCollector[int64]("latPerIter", stats.DEFAULT_COLLECT_DURATION)
	hasProcessData := false
	paused := false
	commitTimer := time.Now()
	snapshotTimer := time.Now()

	fmt.Fprintf(os.Stderr, "commit every(ms): %v, waitEndMark: %v, fixed output parNum: %d, snapshot every(s): %v, sinkBufSizeMax: %v\n",
		meta.args.commitEvery, meta.args.waitEndMark, meta.args.fixedOutParNum, meta.args.snapshotEvery, meta.args.bufMaxSize)
	init := false
	var once sync.Once
	warmupCheck := stats.NewWarmupChecker(meta.args.warmup)
	for {
		// procStart := time.Now()
		once.Do(func() {
			warmupCheck.StartWarmup()
		})
		warmupCheck.Check()
		timeSinceTranStart := time.Since(commitTimer)
		shouldCommitByTime := (meta.args.commitEvery != 0 && timeSinceTranStart > meta.args.commitEvery)
		// debug.Fprintf(os.Stderr, "shouldCommitByTime: %v, timeSinceTranStart: %v, cur_elapsed: %v\n",
		// 	shouldCommitByTime, timeSinceTranStart, cur_elapsed)

		// should commit
		if shouldCommitByTime && hasProcessData {
			err_out := commitTransaction(ctx, meta, &paused, false, &snapshotTimer)
			if err_out != nil {
				fmt.Fprintf(os.Stderr, "commitTransaction err: %v\n", err_out.Message)
				return err_out
			}
			hasProcessData = false
			commitTimer = time.Now()
		}

		// Exit routine
		cur_elapsed := warmupCheck.ElapsedSinceInitial()
		timeout := meta.args.duration != 0 && cur_elapsed >= meta.args.duration
		if !meta.args.waitEndMark && timeout {
			// elapsed := time.Since(procStart)
			// latencies.AddSample(elapsed.Microseconds())
			err_out := commitTransaction(ctx, meta, &paused, true, &snapshotTimer)
			if err_out != nil {
				fmt.Fprintf(os.Stderr, "commitTransaction err: %v\n", err_out.Message)
				return err_out
			}
			ret := &common.FnOutput{Success: true}
			updateReturnMetric(ret, &warmupCheck,
				meta.args.waitEndMark, meta.t.GetEndDuration(), meta.args.ectx.SubstreamNum())
			return ret
		}

		// begin new transaction
		if !hasProcessData && (!init || paused) {
			// debug.Fprintf(os.Stderr, "fixedOutParNum: %d\n", args.fixedOutParNum)
			err := initAfterMarkOrCommit(meta.t, meta.args, meta.tm, &init, &paused)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] initAfterMarkOrCommit failed: %v\n", err)
				return common.GenErrFnOutput(err)
			}
		}

		app_ret, ctrlRawMsgArr := meta.t.appProcessFunc(ctx, meta.t, meta.args.ectx)
		if app_ret != nil {
			if app_ret.Success {
				// consume timeout but not sure whether there's more data is coming; continue to process
				debug.Assert(ctrlRawMsgArr == nil, "when timeout, ctrlMsg should not be returned")
				continue
			} else {
				if hasProcessData {
					if err := meta.tm.AbortTransaction(ctx); err != nil {
						return common.GenErrFnOutput(fmt.Errorf("abort failed: %v\n", err))
					}
				}
				return app_ret
			}
		}
		if !hasProcessData {
			hasProcessData = true
		}
		if ctrlRawMsgArr != nil {
			fmt.Fprintf(os.Stderr, "got ctrl msg, exp finish\n")
			err_out := commitTransaction(ctx, meta, &paused, true, &snapshotTimer)
			if err_out != nil {
				fmt.Fprintf(os.Stderr, "commitTransaction err: %v\n", err_out.Message)
				return err_out
			}
			meta.t.PrintRemainingStats()
			meta.tm.OutputRemainingStats()
			fmt.Fprintf(os.Stderr, "finish final commit\n")
			return handleCtrlMsg(ctx, ctrlRawMsgArr, meta.t, meta.args, &warmupCheck, nil)
		}
		// if warmupCheck.AfterWarmup() {
		// 	elapsed := time.Since(procStart)
		// 	latencies.AddSample(elapsed.Microseconds())
		// }
	}
}

func commitTransaction(ctx context.Context,
	meta *txnProcessMeta,
	paused *bool,
	finalCommit bool,
	snapshotTimer *time.Time,
) *common.FnOutput {
	// debug.Fprintf(os.Stderr, "about to pause\n")
	if meta.t.pauseFunc != nil {
		if ret := meta.t.pauseFunc(meta.args.guarantee); ret != nil {
			return ret
		}
		*paused = true
	}
	commitTxnBeg := stats.TimerBegin()
	// debug.Fprintf(os.Stderr, "paused\n")
	waitPrev := stats.TimerBegin()
	// for _, src := range meta.args.ectx.Consumers() {
	// 	meta.tm.AddTopicTrackConsumedSeqs(src.TopicName(), meta.args.ectx.SubstreamNum())
	// }

	waited, err := meta.tm.EnsurePrevTxnFinAndAppendMeta(ctx)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	waitAndStart := stats.Elapsed(waitPrev).Microseconds()
	// debug.Fprintf(os.Stderr, "waited prev txn fin\n")
	flushAllStart := stats.TimerBegin()
	f, err := flushStreams(ctx, meta.t, meta.args)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("flushStreams: %v", err))
	}
	flushTime := stats.Elapsed(flushAllStart).Microseconds()
	// debug.Fprintf(os.Stderr, "flushed\n")

	offsetBeg := stats.TimerBegin()
	err = meta.tm.AppendConsumedSeqNum(ctx, meta.args.ectx.Consumers(), meta.args.ectx.SubstreamNum())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] append offset failed: %v\n", err)
		return common.GenErrFnOutput(fmt.Errorf("append offset failed: %v\n", err))
	}
	sendOffsetElapsed := stats.Elapsed(offsetBeg).Microseconds()

	var logOff uint64
	// var shouldExit bool
	cBeg := stats.TimerBegin()
	if env_config.ASYNC_SECOND_PHASE && !finalCommit {
		logOff, _, err = meta.tm.CommitTransactionAsyncComplete(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] commitAsyncComplete failed: %v\n", err)
			return common.GenErrFnOutput(fmt.Errorf("commitAsyncComplete failed: %v\n", err))
		}
	} else {
		logOff, _, err = meta.tm.CommitTransaction(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] commit failed: %v\n", err)
			return common.GenErrFnOutput(fmt.Errorf("commit failed: %v\n", err))
		}
	}
	// debug.Fprintf(os.Stderr, "committed transaction\n")
	if env_config.CREATE_SNAPSHOT && meta.args.snapshotEvery != 0 && time.Since(*snapshotTimer) > meta.args.snapshotEvery {
		// auxdata is set to precommit
		createSnapshot(ctx, meta.args, []commtypes.TpLogOff{{LogOff: logOff}})
		*snapshotTimer = time.Now()
	}
	cElapsed := stats.Elapsed(cBeg).Microseconds()
	commitTxnElapsed := stats.Elapsed(commitTxnBeg).Microseconds()
	meta.t.txnCounter.Tick(1)
	if waited == transaction.WaitPrev {
		meta.t.waitedInCmtCounter.Tick(1)
	} else if waited == transaction.AppendedNewPar {
		meta.t.appendedMetaInCmtCounter.Tick(1)
	} else if waited == transaction.WaitAndAppend {
		meta.t.waitAndAppendInCmtCounter.Tick(1)
	}
	if waited != transaction.None {
		meta.t.waitPrevTxnInCmt.AddSample(waitAndStart)
	}

	meta.t.txnCommitTime.AddSample(commitTxnElapsed)
	meta.t.commitTxnAPITime.AddSample(cElapsed)
	meta.t.flushStageTime.AddSample(flushTime)
	meta.t.sendOffsetTime.AddSample(sendOffsetElapsed)
	if f > 0 {
		meta.t.flushAtLeastOne.AddSample(flushTime)
	}
	// if shouldExit {
	// 	return &common.FnOutput{Success: true, Message: "exit"}
	// }
	return nil
}
