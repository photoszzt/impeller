package stream_task

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/control_channel"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/env_config"
	"sharedlog-stream/pkg/epoch_manager"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/txn_data"
	"sync"
	"time"
)

func SetKVStoreWithChangelogSnapshot[K, V any](
	ctx context.Context,
	rs *snapshot_store.RedisSnapshotStore,
	kvstore store.KeyValueStoreBackedByChangelogG[K, V],
	payloadSerde commtypes.SerdeG[commtypes.PayloadArr],
) {
	kvstore.SetSnapshotCallback(ctx,
		func(ctx context.Context,
			tpLogoff []commtypes.TpLogOff,
			unprocessed []commtypes.ChkptMetaData,
			snapshot []*commtypes.KeyValuePair[K, V],
		) error {
			out, b, err := encodeKVSnapshot[K, V](kvstore, snapshot, payloadSerde)
			defer func() {
				if payloadSerde.UsedBufferPool() && b != nil {
					*b = out
					commtypes.PushBuffer(b)
				}
			}()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "kv snapshot size: %d, store snapshot at %x\n", len(out), tpLogoff[0].LogOff)
			tp := fmt.Sprintf("%s-%d", kvstore.ChangelogTopicName(), kvstore.SubstreamNum())
			return rs.StoreSnapshot(ctx, out, tp, tpLogoff[0].LogOff, kvstore.SubstreamNum())
		})
}

func SetWinStoreWithChangelogSnapshot[K, V any](
	ctx context.Context,
	rs *snapshot_store.RedisSnapshotStore,
	winStore store.WindowStoreBackedByChangelogG[K, V],
	payloadSerde commtypes.SerdeG[commtypes.PayloadArr],
) {
	winStore.SetWinSnapshotCallback(ctx,
		func(ctx context.Context, tpLogOff []commtypes.TpLogOff,
			chkptMeta []commtypes.ChkptMetaData,
			snapshot []*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V],
		) error {
			out, b, err := encodeWinSnapshot[K, V](winStore, snapshot, payloadSerde)
			defer func() {
				if payloadSerde.UsedBufferPool() && b != nil {
					*b = out
					commtypes.PushBuffer(b)
				}
			}()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "win snapshot size: %d, store snapshot at %x\n", len(out), tpLogOff[0].LogOff)
			tp := fmt.Sprintf("%s-%d", winStore.ChangelogTopicName(), winStore.SubstreamNum())
			return rs.StoreSnapshot(ctx, out, tp, tpLogOff[0].LogOff, winStore.SubstreamNum())
		})
}

