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
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_restore"
	"sharedlog-stream/pkg/transaction"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type ProcessFunc func(ctx context.Context, task *StreamTask,
	args processor.ExecutionContext) (*common.FnOutput, []*commtypes.RawMsgAndSeq)

// in case the task consumes multiple streams, the task consumes from the same substream number
// and the substreams must have the same number of substreams.
type StreamTask struct {
	appProcessFunc ProcessFunc
	pauseFunc      PauseFuncType
	resumeFunc     ResumeFuncType
	initFunc       func(task *StreamTask)
	HandleErrFunc  func() error

	flushStageTime    stats.PrintLogStatsCollector[int64]
	flushAtLeastOne   stats.PrintLogStatsCollector[int64]
	commitTxnAPITime  stats.PrintLogStatsCollector[int64]
	sendOffsetTime    stats.PrintLogStatsCollector[int64]
	txnCommitTime     stats.PrintLogStatsCollector[int64]
	markPartUs        stats.PrintLogStatsCollector[int64]
	epochMarkTime     stats.PrintLogStatsCollector[int64]
	markEpochPrepare  stats.PrintLogStatsCollector[int64]
	markEpochAppend   stats.PrintLogStatsCollector[int64]
	waitPrevTxnInCmt  stats.PrintLogStatsCollector[int64]
	waitPrevTxnInPush stats.PrintLogStatsCollector[int64]
	markerSize        stats.PrintLogStatsCollector[int]
	producerFlush     stats.PrintLogStatsCollector[int64]
	kvcFlush          stats.PrintLogStatsCollector[int64]
	wscFlush          stats.PrintLogStatsCollector[int64]
	finishChkpt       stats.StatsCollector[int64]

	txnCounter                 stats.Counter
	waitedInCmtCounter         stats.Counter
	waitedInPushCounter        stats.AtomicCounter
	appendedMetaInCmtCounter   stats.Counter
	appendedMetaInPushCounter  stats.AtomicCounter
	waitAndAppendInCmtCounter  stats.Counter
	waitAndAppendInPushCounter stats.AtomicCounter

	endDuration    time.Duration
	epochMarkTimes uint32
	isFinalStage   bool
}

func (t *StreamTask) PrintRemainingStats() {
	t.flushStageTime.PrintRemainingStats()
	t.flushAtLeastOne.PrintRemainingStats()
	t.commitTxnAPITime.PrintRemainingStats()
	t.sendOffsetTime.PrintRemainingStats()
	t.txnCommitTime.PrintRemainingStats()
	t.markPartUs.PrintRemainingStats()
	t.epochMarkTime.PrintRemainingStats()
	t.markEpochPrepare.PrintRemainingStats()
	t.markEpochAppend.PrintRemainingStats()
	t.waitPrevTxnInCmt.PrintRemainingStats()
	t.waitPrevTxnInPush.PrintRemainingStats()
	t.markerSize.PrintRemainingStats()
	t.producerFlush.PrintRemainingStats()
	t.kvcFlush.PrintRemainingStats()
	t.wscFlush.PrintRemainingStats()
	t.finishChkpt.PrintRemainingStats()
	if t.txnCounter.GetCount() != 0 {
		t.txnCounter.Report()
		t.waitedInCmtCounter.Report()
		t.waitedInPushCounter.Report()
		t.appendedMetaInCmtCounter.Report()
		t.appendedMetaInPushCounter.Report()
		t.waitAndAppendInCmtCounter.Report()
		t.waitAndAppendInPushCounter.Report()

		total := float64(t.txnCounter.GetCount())
		waitedInCmt := float64(t.waitedInCmtCounter.GetCount())
		appendedInCmt := float64(t.appendedMetaInCmtCounter.GetCount())
		waitAndAppendInCmt := float64(t.waitAndAppendInCmtCounter.GetCount())

		waitedInPush := float64(t.waitedInPushCounter.GetCount())
		appendedInPush := float64(t.appendedMetaInPushCounter.GetCount())
		waitAndAppendInPush := float64(t.waitAndAppendInPushCounter.GetCount())

		fmt.Fprintf(os.Stderr,
			"waitedInCmt: %v, appendedInCmt: %v, waitAndAppendInCmt: %v, waitedInPush: %v, appendedInPush: %v, waitAndAppendInPush: %v\n",
			waitedInCmt/total, appendedInCmt/total, waitAndAppendInCmt/total,
			waitedInPush/total, appendedInPush/total, waitAndAppendInPush/total)
	}
}

func (t *StreamTask) SetEndDuration(startTimeMs int64) {
	t.endDuration = time.Since(time.UnixMilli(startTimeMs))
}

func (t *StreamTask) GetEndDuration() time.Duration {
	return t.endDuration
}

type SetupSnapshotCallbackFunc func(ctx context.Context, serdeFormat commtypes.SerdeFormat,
	rs snapshot_store.SnapshotStore) error

