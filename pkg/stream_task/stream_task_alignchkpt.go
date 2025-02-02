package stream_task

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/checkpt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/snapshot_store"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/store"
	"time"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/go-redis/redis/v9"
)

// when restoring the checkpoint, it's reading from
// the sequence number of the checkpoint marker.
// the source stream's record between the checkpoint marker
// and its current tail should be ignored.
//  A_in                A_out/B_in               B_out
// ┌───┬────┐  ┌───┐  ┌────┬────────┐  ┌───┐  ┌───┬────────┐
// │|||│    ├──┤ A ├──┤||||│disgard ├──┤ B ├──┤|||│disgard │
// └───┴────┘  └───┘  └────┴────────┘  └───┘  └───┴────────┘
//  marker              marker                   marker

func SetKVStoreChkpt[K, V any](
	ctx context.Context,
	// mc *snapshot_store.MinioChkptStore,
	mc *snapshot_store.RedisSnapshotStore,
	kvstore store.CoreKeyValueStoreG[K, V],
	chkptSerde commtypes.SerdeG[commtypes.Checkpoint],
) {
	kvstore.SetSnapshotCallback(ctx,
		func(ctx context.Context,
			tpLogoff []commtypes.TpLogOff,
			chkptMeta []commtypes.ChkptMetaData,
			snapshot []*commtypes.KeyValuePair[K, V],
		) error {
			kvPairSerdeG := kvstore.GetKVSerde()
			outBin := make([][]byte, 0, len(snapshot))
			for _, kv := range snapshot {
				bin, _, err := kvPairSerdeG.Encode(kv)
				if err != nil {
					return err
				}
				outBin = append(outBin, bin)
			}
			out, buf, err := chkptSerde.Encode(commtypes.Checkpoint{
				KvArr:     outBin,
				ChkptMeta: chkptMeta,
			})
			if err != nil {
				return err
			}
			// fmt.Fprintf(os.Stderr, "kv snapshot size: %d, store snapshot at %x\n", len(out), tpLogoff[0].LogOff)
			err = mc.StoreAlignChkpt(ctx, out, tpLogoff, kvstore.Name(), kvstore.GetInstanceId())
			if chkptSerde.UsedBufferPool() && buf != nil {
				*buf = out
				commtypes.PushBuffer(buf)
			}
			// if kvPairSerdeG.UsedBufferPool() {
			// 	buf2 := new([]byte)
			// 	for _, p := range outBin {
			// 		*buf2 = p
			// 		commtypes.PushBuffer(buf2)
			// 	}
			// }
			return err
		})
}

func SetWinStoreChkpt[K, V any](
	ctx context.Context,
	// mc *snapshot_store.MinioChkptStore,
	mc *snapshot_store.RedisSnapshotStore,
	winStore store.CoreWindowStoreG[K, V],
	chkptSerde commtypes.SerdeG[commtypes.Checkpoint],
) {
	winStore.SetWinSnapshotCallback(ctx,
		func(ctx context.Context,
			tpLogOff []commtypes.TpLogOff,
			chkptMeta []commtypes.ChkptMetaData,
			snapshot []*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V],
		) error {
			kvPairSerdeG := winStore.GetKVSerde()
			outBin := make([][]byte, 0, len(snapshot))
			for _, kv := range snapshot {
				bin, _, err := kvPairSerdeG.Encode(kv)
				if err != nil {
					return err
				}
				outBin = append(outBin, bin)
			}
			out, buf, err := chkptSerde.Encode(commtypes.Checkpoint{
				KvArr:     outBin,
				ChkptMeta: chkptMeta,
			})
			if err != nil {
				return err
			}
			// fmt.Fprintf(os.Stderr, "win snapshot size: %d, store snapshot at %x\n", len(out), tpLogOff[0].LogOff)
			err = mc.StoreAlignChkpt(ctx, out, tpLogOff, winStore.Name(), winStore.GetInstanceId())
			if chkptSerde.UsedBufferPool() && buf != nil {
				*buf = out
				commtypes.PushBuffer(buf)
			}
			// if kvPairSerdeG.UsedBufferPool() {
			// 	buf2 := new([]byte)
			// 	for _, p := range outBin {
			// 		*buf2 = p
			// 		commtypes.PushBuffer(buf2)
			// 	}
			// }
			return err
		})
}

type ChkptMngr struct {
	prodId commtypes.ProducerId
	rcm    checkpt.RedisChkptManager
}

func NewChkptMngr(ctx context.Context,
	rc []*redis.Client,
) *ChkptMngr {
	ckptm := ChkptMngr{
		prodId: commtypes.NewProducerId(),
	}
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	ckptm.prodId.InitTaskId(env)
	ckptm.prodId.TaskEpoch = 1
	ckptm.rcm = checkpt.NewRedisChkptManagerFromClients(rc)
	return &ckptm
}

func (em *ChkptMngr) GetCurrentEpoch() uint32             { return em.prodId.TaskEpoch }
func (em *ChkptMngr) GetCurrentTaskId() uint64            { return em.prodId.TaskId }
func (em *ChkptMngr) GetProducerId() commtypes.ProducerId { return em.prodId }