func SetupManagersForEpoch(ctx context.Context,
	args *StreamTaskArgs, rs *snapshot_store.RedisSnapshotStore,
) (*epoch_manager.EpochManager, *control_channel.ControlChannelManager, error) {
	checkStreamArgs(args)
	em, err := epoch_manager.NewEpochManager(args.transactionalId, args.serdeFormat)
	if err != nil {
		return nil, nil, err
	}
	cmm, err := control_channel.NewControlChannelManager(args.appId,
		args.serdeFormat, args.bufMaxSize, args.ectx.CurEpoch(), args.ectx.SubstreamNum())
	if err != nil {
		return nil, nil, err
	}
	debug.Fprintf(os.Stderr, "[%d] start restore potential mapping\n", args.ectx.SubstreamNum())
	err = cmm.RestoreMappingAndWaitForPrevTask(
		ctx, args.ectx.FuncName(), env_config.CREATE_SNAPSHOT, args.serdeFormat,
		args.kvChangelogs, args.windowStoreChangelogs, rs)
	if err != nil {
		return nil, nil, fmt.Errorf("RestoreMappingAndWaitForPrevTask: %v", err)
	}
	fmt.Fprintf(os.Stderr, "[%d] restore potential mapping done\n", args.ectx.SubstreamNum())
	initEmStart := time.Now()
	recentMeta, rawMetaMsg, err := em.Init(ctx)
	if err != nil {
		return nil, nil, err
	}
	initEmElapsed := time.Since(initEmStart)
	fmt.Fprintf(os.Stderr, "[%d] Init EpochManager took %v, task epoch %#x, task id %#x\n",
		args.ectx.SubstreamNum(), initEmElapsed, em.GetCurrentEpoch(), em.GetCurrentTaskId())
	trackStream := func(name string, stream *sharedlog_stream.ShardedSharedLogStream) {
		em.RecordTpHashes(name, stream)
	}
	configChangelogExactlyOnce(em, args, trackStream)
	lastMark := uint64(0)
	if recentMeta != nil {
		restoreBeg := time.Now()
		offsetMap := recentMeta.ConSeqNums
		auxData := rawMetaMsg.AuxData
		auxMetaSeq := rawMetaMsg.LogSeqNum
		fmt.Fprintf(os.Stderr, "[%d] start restore, recent AuxData: %v, recent auxMetaSeq: %#x\n",
			args.ectx.SubstreamNum(), auxData, auxMetaSeq)
		if len(offsetMap) == 0 {
			// the last commit doesn't consume anything
			var meta *commtypes.EpochMarker
			var rawMsg *commtypes.RawMsg
			meta, rawMsg, err = em.FindLastEpochMetaThatHasConsumed(ctx, rawMetaMsg.LogSeqNum)
			if err != nil {
				return nil, nil, err
			}
			if len(auxData) == 0 && len(rawMsg.AuxData) != 0 {
				auxData = rawMsg.AuxData
				auxMetaSeq = rawMsg.LogSeqNum
			}
			if meta == nil {
				panic("consumed sequnce number must be recorded in epoch meta")
			} else {
				offsetMap = meta.ConSeqNums
			}
		}
		if env_config.CREATE_SNAPSHOT {
			loadSnapBeg := time.Now()
			if len(auxData) == 0 {
				fmt.Fprintf(os.Stderr, "[%d] read back for snapshot from 0x%x\n", args.ectx.SubstreamNum(), rawMetaMsg.LogSeqNum)
				auxData, auxMetaSeq, err = em.FindLastEpochMetaWithAuxData(ctx, rawMetaMsg.LogSeqNum)
				if err != nil {
					return nil, nil, fmt.Errorf("[ERR] FindLastEpochMetaWithAuxData: %v", err)
				}
			}
			fmt.Fprintf(os.Stderr, "[%d] auxData: %v, auxMetaSeq: %#x\n", args.ectx.SubstreamNum(), auxData, auxMetaSeq)
			err = loadSnapshot(ctx, args, auxData, auxMetaSeq, rs)
			if err != nil {
				return nil, nil, err
			}
			loadSnapElapsed := time.Since(loadSnapBeg)
			fmt.Fprintf(os.Stderr, "[%d] load snapshot took %v\n", args.ectx.SubstreamNum(), loadSnapElapsed)
		}
		lastMark = auxMetaSeq
		restoreSsBeg := time.Now()
		err = restoreStateStore(ctx, args)
		if err != nil {
			return nil, nil, err
		}
		restoreStateStoreElapsed := time.Since(restoreSsBeg)
		setOffsetOnStream(offsetMap, args)
		restoreElapsed := time.Since(restoreBeg)
		fmt.Fprintf(os.Stderr, "restore state store took %v, restore elapsed: %v\n",
			restoreStateStoreElapsed, restoreElapsed)
	}
	fmt.Fprintf(os.Stderr, "[%d] %s down restore, epoch: %d ts: %d\n",
		args.ectx.SubstreamNum(), args.ectx.FuncName(), args.ectx.CurEpoch(), time.Now().UnixMilli())
	for _, src := range args.ectx.Consumers() {
		if !src.IsInitialSource() {
			src.ConfigExactlyOnce(args.guarantee)
		}
		trackStream(src.TopicName(),
			src.Stream().(*sharedlog_stream.ShardedSharedLogStream))
	}
	for _, p := range args.ectx.Producers() {
		p.SetLastMarkerSeq(lastMark)
		p.ConfigExactlyOnce(em, args.guarantee)
		trackStream(p.TopicName(), p.Stream().(*sharedlog_stream.ShardedSharedLogStream))
	}
	for _, kvs := range args.kvChangelogs {
		kvs.SetLastMarkerSeq(lastMark)
	}
	for _, ws := range args.windowStoreChangelogs {
		ws.SetLastMarkerSeq(lastMark)
	}
	trackParFunc := func(
		topicName string, substreamId uint8,
	) {
		em.AddTopicSubstream(topicName, substreamId)
	}
	recordFinish := func(ctx context.Context, funcName string, instanceID uint8) error {
		return cmm.RecordPrevInstanceFinish(ctx, funcName, instanceID, args.ectx.CurEpoch())
	}
	updateFuncs(args, trackParFunc, recordFinish, func(ctx context.Context) error { return nil })
	return em, cmm, nil
}