func EmptySetupSnapshotCallback(ctx context.Context, serdeFormat commtypes.SerdeFormat,
	rs snapshot_store.SnapshotStore,
) error {
	return nil
}

func ExecuteApp(ctx context.Context,
	t *StreamTask,
	streamTaskArgs *StreamTaskArgs,
	setupSnapshotCallback SetupSnapshotCallbackFunc,
	outputRemainingStats func(),
) *common.FnOutput {
	var ret *common.FnOutput
	// var mc *snapshot_store.MinioChkptStore
	var rs snapshot_store.RedisSnapshotStore
	var snap_store snapshot_store.SnapshotStore
	var err error
	if streamTaskArgs.guarantee == exactly_once_intr.TWO_PHASE_COMMIT ||
		streamTaskArgs.guarantee == exactly_once_intr.EPOCH_MARK ||
		streamTaskArgs.guarantee == exactly_once_intr.AT_LEAST_ONCE ||
		streamTaskArgs.guarantee == exactly_once_intr.REMOTE_2PC ||
		streamTaskArgs.guarantee == exactly_once_intr.ALIGN_CHKPT {
		create := env_config.CREATE_SNAPSHOT
		if streamTaskArgs.guarantee == exactly_once_intr.ALIGN_CHKPT {
			// var err error
			create = true
			// mc, err = snapshot_store.NewMinioChkptStore()
			// if err != nil {
			// 	return common.GenErrFnOutput(err)
			// }
		}
		rs = snapshot_store.NewRedisSnapshotStore(create)
		snap_store = &rs
		// if streamTaskArgs.guarantee == exactly_once_intr.ALIGN_CHKPT {
		// 	snap_store = mc
		// } else {
		// 	snap_store = &rs
		// }
		err = setupSnapshotCallback(ctx, streamTaskArgs.serdeFormat, snap_store)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		debug.Fprintf(os.Stderr, "[%d] done setup snapshot\n", streamTaskArgs.ectx.SubstreamNum())
	}
	if streamTaskArgs.guarantee == exactly_once_intr.TWO_PHASE_COMMIT {
		meta := txnProcessMeta{
			t:    t,
			args: streamTaskArgs,
		}
		meta.tm, meta.cmm, err = SetupManagersFor2pc(ctx, t, streamTaskArgs, &rs)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		prodId := meta.tm.GetProducerId()
		fmt.Fprintf(os.Stderr, "[%d] prodId: %s\n", meta.args.ectx.SubstreamNum(), prodId.String())
		debug.Fprint(os.Stderr, "begin transaction processing\n")
		ret = processWithTransaction(ctx, &meta)
		debug.Fprintf(os.Stderr, "2pc ret: %v\n", ret)
	} else if streamTaskArgs.guarantee == exactly_once_intr.REMOTE_2PC {
		meta := rtxnProcessMeta{
			t:    t,
			args: streamTaskArgs,
		}
		meta.rtm_client, meta.cmm, err = SetupManagerForRemote2pc(ctx, t, streamTaskArgs, &rs)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		prodId := meta.rtm_client.GetProducerId()
		fmt.Fprintf(os.Stderr, "[%d] prodId: %s\n", meta.args.ectx.SubstreamNum(), prodId.String())
		debug.Fprint(os.Stderr, "begin remote 2pc processing\n")
		ret = processWithRTxnMngr(ctx, &meta)
		debug.Fprintf(os.Stderr, "remote 2pc ret: %v\n", ret)
	} else if streamTaskArgs.guarantee == exactly_once_intr.EPOCH_MARK {
		meta := epochProcessMeta{
			t:    t,
			args: streamTaskArgs,
		}
		meta.em, meta.cmm, err = SetupManagersForEpoch(ctx, streamTaskArgs, &rs)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		prodId := meta.em.GetProducerId()
		fmt.Fprintf(os.Stderr, "[%d] prodId: %s\n", meta.args.ectx.SubstreamNum(), prodId.String())
		debug.Fprint(os.Stderr, "begin epoch processing\n")
		ret = processInEpoch(ctx, &meta)
		debug.Fprintf(os.Stderr, "epoch ret: %v\n", ret)
	} else if streamTaskArgs.guarantee == exactly_once_intr.AT_LEAST_ONCE {
		debug.Fprint(os.Stderr, "begin at least once epoch processing\n")
		ret = process(ctx, t, streamTaskArgs)
		debug.Fprintf(os.Stderr, "at least once return: %v\n", ret)
	} else if streamTaskArgs.guarantee == exactly_once_intr.NO_GUARANTEE {
		debug.Fprintf(os.Stderr, "begin processing without guarantee\n")
		ret = processNoProto(ctx, t, streamTaskArgs)
		debug.Fprintf(os.Stderr, "unsafe ret: %v\n", ret)
	} else if streamTaskArgs.guarantee == exactly_once_intr.ALIGN_CHKPT {
		debug.Fprintf(os.Stderr, "begin align checkpoint processing\n")
		cmm, err := control_channel.NewControlChannelManager(
			streamTaskArgs.appId,
			streamTaskArgs.serdeFormat, streamTaskArgs.bufMaxSize,
			streamTaskArgs.ectx.CurEpoch(), streamTaskArgs.ectx.SubstreamNum())
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		err = cmm.RestoreMappingAndWaitForPrevTask(
			ctx, streamTaskArgs.ectx.FuncName(), env_config.CREATE_SNAPSHOT, streamTaskArgs.serdeFormat,
			streamTaskArgs.kvChangelogs, streamTaskArgs.windowStoreChangelogs, &rs)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		recordFinish := func(ctx context.Context, funcName string, instanceID uint8) error {
			return cmm.RecordPrevInstanceFinish(ctx, funcName, instanceID, streamTaskArgs.ectx.CurEpoch())
		}
		streamTaskArgs.ectx.SetRecordFinishFunc(recordFinish)
		ret = processAlignChkpt(ctx, t, streamTaskArgs, rs.GetRedisClients(), &rs)
		debug.Fprintf(os.Stderr, "align chkpt ret: %v\n", ret)
	} else {
		fmt.Fprintf(os.Stderr, "unrecognized guarantee: %v\n", streamTaskArgs.guarantee)
		return &common.FnOutput{Success: false, Message: "unrecognized guarantee"}
	}
	outputRemainingStats()
	rs.PrintRemainingStats()
	for _, kvc := range streamTaskArgs.kvChangelogs {
		kvc.OutputRemainingStats()
	}
	for _, wsc := range streamTaskArgs.windowStoreChangelogs {
		wsc.OutputRemainingStats()
	}
	for _, src := range streamTaskArgs.ectx.Consumers() {
		src.OutputRemainingStats()
		fmt.Fprintf(os.Stderr, "%s msgCnt %d, ctrlCnt %d, epochCnt %d, logEntry %d\n",
			src.Name(), src.GetCount(), src.NumCtrlMsg(), src.NumEpoch(), src.NumLogEntry())
		if ret.Counts == nil {
			ret.Counts = make(map[string]uint64)
		}
		ret.Counts[src.Name()] = src.GetCount()
		ret.Counts[src.Name()+"_ctrl"] = uint64(src.NumCtrlMsg())
		ret.Counts[src.Name()+"_epoch"] = uint64(src.NumEpoch())
		ret.Counts[src.Name()+"_data"] = src.GetCount() - uint64(src.NumCtrlMsg()) - uint64(src.NumEpoch())
		ret.Counts[src.Name()+"_logEntry"] = src.NumLogEntry()
	}
	for _, sink := range streamTaskArgs.ectx.Producers() {
		sink.OutputRemainingStats()
		ret.Counts[sink.Name()] = sink.GetCount()
		// ret.Counts[sink.Name()+"_ctrl"] = uint64(sink.NumCtrlMsg())
		fmt.Fprintf(os.Stderr, "%s msgCnt %d, ctrlCnt %d\n",
			sink.Name(), sink.GetCount(), sink.NumCtrlMsg())
	}
	if ret != nil && ret.Success {
		for _, sink := range streamTaskArgs.ectx.Producers() {
			if sink.IsFinalOutput() {
				ret.Latencies["eventTimeLatency_"+sink.Name()] = sink.GetEventTimeLatency()
				ret.EventTs = sink.GetEventTs()
			}
		}
	}
	return ret
}

