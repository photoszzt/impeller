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
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/transaction"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetRemoteTxnMngrAddr() []string {
	raw_addr := os.Getenv("RTXN_MNGR_ADDR")
	return strings.Split(raw_addr, ",")
}

func prepareInit(rtm_client exactly_once_intr.ReadOnlyExactlyOnceManager, args *StreamTaskArgs) *remote_txn_rpc.InitArg {
	arg := remote_txn_rpc.InitArg{
		TransactionalId: args.transactionalId,
		SubstreamNum:    uint32(args.ectx.SubstreamNum()),
		BufMaxSize:      args.bufMaxSize,
	}
	for _, c := range args.ectx.Consumers() {
		arg.InputStreamInfos = append(arg.InputStreamInfos, &remote_txn_rpc.StreamInfo{
			TopicName:    c.TopicName(),
			NumPartition: uint32(c.Stream().NumPartition()),
		})
		if !c.IsInitialSource() {
			c.ConfigExactlyOnce(args.guarantee)
		}
	}
	for _, c := range args.ectx.Producers() {
		arg.OutputStreamInfos = append(arg.OutputStreamInfos, &remote_txn_rpc.StreamInfo{
			TopicName:    c.TopicName(),
			NumPartition: uint32(c.Stream().NumPartition()),
		})
		c.ConfigExactlyOnce(rtm_client, args.guarantee)
	}
	for _, kvc := range args.kvChangelogs {
		arg.KVChangelogInfos = append(arg.KVChangelogInfos, &remote_txn_rpc.StreamInfo{
			TopicName:    kvc.ChangelogTopicName(),
			NumPartition: uint32(kvc.Stream().NumPartition()),
		})
		kvc.ConfigureExactlyOnce(rtm_client, args.guarantee)
	}
	for _, wsc := range args.windowStoreChangelogs {
		arg.WinChangelogInfos = append(arg.WinChangelogInfos, &remote_txn_rpc.StreamInfo{
			TopicName:    wsc.ChangelogTopicName(),
			NumPartition: uint32(wsc.Stream().NumPartition()),
		})
		wsc.ConfigureExactlyOnce(rtm_client, args.guarantee)
	}
	return &arg
}