func setLastMarkSeq(logOff uint64, args *StreamTaskArgs) {
	for _, p := range args.ectx.Producers() {
		p.SetLastMarkerSeq(logOff)
	}
	for _, kvs := range args.kvChangelogs {
		kvs.SetLastMarkerSeq(logOff)
	}
	for _, ws := range args.windowStoreChangelogs {
		ws.SetLastMarkerSeq(logOff)
	}
}

func checkForTimeout(
	args *StreamTaskArgs,
	warmupCheck *stats.Warmup,
	timeoutPrintOnce *sync.Once,
	lastPrint *time.Time,
) (time.Duration, bool) {
	cur_elapsed := warmupCheck.ElapsedSinceInitial()
	timeout := args.duration != 0 && cur_elapsed >= args.duration
	if timeout {
		timeoutPrintOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "timeout after %v\n", cur_elapsed)
			for _, src := range args.ectx.Consumers() {
				fmt.Fprintf(os.Stderr, "%s msgCnt %d, ctrlCnt %d, epochCnt %d, logEntry %d\n",
					src.Name(), src.GetCount(), src.NumCtrlMsg(), src.NumEpoch(), src.NumLogEntry())
			}
			*lastPrint = time.Now()
		})
		print_elapsed := time.Since(*lastPrint)
		if print_elapsed > 20*time.Second {
			fmt.Fprintf(os.Stderr, "timeout after %v\n", cur_elapsed)
			for _, src := range args.ectx.Consumers() {
				fmt.Fprintf(os.Stderr, "%s msgCnt %d, ctrlCnt %d, epochCnt %d, logEntry %d\n",
					src.Name(), src.GetCount(), src.NumCtrlMsg(), src.NumEpoch(), src.NumLogEntry())
			}
			*lastPrint = time.Now()
		}
	}
	return cur_elapsed, timeout
}

func pausedFlushMark(
	ctx context.Context, meta *epochProcessMeta,
	snapshotTime *[]int64, snapshotTimer *time.Time,
	paused *bool,
) (bool, *common.FnOutput) {
	if meta.t.pauseFunc != nil {
		if ret := meta.t.pauseFunc(meta.args.guarantee); ret != nil {
			return false, ret
		}
		*paused = true
	}

	markBegin := time.Now()
	flushAllStart := stats.TimerBegin()
	f, err := flushStreams(ctx, meta.t, meta.args)
	if err != nil {
		return false, common.GenErrFnOutput(fmt.Errorf("flushStreams: %v", err))
	}
	flushTime := stats.Elapsed(flushAllStart).Microseconds()

	mPartBeg := time.Now()
	logOff, shouldExit, err := markEpoch(ctx, meta)
	if err != nil {
		return false, common.GenErrFnOutput(fmt.Errorf("markEpoch: %v", err))
	}
	setLastMarkSeq(logOff, meta.args)
	if env_config.CREATE_SNAPSHOT && meta.args.snapshotEvery != 0 && time.Since(*snapshotTimer) > meta.args.snapshotEvery {
		snStart := time.Now()
		createSnapshot(ctx, meta.args, []commtypes.TpLogOff{{LogOff: logOff}})
		elapsed := time.Since(snStart)
		*snapshotTime = append(*snapshotTime, elapsed.Microseconds())
		*snapshotTimer = time.Now()
	}
	markElapsed := time.Since(markBegin)
	mPartElapsed := time.Since(mPartBeg)

	meta.t.epochMarkTime.AddSample(markElapsed.Microseconds())
	meta.t.flushStageTime.AddSample(flushTime)
	meta.t.markPartUs.AddSample(mPartElapsed.Microseconds())
	if f > 0 {
		meta.t.flushAtLeastOne.AddSample(flushTime)
	}
	meta.t.epochMarkTimes += 1
	return shouldExit, nil
}