func encodeKVSnapshot[K, V any](
	kvstore store.CoreKeyValueStoreG[K, V],
	snapshot []*commtypes.KeyValuePair[K, V],
	payloadSerde commtypes.SerdeG[commtypes.PayloadArr],
) ([]byte, *[]byte, error) {
	kvPairSerdeG := kvstore.GetKVSerde()
	return commtypes.EncodeKVPairs(snapshot, payloadSerde, kvPairSerdeG)
}

func encodeWinSnapshot[K, V any](
	winStore store.CoreWindowStoreG[K, V],
	snapshot []*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V],
	payloadSerde commtypes.SerdeG[commtypes.PayloadArr],
) ([]byte, *[]byte, error) {
	kvPairSerdeG := winStore.GetKVSerde()
	return commtypes.EncodeKVPairs(snapshot, payloadSerde, kvPairSerdeG)
}

func handleCtrlMsg(
	ctx context.Context,
	ctrlRawMsgArr []*commtypes.RawMsgAndSeq,
	t *StreamTask,
	args *StreamTaskArgs,
	warmupCheck *stats.Warmup,
	// rs *snapshot_store.MinioChkptStore,
	rs *snapshot_store.RedisSnapshotStore,
) *common.FnOutput {
	if ctrlRawMsgArr[0].Mark == commtypes.SCALE_FENCE {
		ret := handleScaleEpochAndBytes(ctx, ctrlRawMsgArr[0], args)
		if ret.Success {
			updateReturnMetric(ret, warmupCheck,
				false, t.GetEndDuration(), args.ectx.SubstreamNum())
		}
		return ret
	} else if ctrlRawMsgArr[0].Mark == commtypes.CHKPT_MARK && args.guarantee != exactly_once_intr.ALIGN_CHKPT {
		return common.GenErrFnOutput(common_errors.ErrChkptMarkerInvalidGuarantee)
	} else if ctrlRawMsgArr[0].Mark == commtypes.STREAM_END {
		epochMarker := commtypes.EpochMarker{
			StartTime: ctrlRawMsgArr[0].StartTime,
			Mark:      commtypes.STREAM_END,
			ProdIndex: args.ectx.SubstreamNum(),
		}
		encoded, b, err := args.epochMarkerSerde.Encode(epochMarker)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		ctrlRawMsgArr[0].Payload = encoded
		err = forwardCtrlMsg(ctx, ctrlRawMsgArr[0], args, "stream end mark")
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		if args.epochMarkerSerde.UsedBufferPool() && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
		if args.guarantee == exactly_once_intr.ALIGN_CHKPT {
			tpLogOff := make([]commtypes.TpLogOff, 0, len(args.ectx.Consumers()))
			// use the last stream end marker seqnum here as this is the end of a stream
			// there's no record after this record.
			for idx, c := range args.ectx.Consumers() {
				tpLogOff = append(tpLogOff, commtypes.TpLogOff{
					Tp:     fmt.Sprintf("%s-%d", c.Stream().TopicName(), args.ectx.SubstreamNum()),
					LogOff: ctrlRawMsgArr[idx].LogSeqNum,
				})
			}
			// the last checkpoint will be wait here.
			err := createChkpt(ctx, args, tpLogOff, nil, rs)
			if err != nil {
				return common.GenErrFnOutput(err)
			}
		}
		t.SetEndDuration(ctrlRawMsgArr[0].StartTime)
		ret := &common.FnOutput{Success: true}
		updateReturnMetric(ret, warmupCheck,
			args.waitEndMark, t.GetEndDuration(), args.ectx.SubstreamNum())
		return ret
	} else {
		return common.GenErrFnOutput(
			fmt.Errorf("unexpected ctrl msg with mark: %v", ctrlRawMsgArr[0].Mark))
	}
}