func prepareGrpc(instance uint8) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	retryPolicy := `{
		"methodConfig": [{
		  "name": [{"service": "remote_txn_rpc.RemoteTxnMngr"}],
		  "waitForReady": true,
		  "retryPolicy": {
			  "MaxAttempts": 4,
			  "InitialBackoff": ".002s",
			  "MaxBackoff": ".01s",
			  "BackoffMultiplier": 2.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultServiceConfig(retryPolicy))
	rmngr_addr := GetRemoteTxnMngrAddr()
	idx := uint64(instance) % uint64(len(rmngr_addr))
	debug.Fprintf(os.Stderr, "remote txn manager addr: %v, connecting to %v\n",
		rmngr_addr, rmngr_addr[idx])
	return grpc.Dial(rmngr_addr[idx], opts...)
}

func getNodeConstraint(instance uint8) string {
	raw_addr := os.Getenv("RTXN_MNGR_NODES")
	nodes := strings.Split(raw_addr, ",")
	idx := uint64(instance) % uint64(len(nodes))
	debug.Fprintf(os.Stderr, "remote txn manager nodes: %v, choose %v\n",
		nodes, nodes[idx])
	return nodes[idx]
}

func SetupManagerForRemote2pc(ctx context.Context, t *StreamTask, args *StreamTaskArgs, rs *snapshot_store.RedisSnapshotStore,
) (*transaction.RemoteTxnMngrClientGrpc, *control_channel.ControlChannelManager, error) {
	checkStreamArgs(args)
	conn, err := prepareGrpc(args.ectx.SubstreamNum())
	if err != nil {
		return nil, nil, err
	}
	client := transaction.NewRemoteTxnMngrClientGrpc(conn, args.transactionalId)
	// node := getNodeConstraint(args.ectx.SubstreamNum())
	// client := transaction.NewRemoteTxnMngrClientBoki(args.faas_gateway, node, args.serdeFormat, args.transactionalId)
	// debug.Fprintf(os.Stderr, "[%d] remote txn manager client setup done\n",
	// 	args.ectx.SubstreamNum())

	arg := prepareInit(client, args)
	// debug.Fprintf(os.Stderr, "[%d] done prepare init\n", args.ectx.SubstreamNum())
	initReply, err := client.Init(ctx, arg)
	if err != nil {
		return nil, nil, err
	}
	client.UpdateProducerId(initReply.ProdId)
	// prodId := client.GetProducerId()
	// debug.Fprintf(os.Stderr, "[%d] done init transaction manager, got task id: %#x, task epoch: %#x\n",
	// 	args.ectx.SubstreamNum(), prodId.TaskId, prodId.TaskEpoch)
	if len(initReply.OffsetPairs) != 0 {
		restoreBeg := time.Now()
		// if env_config.CREATE_SNAPSHOT {
		// 	loadSnapBeg := time.Now()
		// 	err = loadSnapshot(ctx, args, initRet.RecentTxnAuxData, initRet.RecentTxnLogSeq, rs)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	loadSnapElapsed := time.Since(loadSnapBeg)
		// 	fmt.Fprintf(os.Stderr, "load snapshot took %v\n", loadSnapElapsed)
		// }
		restoreSsBeg := time.Now()
		err = restoreStateStore(ctx, args)
		if err != nil {
			return nil, nil, err
		}
		restoreStateStoreElapsed := time.Since(restoreSsBeg)
		fmt.Fprintf(os.Stderr, "restore state store took %v\n", restoreStateStoreElapsed)
		err = updateStreamCursor(initReply.OffsetPairs, args)
		if err != nil {
			return nil, nil, err
		}
		debug.Fprintf(os.Stderr, "down restore\n")
		restoreElapsed := time.Since(restoreBeg)
		fmt.Fprintf(os.Stderr, "down restore, elapsed: %v\n", restoreElapsed)
	}
	cmm, err := control_channel.NewControlChannelManager(args.appId,
		commtypes.SerdeFormat(args.serdeFormat), args.bufMaxSize,
		args.ectx.CurEpoch(), args.ectx.SubstreamNum())
	if err != nil {
		return nil, nil, err
	}
	err = cmm.RestoreMappingAndWaitForPrevTask(ctx, args.ectx.FuncName(),
		env_config.CREATE_SNAPSHOT, args.serdeFormat,
		args.kvChangelogs, args.windowStoreChangelogs, rs)
	if err != nil {
		return nil, nil, err
	}
	trackParFunc := func(topicName string, substreamId uint8) {
		// debug.Fprintf(os.Stderr, "track tp %s substreamid %d\n", topicName, substreamId)
		client.AddTopicSubstream(topicName, substreamId)
	}
	recordFinish := func(ctx context.Context, funcName string, instanceID uint8) error {
		return cmm.RecordPrevInstanceFinish(ctx, funcName, instanceID, args.ectx.CurEpoch())
	}
	flushCallbackFunc := func(ctx context.Context) error {
		waitPrev := stats.TimerBegin()
		err := client.EnsurePrevTxnFinAndAppendMeta(ctx)
		waitAndStart := stats.Elapsed(waitPrev).Microseconds()
		t.waitPrevTxnInPush.AddSample(waitAndStart)
		return err
	}
	updateFuncs(args,
		trackParFunc, recordFinish, flushCallbackFunc)
	for _, p := range args.ectx.Producers() {
		p.SetFlushCallback(flushCallbackFunc)
	}
	return client, cmm, nil
}

type rtxnProcessMeta struct {
	t          *StreamTask
	rtm_client *transaction.RemoteTxnMngrClientGrpc
	// rtm_client *transaction.RemoteTxnMngrClientBoki
	cmm  *control_channel.ControlChannelManager
	args *StreamTaskArgs
}

func processWithRTxnMngr(
	ctx context.Context,
	meta *rtxnProcessMeta,
) *common.FnOutput {
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
			err_out := commitTxnRemote(ctx, meta, &paused, false, &snapshotTimer)
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
			err_out := commitTxnRemote(ctx, meta, &paused, true, &snapshotTimer)
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
			err := initAfterMarkOrCommit(meta.t, meta.args, meta.rtm_client, &init, &paused)
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
					if err := meta.rtm_client.AbortTransaction(ctx); err != nil {
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
			err_out := commitTxnRemote(ctx, meta, &paused, true, &snapshotTimer)
			if err_out != nil {
				fmt.Fprintf(os.Stderr, "commitTransaction err: %v\n", err_out.Message)
				return err_out
			}
			meta.t.PrintRemainingStats()
			meta.rtm_client.OutputRemainingStats()
			fmt.Fprintf(os.Stderr, "finish final commit\n")
			return handleCtrlMsg(ctx, ctrlRawMsgArr, meta.t, meta.args, &warmupCheck, nil)
		}
	}
}

func commitTxnRemote(ctx context.Context,
	meta *rtxnProcessMeta,
	paused *bool,
	finalCommit bool,
	snapshotTimer *time.Time,
) *common.FnOutput {
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
	// 	meta.rtm_client.AddTopicTrackConsumedSeqs(src.TopicName(), meta.args.ectx.SubstreamNum())
	// }
	err := meta.rtm_client.EnsurePrevTxnFinAndAppendMeta(ctx)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	waitAndStart := stats.Elapsed(waitPrev).Microseconds()

	flushAllStart := stats.TimerBegin()
	f, err := flushStreams(ctx, meta.t, meta.args)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("flushStreams: %v", err))
	}
	flushTime := stats.Elapsed(flushAllStart).Microseconds()

	offsetBeg := stats.TimerBegin()
	err = meta.rtm_client.AppendConsumedSeqNum(ctx, meta.args.ectx.Consumers(), meta.args.ectx.SubstreamNum())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] append offset failed: %v\n", err)
		return common.GenErrFnOutput(fmt.Errorf("append offset failed: %v\n", err))
	}
	sendOffsetElapsed := stats.Elapsed(offsetBeg).Microseconds()

	var logOff uint64
	cBeg := stats.TimerBegin()
	if env_config.ASYNC_SECOND_PHASE && !finalCommit {
		logOff, err = meta.rtm_client.CommitTransactionAsyncComplete(ctx)
		if err != nil {
			log.Error().Err(err).Msg("commitAsyncComplete failed")
			return common.GenErrFnOutput(fmt.Errorf("commitAsyncComplete failed: %v", err))
		}
	} else {
		fmt.Fprintf(os.Stderr, "[%d] final commit\n", meta.args.ectx.SubstreamNum())
		logOff, err = meta.rtm_client.CommitTransaction(ctx)
		if err != nil {
			log.Error().Err(err).Msg("commit failed")
			return common.GenErrFnOutput(fmt.Errorf("commit failed: %v\n", err))
		}
	}
	if env_config.CREATE_SNAPSHOT && meta.args.snapshotEvery != 0 && time.Since(*snapshotTimer) > meta.args.snapshotEvery {
		// auxdata is set to precommit
		createSnapshot(ctx, meta.args, []commtypes.TpLogOff{{LogOff: logOff}})
		*snapshotTimer = time.Now()
	}
	cElapsed := stats.Elapsed(cBeg).Microseconds()
	commitTxnElapsed := stats.Elapsed(commitTxnBeg).Microseconds()
	meta.t.waitPrevTxnInCmt.AddSample(waitAndStart)
	meta.t.txnCounter.Tick(1)

	meta.t.txnCommitTime.AddSample(commitTxnElapsed)
	meta.t.commitTxnAPITime.AddSample(cElapsed)
	meta.t.flushStageTime.AddSample(flushTime)
	meta.t.sendOffsetTime.AddSample(sendOffsetElapsed)
	if f > 0 {
		meta.t.flushAtLeastOne.AddSample(flushTime)
	}
	return nil
}