type epochProcessMeta struct {
	t    *StreamTask
	em   *epoch_manager.EpochManager
	cmm  *control_channel.ControlChannelManager
	args *StreamTaskArgs
}

func processInEpoch(
	ctx context.Context,
	meta *epochProcessMeta,
) *common.FnOutput {
	// trackStreamAndConfigureExactlyOnce(meta.args, meta.em,
	// 	func(name string, stream *sharedlog_stream.ShardedSharedLogStream) {
	// 		meta.em.RecordTpHashes(name, stream)
	// 	})

	// execIntrMs := stats.NewPrintLogStatsCollector[int64]("execIntrMs")
	// thisAndLastCmtMs := stats.NewPrintLogStatsCollector[int64]("thisAndLastCmtMs")
	snapshotTime := make([]int64, 0, 20)
	hasProcessData := false
	init := false
	paused := false
	// latencies := stats.NewInt64Collector("latPerIter", stats.DEFAULT_COLLECT_DURATION)
	markTimer := time.Now()
	snapshotTimer := time.Now()
	var lastPrint time.Time
	var once sync.Once
	warmupCheck := stats.NewWarmupChecker(meta.args.warmup)
	fmt.Fprintf(os.Stderr, "commit every(ms): %v, waitEndMark: %v, fixed output parNum: %d, snapshot every(s): %v, sink buf max: %v\n",
		meta.args.commitEvery, meta.args.waitEndMark, meta.args.fixedOutParNum, meta.args.snapshotEvery, meta.args.bufMaxSize)
	testForFail := false
	var failAfter time.Duration
	failParam, ok := meta.args.testParams[meta.args.ectx.FuncName()]
	if ok && failParam.InstanceId == meta.args.ectx.SubstreamNum() {
		testForFail = true
		failAfter = time.Duration(failParam.FailAfterS) * time.Second
		fmt.Fprintf(os.Stderr, "%s(%d) will fail after %v\n",
			meta.args.ectx.FuncName(), meta.args.ectx.SubstreamNum(), failAfter)
	}
	var timeoutPrintOnce sync.Once
	for {
		// procStart := time.Now()
		once.Do(func() {
			warmupCheck.StartWarmup()
		})
		warmupCheck.Check()
		timeSinceLastMark := time.Since(markTimer)
		shouldMarkByTime := meta.args.commitEvery != 0 && timeSinceLastMark >= meta.args.commitEvery
		if shouldMarkByTime && hasProcessData {
			// execIntrMs.AddSample(timeSinceLastMark.Milliseconds())
			shouldExit, fn_out := pausedFlushMark(ctx, meta, &snapshotTime, &snapshotTimer, &paused)
			if fn_out != nil {
				return fn_out
			}
			if shouldExit {
				ret := &common.FnOutput{Success: true}
				if testForFail {
					ret = &common.FnOutput{Success: true, Message: common_errors.ErrReturnDueToTest.Error()}
				}
				fmt.Fprintf(os.Stderr, "epoch_mark_times: %d\n", meta.t.epochMarkTimes)
				meta.t.PrintRemainingStats()
				// execIntrMs.PrintRemainingStats()
				// thisAndLastCmtMs.PrintRemainingStats()
				updateReturnMetric(ret, &warmupCheck,
					false, meta.t.GetEndDuration(), meta.args.ectx.SubstreamNum())
				return ret
			}
			hasProcessData = false
			// btwThisLastCmt := time.Since(markTimer)
			// thisAndLastCmtMs.AddSample(btwThisLastCmt.Milliseconds())
			markTimer = time.Now()
		}
		// Exit routine
		cur_elapsed, timeout := checkForTimeout(meta.args, &warmupCheck, &timeoutPrintOnce, &lastPrint)
		exitDueToFailTest := testForFail && cur_elapsed >= failAfter
		if (!meta.args.waitEndMark && timeout) || exitDueToFailTest {
			r := finalMark(ctx, meta, snapshotTime, exitDueToFailTest)
			if r != nil {
				return r
			}
			ret := &common.FnOutput{Success: true}
			if exitDueToFailTest {
				ret = &common.FnOutput{Success: true, Message: common_errors.ErrReturnDueToTest.Error()}
			}
			meta.t.PrintRemainingStats()
			// execIntrMs.PrintRemainingStats()
			// thisAndLastCmtMs.PrintRemainingStats()
			updateReturnMetric(ret, &warmupCheck,
				false, meta.t.GetEndDuration(), meta.args.ectx.SubstreamNum())
			return ret
		}

		// init
		if !hasProcessData && (!init || paused) {
			err := initAfterMarkOrCommit(meta.t, meta.args, meta.em, &init, &paused)
			if err != nil {
				return common.GenErrFnOutput(fmt.Errorf("initAfterMarkOrCommit: %v", err))
			}
		}
		app_ret, ctrlRawMsgArr := meta.t.appProcessFunc(ctx, meta.t, meta.args.ectx)
		if app_ret != nil {
			if app_ret.Success {
				debug.Assert(ctrlRawMsgArr == nil, "when timeout, ctrlMsg should not be returned")
				// consume timeout but not sure whether there's more data is coming; continue to process
				continue
			}
			return app_ret
		}
		if !hasProcessData {
			hasProcessData = true
		}
		if ctrlRawMsgArr != nil {
			fmt.Fprintf(os.Stderr, "exit due to ctrlMsg\n")
			r := finalMark(ctx, meta, snapshotTime, false)
			if r != nil {
				return r
			}
			// execIntrMs.PrintRemainingStats()
			// thisAndLastCmtMs.PrintRemainingStats()
			return handleCtrlMsg(ctx, ctrlRawMsgArr, meta.t, meta.args, &warmupCheck, nil)
		}
	}
}