func timedFlushStreams(
	ctx context.Context,
	t *StreamTask,
	args *StreamTaskArgs,
) *common.FnOutput {
	flushAllStart := stats.TimerBegin()
	f, ret_err := flushStreams(ctx, t, args)
	if ret_err != nil {
		return common.GenErrFnOutput(ret_err)
	}
	flushTime := stats.Elapsed(flushAllStart).Microseconds()
	t.flushStageTime.AddSample(flushTime)
	if f > 0 {
		t.flushAtLeastOne.AddSample(flushTime)
	}
	return nil
}

func pauseTimedFlushStreams(
	ctx context.Context,
	t *StreamTask,
	args *StreamTaskArgs,
) *common.FnOutput {
	if t.pauseFunc != nil {
		if ret := t.pauseFunc(args.guarantee); ret != nil {
			return ret
		}
	}
	return timedFlushStreams(ctx, t, args)
}

func flushSerial[V store.StoreWithFlush](ctx context.Context, v []V) (uint32, int64, error) {
	flushed := uint32(0)
	elapsed := int64(0)
	if len(v) != 0 {
		kvcBeg := stats.TimerBegin()
		for _, kvchangelog := range v {
			f, err := kvchangelog.Flush(ctx)
			if err != nil {
				log.Error().Err(err).Msg("flush store/producer")
				return 0, 0, fmt.Errorf("flush store/producer: %v", err)
			}
			flushed += f
		}
		elapsed = stats.Elapsed(kvcBeg).Microseconds()
	}
	return flushed, elapsed, nil

}

func flushParallel[V store.StoreWithFlush](ctx context.Context, v []V) (uint32, int64, error) {
	flushed := uint32(0)
	elapsed := int64(0)
	if len(v) != 0 {
		kvcBeg := stats.TimerBegin()
		if len(v) > 1 {
			var atomic_flushed atomic.Uint32
			bgGrp, bgCtx := errgroup.WithContext(ctx)
			for _, scl := range v {
				s := scl
				bgGrp.Go(func() error {
					f, err := s.Flush(bgCtx)
					if err != nil {
						log.Error().Err(err).Msg("flush store/producer")
						return fmt.Errorf("flush store/producer: %v", err)
					}
					atomic_flushed.Add(f)
					return nil
				})
			}
			err := bgGrp.Wait()
			if err != nil {
				return 0, 0, err
			}
			flushed += atomic_flushed.Load()
		} else {
			for _, kvchangelog := range v {
				f, err := kvchangelog.Flush(ctx)
				if err != nil {
					log.Error().Err(err).Msg("flush store/producer")
					return 0, 0, fmt.Errorf("flush store/producer: %v", err)
				}
				flushed += f
			}
		}
		elapsed = stats.Elapsed(kvcBeg).Microseconds()
	}
	return flushed, elapsed, nil

}