func processAlignChkpt(ctx context.Context, t *StreamTask, args *StreamTaskArgs,
	rc []*redis.Client,
	mc *snapshot_store.RedisSnapshotStore,
	// mc *snapshot_store.MinioChkptStore,
) *common.FnOutput {
	init := false
	paused := false
	// gotNoData := 0
	var finalOutTpNames []string
	chkptMngr := NewChkptMngr(ctx, rc)
	prodId := chkptMngr.GetProducerId()
	fmt.Fprintf(os.Stderr, "[%d] prodId: %s, warmup time: %v, flush every: %v, waitEndMark: %v\n",
		args.ectx.SubstreamNum(), prodId.String(), args.warmup, args.flushEvery, args.waitEndMark)
	prodConsumerExactlyOnce(args, chkptMngr)
	warmupCheck := stats.NewWarmupChecker(args.warmup)
	// conn, err := PrepareChkptClientGrpc(args.engine1)
	conn, err := checkpt.PrepareChkptClientGrpc()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	client := checkpt.NewChkptMngrClient(conn)
	warmupCheck.StartWarmup()
	alignChkptTime := stats.NewStatsCollector[int64](fmt.Sprintf("createChkpt_%d(ms)", args.ectx.SubstreamNum()),
		stats.DEFAULT_COLLECT_DURATION)
	for _, tp := range args.ectx.Producers() {
		finalOutTpNames = append(finalOutTpNames, tp.TopicName())
	}
	ticker := time.NewTicker(args.flushEvery)
	defer ticker.Stop()
	for {
		warmupCheck.Check()
		select {
		case <-ticker.C:
			ret_err := timedFlushStreams(ctx, t, args)
			if ret_err != nil {
				return ret_err
			}
		default:
		}
		if !init || paused {
			resumeAndInit(t, args, &init, &paused)
		}
		ret, ctrlRawMsgArr := t.appProcessFunc(ctx, t, args.ectx)
		if ret != nil {
			if ret.Success {
				// gotNoData += 1
				continue
			}
			return ret
		}
		if ctrlRawMsgArr != nil {
			if ret_err := pauseTimedFlushStreams(ctx, t, args); ret_err != nil {
				return ret_err
			}
			paused = true
			debug.Assert(ctrlRawMsgArr[0] != nil, "ctrlRawMsgArr should have at least one element")
			if ctrlRawMsgArr[0].Mark != commtypes.CHKPT_MARK {
				// fmt.Fprintf(os.Stderr, "exit due to ctrlMsg, gotNoDataCount: %v\n", gotNoData)
				fmt.Fprintf(os.Stderr, "exit due to ctrlMsg\n")
				alignChkptTime.PrintRemainingStats()
				return handleCtrlMsg(ctx, ctrlRawMsgArr, t, args, &warmupCheck, mc)
			}
			s := time.Now()
			err := checkpoint(ctx, t, args, ctrlRawMsgArr, mc, client, finalOutTpNames)
			if err != nil {
				return common.GenErrFnOutput(err)
			}
			elapsed := time.Since(s)
			alignChkptTime.AddSample(elapsed.Milliseconds())
		}
	}
}

func checkpoint(
	ctx context.Context,
	t *StreamTask,
	args *StreamTaskArgs,
	ctrlRawMsgArr []*commtypes.RawMsgAndSeq,
	// mc *snapshot_store.MinioChkptStore,
	mc *snapshot_store.RedisSnapshotStore,
	// rcm *checkpt.RedisChkptManager,
	client checkpt.ChkptMngrClient,
	finalOutTpNames []string,
) error {
	l := len(args.ectx.Consumers())
	tpLogOff := make([]commtypes.TpLogOff, 0, l)
	chkptMeta := make([]commtypes.ChkptMetaData, 0, l)
	for idx, c := range args.ectx.Consumers() {
		tlo := commtypes.TpLogOff{
			Tp:     fmt.Sprintf("%s-%d", c.Stream().TopicName(), args.ectx.SubstreamNum()),
			LogOff: ctrlRawMsgArr[idx].FirstChkptMarkSeq,
		}
		tpLogOff = append(tpLogOff, tlo)
		chkptMeta = append(chkptMeta, commtypes.ChkptMetaData{
			Unprocessed:     ctrlRawMsgArr[idx].UnprocessSeq,
			LastChkptMarker: ctrlRawMsgArr[idx].LogSeqNum,
		})
	}
	epochMarker := commtypes.EpochMarker{
		StartTime: ctrlRawMsgArr[0].StartTime,
		Mark:      commtypes.CHKPT_MARK,
		ProdIndex: args.ectx.SubstreamNum(),
	}
	encoded, b, err := args.epochMarkerSerde.Encode(epochMarker)
	defer func() {
		if args.epochMarkerSerde.UsedBufferPool() && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return err
	}
	ctrlRawMsgArr[0].Payload = encoded
	err = forwardCtrlMsg(ctx, ctrlRawMsgArr[0], args, "chkpt mark")
	if err != nil {
		return err
	}
	err = createChkpt(ctx, args, tpLogOff, chkptMeta, mc)
	if err != nil {
		return err
	}
	if t.isFinalStage {
		beg := time.Now()
		_, err = client.FinishChkpt(ctx, &checkpt.FinMsg{TopicNames: finalOutTpNames})
		if err != nil {
			return err
		}
		elapsed := time.Since(beg).Microseconds()
		t.finishChkpt.AddSample(elapsed)
	}
	return nil
}