func finalMark(
	dctx context.Context,
	meta *epochProcessMeta,
	snapshotTime []int64, exitDueToFailTest bool,
) *common.FnOutput {
	markBegin := time.Now()
	if meta.t.pauseFunc != nil {
		if ret := meta.t.pauseFunc(meta.args.guarantee); ret != nil {
			return ret
		}
	}
	flushAllStart := stats.TimerBegin()
	f, err := flushStreams(dctx, meta.t, meta.args)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("flushStreams failed: %v", err))
	}
	flushTime := stats.Elapsed(flushAllStart).Microseconds()
	kms, err := getKeyMapping(dctx, meta.args)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("getKeyMapping failed: %v", err))
	}
	mPartBeg := time.Now()
	logOff, _, err := markEpoch(dctx, meta)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("markEpoch failed: %v", err))
	}
	fmt.Fprintf(os.Stderr, "[%d] final mark seqNum: %#x\n", meta.args.ectx.SubstreamNum(), logOff)
	if env_config.CREATE_SNAPSHOT && meta.args.snapshotEvery != 0 && !exitDueToFailTest {
		snStart := time.Now()
		createSnapshot(dctx, meta.args, []commtypes.TpLogOff{{LogOff: logOff}})
		for _, kv := range meta.args.kvChangelogs {
			err = kv.WaitForAllSnapshot()
			if err != nil {
				fmt.Fprintf(os.Stderr, "KV WaitForAllSnapshot failed: %v\n", err)
				// return common.GenErrFnOutput(fmt.Errorf("KV WaitForAllSnapshot failed: %v", err))
			}
		}
		for _, wsc := range meta.args.windowStoreChangelogs {
			err = wsc.WaitForAllSnapshot()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Win WaitForAllSnapshot failed: %v\n", err)
				// return common.GenErrFnOutput(fmt.Errorf("Win WaitForAllSnapshot failed: %v", err))
			}
		}
		elapsed := time.Since(snStart)
		snapshotTime = append(snapshotTime, elapsed.Microseconds())
		fmt.Fprintf(os.Stderr, "final snapshot time: %v\n", snapshotTime)
	}
	err = meta.cmm.OutputKeyMapping(dctx, kms)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("OutputKeyMapping failed: %v", err))
	}
	fmt.Fprintf(os.Stderr, "[%d] %s final mark, epoch %d, ts %d\n",
		meta.args.ectx.SubstreamNum(), meta.args.ectx.FuncName(),
		meta.cmm.CurrentEpoch(), time.Now().UnixMilli())
	markElapsed := time.Since(markBegin)
	mPartElapsed := time.Since(mPartBeg)
	meta.t.epochMarkTime.AddSample(markElapsed.Microseconds())
	meta.t.markPartUs.AddSample(mPartElapsed.Microseconds())
	meta.t.flushStageTime.AddSample(flushTime)
	if f > 0 {
		meta.t.flushAtLeastOne.AddSample(flushTime)
	}
	fmt.Fprintf(os.Stderr, "[%d] epoch_mark_times: %d\n", meta.args.ectx.SubstreamNum(), meta.t.epochMarkTimes)
	meta.t.PrintRemainingStats()
	return nil
}