func flushStoreWithClSerial[V store.StoreBackedByChangelog](ctx context.Context, v map[string]V) (uint32, int64, error) {
	flushed := uint32(0)
	elapsed := int64(0)
	if len(v) != 0 {
		kvcBeg := stats.TimerBegin()
		for _, kvchangelog := range v {
			f, err := kvchangelog.Flush(ctx)
			if err != nil {
				return 0, 0, fmt.Errorf("store with cl flush: %v", err)
			}
			flushed += f
		}
		elapsed = stats.Elapsed(kvcBeg).Microseconds()
	}
	return flushed, elapsed, nil
}

func flushStoreWithCl[V store.StoreBackedByChangelog](ctx context.Context, v map[string]V) (uint32, int64, error) {
	flushed := uint32(0)
	elapsed := int64(0)
	if len(v) != 0 {
		kvcBeg := stats.TimerBegin()
		if len(v) > 1 {
			var atomic_flushed atomic.Uint32
			bgGrp, bgCtx := errgroup.WithContext(ctx)
			for _, scl := range v {
				s := scl
				bgGrp.Go(func() error {
					f, err := s.Flush(bgCtx)
					if err != nil {
						return fmt.Errorf("store with cl flush: %v", err)
					}
					atomic_flushed.Add(f)
					return nil
				})
			}
			err := bgGrp.Wait()
			if err != nil {
				return 0, 0, err
			}
			flushed += atomic_flushed.Load()
		} else {
			for _, kvchangelog := range v {
				f, err := kvchangelog.Flush(ctx)
				if err != nil {
					return 0, 0, fmt.Errorf("store with cl flush: %v", err)
				}
				flushed += f
			}
		}
		elapsed = stats.Elapsed(kvcBeg).Microseconds()
	}
	return flushed, elapsed, nil
}

func flushStreams(ctx context.Context,
	t *StreamTask,
	args *StreamTaskArgs,
) (uint32, error) {
	ser_flushed := uint32(0)
	flushed, elapsed, err := flushStoreWithClSerial(ctx, args.kvChangelogs)
	if err != nil {
		return 0, fmt.Errorf("kvc flush: %v", err)
	}
	ser_flushed += flushed
	if len(args.kvChangelogs) != 0 {
		t.kvcFlush.AddSample(elapsed)
	}
	flushed, elapsed, err = flushStoreWithClSerial(ctx, args.windowStoreChangelogs)
	if err != nil {
		return 0, fmt.Errorf("wsc flush: %v", err)
	}
	ser_flushed += flushed
	if len(args.windowStoreChangelogs) != 0 {
		t.wscFlush.AddSample(elapsed)
	}
	_, _, err = flushSerial(ctx, args.kvs)
	if err != nil {
		return 0, fmt.Errorf("kv flush: %v", err)
	}
	_, _, err = flushSerial(ctx, args.wscs)
	if err != nil {
		return 0, fmt.Errorf("ws flush: %v", err)
	}
	flushed, elapsed, err = flushSerial(ctx, args.ectx.Producers())
	if err != nil {
		return 0, fmt.Errorf("sink flush: %v", err)
	}
	ser_flushed += flushed
	t.producerFlush.AddSample(elapsed)
	return ser_flushed, nil
}

func createSnapshot(ctx context.Context, args *StreamTaskArgs, tplogOff []commtypes.TpLogOff) {
	for _, kvchangelog := range args.kvChangelogs {
		kvchangelog.Snapshot(ctx, tplogOff, nil, false)
	}
	for _, wschangelog := range args.windowStoreChangelogs {
		wschangelog.Snapshot(ctx, tplogOff, nil, false)
	}
}

func createChkpt(ctx context.Context, args *StreamTaskArgs, tplogOff []commtypes.TpLogOff,
	chkptMeta []commtypes.ChkptMetaData,
	// mc *snapshot_store.MinioChkptStore,
	mc *snapshot_store.RedisSnapshotStore,
) error {
	// no stores
	if len(args.kvs) == 0 && len(args.wscs) == 0 {
		if !args.ectx.AllConsumersIsInit() {
			return mc.StoreSrcLogoff(ctx, tplogOff, args.ectx.SubstreamNum())
		} else {
			return nil
		}
	}
	for _, kv := range args.kvs {
		kv.Snapshot(ctx, tplogOff, chkptMeta, true)
	}
	for _, wsc := range args.wscs {
		wsc.Snapshot(ctx, tplogOff, chkptMeta, true)
	}
	for _, kv := range args.kvs {
		err := kv.WaitForAllSnapshot()
		if err != nil {
			return fmt.Errorf("KV WaitForAllSnapshot failed: %v", err)
		}
	}
	for _, wsc := range args.wscs {
		err := wsc.WaitForAllSnapshot()
		if err != nil {
			return fmt.Errorf("Win WaitForAllSnapshot failed: %v", err)
		}
	}
	return nil
}