func getKeyMapping(ctx context.Context, args *StreamTaskArgs) (map[string][]txn_data.KeyMaping, error) {
	kms := make(map[string][]txn_data.KeyMaping)
	for _, kv := range args.kvChangelogs {
		err := kv.BuildKeyMeta(ctx, kms)
		if err != nil {
			return nil, err
		}
	}
	for _, wsc := range args.windowStoreChangelogs {
		err := wsc.BuildKeyMeta(kms)
		if err != nil {
			return nil, err
		}
	}
	return kms, nil
}

func markEpoch(ctx context.Context,
	meta *epochProcessMeta,
) (logOff uint64, shouldExit bool, err error) {
	prepareStart := stats.TimerBegin()
	// should rely on conditional append
	// rawMsgs, err := meta.em.SyncToRecent(ctx)
	// if err != nil {
	// 	return 0, false, fmt.Errorf("SyncToRecent: %v", err)
	// }
	// for _, rawMsg := range rawMsgs {
	// 	if rawMsg.Mark == commtypes.FENCE {
	// 		if (rawMsg.ProdId.TaskId == meta.em.GetCurrentTaskId() && rawMsg.ProdId.TaskEpoch > meta.em.GetCurrentEpoch()) ||
	// 			rawMsg.ProdId.TaskId != meta.em.GetCurrentTaskId() {
	// 			return 0, true, nil
	// 		}
	// 	}
	// }
	epochMarker, err := epoch_manager.GenEpochMarker(ctx, meta.em, meta.args.ectx,
		meta.args.kvChangelogs, meta.args.windowStoreChangelogs)
	if err != nil {
		return 0, false, fmt.Errorf("GenEpochMarker: %v", err)
	}
	epochMarkerTags, epochMarkerTopics := meta.em.GenTagsAndTopicsForEpochMarker()
	prepareTime := stats.Elapsed(prepareStart).Microseconds()

	mStart := stats.TimerBegin()
	logOff, markerSize, err := meta.em.MarkEpoch(ctx, epochMarker, epochMarkerTags, epochMarkerTopics)
	if err != nil {
		return 0, false, fmt.Errorf("MarkEpoch: %v", err)
	}
	epoch_manager.CleanupState(meta.em, meta.args.ectx.Producers(), meta.args.kvChangelogs, meta.args.windowStoreChangelogs)
	mElapsed := stats.Elapsed(mStart).Microseconds()
	meta.t.markerSize.AddSample(markerSize)
	meta.t.markEpochAppend.AddSample(mElapsed)
	meta.t.markEpochPrepare.AddSample(prepareTime)
	return logOff, false, nil
}