func updateReturnMetric(ret *common.FnOutput, warmupChecker *stats.Warmup,
	waitForEndMark bool, endDuration time.Duration, instanceID uint8,
) {
	ret.Latencies = make(map[string][]int)
	ret.Counts = make(map[string]uint64)
	if waitForEndMark {
		ret.Duration = endDuration.Seconds()
	} else {
		ret.Duration = warmupChecker.ElapsedAfterWarmup().Seconds()
	}
	fmt.Fprintf(os.Stderr, "[%d]duration: %f s, uts: %d\n", instanceID, ret.Duration, time.Now().UnixMilli())
}

func restoreKVStore(ctx context.Context,
	args *StreamTaskArgs,
) error {
	substreamNum := args.ectx.SubstreamNum()
	if env_config.PARALLEL_RESTORE {
		g, ectx := errgroup.WithContext(ctx)
		for _, kvchangelog := range args.kvChangelogs {
			kvc := kvchangelog
			g.Go(func() error {
				return restoreOneKVStore(ectx, kvc, substreamNum)
			})
		}
		return g.Wait()
	} else {
		for _, kvchangelog := range args.kvChangelogs {
			err := restoreOneKVStore(ctx, kvchangelog, substreamNum)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func restoreOneKVStore(ctx context.Context, kvchangelog store.KeyValueStoreOpWithChangelog,
	substreamNum uint8,
) error {
	err := store_restore.RestoreChangelogKVStateStore(ctx, kvchangelog,
		substreamNum)
	if err != nil {
		return fmt.Errorf("RestoreKVStateStore failed: %v", err)
	}
	return nil
}

func restoreChangelogBackedWindowStore(ctx context.Context,
	args *StreamTaskArgs,
) error {
	substreamNum := args.ectx.SubstreamNum()
	if env_config.PARALLEL_RESTORE {
		g, ectx := errgroup.WithContext(ctx)
		for _, wschangelog := range args.windowStoreChangelogs {
			wsc := wschangelog
			g.Go(func() error {
				return store_restore.RestoreChangelogWindowStateStore(ectx, wsc, substreamNum)
			})
		}
		return g.Wait()
	} else {
		for _, wschangelog := range args.windowStoreChangelogs {
			err := store_restore.RestoreChangelogWindowStateStore(ctx, wschangelog, substreamNum)
			if err != nil {
				return fmt.Errorf("RestoreWindowStateStore failed: %v", err)
			}
		}
		return nil
	}
}

func restoreStateStore(ctx context.Context, args *StreamTaskArgs) error {
	if args.kvChangelogs != nil {
		err := restoreKVStore(ctx, args)
		if err != nil {
			return err
		}
	}
	if args.windowStoreChangelogs != nil {
		err := restoreChangelogBackedWindowStore(ctx, args)
		if err != nil {
			return err
		}
	}
	return nil
}

func configChangelogExactlyOnce(
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	args *StreamTaskArgs,
	trackStream func(name string, stream *sharedlog_stream.ShardedSharedLogStream),
) {
	for _, kvchangelog := range args.kvChangelogs {
		kvchangelog.ConfigureExactlyOnce(rem, args.guarantee)
		trackStream(kvchangelog.ChangelogTopicName(), kvchangelog.Stream().(*sharedlog_stream.ShardedSharedLogStream))
	}
	for _, wschangelog := range args.windowStoreChangelogs {
		wschangelog.ConfigureExactlyOnce(rem, args.guarantee)
		trackStream(wschangelog.ChangelogTopicName(), wschangelog.Stream().(*sharedlog_stream.ShardedSharedLogStream))
	}
}

func updateFuncs(streamTaskArgs *StreamTaskArgs,
	trackParFunc exactly_once_intr.TrackProdSubStreamFunc,
	recordFinish exactly_once_intr.RecordPrevInstanceFinishFunc,
	flushCallbackFunc exactly_once_intr.FlushCallbackFunc,
) {
	streamTaskArgs.ectx.SetTrackParFunc(trackParFunc)
	streamTaskArgs.ectx.SetRecordFinishFunc(recordFinish)
	for _, kvchangelog := range streamTaskArgs.kvChangelogs {
		kvchangelog.SetTrackParFunc(trackParFunc)
		kvchangelog.SetStreamFlushCallbackFunc(flushCallbackFunc)
	}
	for _, wschangelog := range streamTaskArgs.windowStoreChangelogs {
		wschangelog.SetTrackParFunc(trackParFunc)
		wschangelog.SetStreamFlushCallbackFunc(flushCallbackFunc)
	}
}

func createOffsetTopicTrackSource(tm *transaction.TransactionManager, args *StreamTaskArgs,
) error {
	for _, src := range args.ectx.Consumers() {
		inputTopicName := src.TopicName()
		err := tm.CreateOffsetTopic(inputTopicName, uint8(src.Stream().NumPartition()), args.bufMaxSize)
		if err != nil {
			return err
		}
		if !src.IsInitialSource() {
			src.ConfigExactlyOnce(args.guarantee)
		}
		tm.RecordTopicStreams(inputTopicName, src.Stream().(*sharedlog_stream.ShardedSharedLogStream))
	}
	return nil
}

func getOffsetMap(ctx context.Context,
	tm *transaction.TransactionManager, args *StreamTaskArgs,
) (map[string]uint64, error) {
	offsetMap := make(map[string]uint64, len(args.ectx.Consumers()))
	for _, src := range args.ectx.Consumers() {
		inputTopicName := src.TopicName()
		offset, err := transaction.GetOffset(ctx, tm, inputTopicName,
			args.ectx.SubstreamNum())
		if err != nil {
			return nil, fmt.Errorf("createOffsetTopicAndGetOffset failed: %v", err)
		}
		offsetMap[inputTopicName] = offset
		debug.Fprintf(os.Stderr, "tp %s offset %d\n", inputTopicName, offset)
	}
	return offsetMap, nil
}

func setOffsetOnStream(offsetMap map[string]uint64,
	args *StreamTaskArgs,
) {
	if offsetMap != nil {
		for _, src := range args.ectx.Consumers() {
			inputTopicName := src.TopicName()
			offset := offsetMap[inputTopicName]
			resetTo := offset + 1
			if offset == 0 {
				resetTo = offset
			}
			debug.Fprintf(os.Stderr, "%s offset restores to %x\n", src.TopicName(), resetTo)
			src.SetCursor(resetTo, args.ectx.SubstreamNum())
		}
	}
}

func findOffsetFromOffsetPairs(topicName string, offsetPairs []*remote_txn_rpc.OffsetPair) (uint64, error) {
	for _, p := range offsetPairs {
		if p.TopicName == topicName {
			return p.Offset, nil
		}
	}
	return 0, common_errors.ErrTopicNotFound
}

func updateStreamCursor(offsetPairs []*remote_txn_rpc.OffsetPair, args *StreamTaskArgs) error {
	if len(offsetPairs) != 0 {
		for _, src := range args.ectx.Consumers() {
			offset, err := findOffsetFromOffsetPairs(src.TopicName(), offsetPairs)
			if err != nil {
				return err
			}
			resetTo := offset + 1
			if offset == 0 {
				resetTo = offset
			}
			debug.Fprintf(os.Stderr, "%s offset restores to %x\n", src.TopicName(), resetTo)
			src.SetCursor(resetTo, args.ectx.SubstreamNum())
		}
	}
	return nil
}

func checkStreamArgs(args *StreamTaskArgs) {
	debug.Assert(len(args.ectx.Consumers()) >= 1, "Srcs should be filled")
	debug.Assert(args.ectx != nil, "program args should be filled")
}

func prodConsumerExactlyOnce(args *StreamTaskArgs,
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
) {
	checkStreamArgs(args)
	for _, src := range args.ectx.Consumers() {
		if !src.IsInitialSource() || args.guarantee == exactly_once_intr.ALIGN_CHKPT {
			src.ConfigExactlyOnce(args.guarantee)
		}
	}
	for _, sink := range args.ectx.Producers() {
		sink.ConfigExactlyOnce(rem, args.guarantee)
	}
}

// func trackStreamAndConfigureExactlyOnce(args *StreamTaskArgs,
// 	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
// 	trackStream func(name string, stream *sharedlog_stream.ShardedSharedLogStream),
// ) {
// 	for _, src := range args.ectx.Consumers() {
// 		if !src.IsInitialSource() {
// 			src.ConfigExactlyOnce(args.guarantee)
// 		}
// 		trackStream(src.TopicName(),
// 			src.Stream().(*sharedlog_stream.ShardedSharedLogStream))
// 	}
// 	for _, sink := range args.ectx.Producers() {
// 		sink.ConfigExactlyOnce(rem, args.guarantee)
// 		trackStream(sink.TopicName(), sink.Stream().(*sharedlog_stream.ShardedSharedLogStream))
// 	}
// }

func resumeAndInit(t *StreamTask, args *StreamTaskArgs, init *bool, paused *bool) {
	if *paused && t.resumeFunc != nil {
		t.resumeFunc(t, args.guarantee)
		*paused = false
	}
	if !*init {
		args.ectx.StartWarmup()
		if t.initFunc != nil {
			t.initFunc(t)
		}
		*init = true
	}
}

func initAfterMarkOrCommit(t *StreamTask, args *StreamTaskArgs,
	tracker exactly_once_intr.TopicSubstreamTracker, init *bool, paused *bool,
) error {
	if args.fixedOutParNum != -1 {
		sinks := args.ectx.Producers()
		debug.Assert(len(sinks) == 1, "fixed out param is only usable when there's only one output stream")
		// debug.Fprintf(os.Stderr, "%s tracking substream %d\n", sinks[0].TopicName(), args.fixedOutParNum)
		tracker.AddTopicSubstream(sinks[0].TopicName(), uint8(args.fixedOutParNum))
	}
	resumeAndInit(t, args, init, paused)
	return nil
}

func forwardCtrlMsg(
	ctx context.Context,
	msg *commtypes.RawMsgAndSeq,
	args *StreamTaskArgs,
	name string,
) error {
	for _, sink := range args.ectx.Producers() {
		if args.fixedOutParNum >= 0 {
			_, err := sink.ProduceCtrlMsg(ctx, msg, []uint8{args.ectx.SubstreamNum()})
			if err != nil {
				return err
			}
			if msg.Mark == commtypes.STREAM_END {
				fmt.Fprintf(os.Stderr, "%d forward %s to %s(%d)\n",
					args.ectx.SubstreamNum(), name, sink.TopicName(), args.fixedOutParNum)
			}
		} else {
			parNums := make([]uint8, 0, sink.Stream().NumPartition())
			for par := uint8(0); par < sink.Stream().NumPartition(); par++ {
				parNums = append(parNums, par)
			}
			_, err := sink.ProduceCtrlMsg(ctx, msg, parNums)
			if err != nil {
				return err
			}
			if msg.Mark == commtypes.STREAM_END {
				fmt.Fprintf(os.Stderr, "%d forward %s to %s(%v)\n",
					args.ectx.SubstreamNum(), name, sink.TopicName(), parNums)
			}
		}
	}
	return nil
}

func handleScaleEpochAndBytes(ctx context.Context, msg *commtypes.RawMsgAndSeq,
	args *StreamTaskArgs,
) *common.FnOutput {
	epochMarker := commtypes.EpochMarker{
		ScaleEpoch: msg.ScaleEpoch,
		Mark:       commtypes.SCALE_FENCE,
		ProdIndex:  args.ectx.SubstreamNum(),
	}
	encoded, b, err := args.epochMarkerSerde.Encode(epochMarker)
	defer func() {
		if args.epochMarkerSerde.UsedBufferPool() && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	msg.Payload = encoded
	err = forwardCtrlMsg(ctx, msg, args, "scale fence")
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	err = args.ectx.RecordFinishFunc()(ctx, args.ectx.FuncName(), args.ectx.SubstreamNum())
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return &common.FnOutput{
		Success: true,
		Message: fmt.Sprintf("%s-%d epoch %d exit",
			args.ectx.FuncName(), args.ectx.SubstreamNum(), args.ectx.CurEpoch()),
	}
}

func loadSnapshot(ctx context.Context,
	args *StreamTaskArgs, auxData []byte, auxMetaSeq uint64, rs *snapshot_store.RedisSnapshotStore,
) error {
	if len(auxData) > 0 {
		uint16Serde := commtypes.Uint16SerdeG{}
		ret, err := uint16Serde.Decode(auxData)
		if err != nil {
			return fmt.Errorf("[ERR] Decode: %v", err)
		}
		if ret == 1 {
			payloadSerde, err := commtypes.GetPayloadArrSerdeG(args.serdeFormat)
			if err != nil {
				return err
			}
			for kvchTp, kvchangelog := range args.kvChangelogs {
				tp := fmt.Sprintf("%s-%d", kvchTp, args.ectx.SubstreamNum())
				snapArr, err := rs.GetSnapshot(ctx, tp, auxMetaSeq, kvchangelog.GetInstanceId())
				if err != nil {
					return fmt.Errorf("[ERR] RedisGetSnapshot: tp=%s, seq=%#x, err=%v", kvchTp, auxMetaSeq, err)
				}
				if len(snapArr) > 0 {
					payloadArr, err := payloadSerde.Decode(snapArr)
					if err != nil {
						return fmt.Errorf("[ERR] Fail to decode snapshot: %v", err)
					}
					err = kvchangelog.RestoreFromSnapshot(payloadArr.Payloads)
					if err != nil {
						return fmt.Errorf("[ERR] restore kv from snapshot: %v", err)
					}
					kvchangelog.Stream().SetCursor(auxMetaSeq+1, kvchangelog.SubstreamNum())
				}
			}
			for wscTp, wsc := range args.windowStoreChangelogs {
				tp := fmt.Sprintf("%s-%d", wscTp, args.ectx.SubstreamNum())
				snapArr, err := rs.GetSnapshot(ctx, tp, auxMetaSeq, wsc.GetInstanceId())
				if err != nil {
					return fmt.Errorf("[ERR] RedisGetSnapshot: tp=%s, seq=%#x, err=%v", wscTp, auxMetaSeq, err)
				}
				if snapArr != nil {
					payloadArr, err := payloadSerde.Decode(snapArr)
					if err != nil {
						return fmt.Errorf("[ERR] Fail to decode snapshot: %v", err)
					}
					err = wsc.RestoreFromSnapshot(ctx, payloadArr.Payloads)
					if err != nil {
						return fmt.Errorf("[ERR] restore window table from snapshot: %v", err)
					}
					wsc.Stream().SetCursor(auxMetaSeq+1, wsc.SubstreamNum())
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "no snapshot found\n")
	}
	return nil
}
