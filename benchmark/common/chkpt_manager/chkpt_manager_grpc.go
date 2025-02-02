package chkpt_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/checkpt"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/txn_data"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/emptypb"

	"cs.utexas.edu/zjia/faas/types"
)

type ChkptManagerHandlerGrpc struct {
	env              types.Environment
	client           checkpt.ChkptMngrClient
	rcm              checkpt.RedisChkptManager
	epochMarkerSerde commtypes.SerdeG[commtypes.EpochMarker]
	srcStream        *sharedlog_stream.ShardedSharedLogStream
}

func NewChkptManagerHandlerGrpc(env types.Environment) types.FuncHandler {
	return &ChkptManagerHandlerGrpc{
		env: env,
	}
}

func (h *ChkptManagerHandlerGrpc) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.ChkptMngrInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}
	conn, err := checkpt.PrepareChkptClientGrpc()
	if err != nil {
		log.Fatal().Err(err)
	}
	h.client = checkpt.NewChkptMngrClient(conn)
	_, err = h.client.ResetCheckpointCount(ctx, &checkpt.FinMsg{TopicNames: parsedInput.FinalOutputTopicNames})
	if err != nil {
		log.Fatal().Err(err)
	}
	h.rcm = checkpt.NewRedisChkptManager()
	PrintChkptMngrInput(parsedInput)
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Chkpt(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *ChkptManagerHandlerGrpc) genChkpt(ctx context.Context, input *common.ChkptMngrInput) error {
	tpNameHash := h.srcStream.TopicNameHash()
	useBuf := h.epochMarkerSerde.UsedBufferPool()
	srcLogOff := make([]commtypes.TpLogOff, 0, input.SrcNumPart)
	for i := uint8(0); i < input.SrcNumPart; i++ {
		chkpt_tag := txn_data.ChkptTag(tpNameHash, i)
		nameHashTag := sharedlog_stream.NameHashWithPartition(tpNameHash, i)
		marker := commtypes.EpochMarker{
			Mark:      commtypes.CHKPT_MARK,
			ProdIndex: i,
		}
		encoded, b, err := h.epochMarkerSerde.Encode(marker)
		if err != nil {
			return err
		}
		logOff, err := h.srcStream.PushWithTag(ctx, encoded, i, []uint64{nameHashTag, chkpt_tag}, nil,
			sharedlog_stream.ControlRecordMeta, commtypes.EmptyProducerId)
		if err != nil {
			return err
		}
		if useBuf && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
		srcLogOff = append(srcLogOff, commtypes.TpLogOff{
			Tp:     fmt.Sprintf("%s-%d", h.srcStream.TopicName(), i),
			LogOff: logOff,
		})
	}
	return h.rcm.StoreInitSrcLogoff(ctx, srcLogOff, input.SrcNumPart)
}

func (h *ChkptManagerHandlerGrpc) WaitForChkptFinish(ctx context.Context, input *common.ChkptMngrInput) (bool, error) {
	for {
		r, err := h.client.CheckChkptFinish(ctx, &checkpt.CheckFinMsg{
			TopicNames: input.FinalOutputTopicNames,
			Pars:       input.FinalNumOutPartitions,
		})
		if err != nil {
			return false, err
		}
		if r.Fin {
			return false, nil
		}
		ended, err := h.client.ChkptMngrEnded(ctx, &emptypb.Empty{})
		if err != nil {
			return false, err
		}
		if ended.Ended {
			return true, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (h *ChkptManagerHandlerGrpc) Chkpt(ctx context.Context, input *common.ChkptMngrInput) *common.FnOutput {
	var err error
	guarantee := exactly_once_intr.GuaranteeMth(input.GuaranteeMth)

	if guarantee == exactly_once_intr.ALIGN_CHKPT {
		serdeFormat := commtypes.SerdeFormat(input.SerdeFormat)
		chkptEveryMs := time.Duration(input.ChkptEveryMs) * time.Millisecond
		chkptTimes := stats.NewStatsCollector[int64]("e2eChkptElapsed(ms)", stats.DEFAULT_COLLECT_DURATION)
		h.epochMarkerSerde, err = commtypes.GetEpochMarkerSerdeG(serdeFormat)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		h.srcStream, err = sharedlog_stream.NewShardedSharedLogStream(
			input.SrcTopicName,
			input.SrcNumPart,
			serdeFormat, input.BufMaxSize)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		debug.Fprintf(os.Stderr, "waiting for the first checkpt done\n")
		should_exit, err := h.WaitForChkptFinish(ctx, input)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		if should_exit {
			return &common.FnOutput{Success: true}
		}
		debug.Fprintf(os.Stderr, "first checkpt is done\n")
		now := time.Now()
		for {
			ended, err := h.client.ChkptMngrEnded(ctx, &emptypb.Empty{})
			if err != nil {
				return common.GenErrFnOutput(err)
			}
			if ended.Ended {
				break
			}
			elapsed := time.Since(now)
			if elapsed < chkptEveryMs {
				diff := chkptEveryMs - elapsed
				// debug.Fprintf(os.Stderr, "about to sleep for %v\n", diff)
				time.Sleep(diff)
			}
			// debug.Fprintf(os.Stderr, "after sleep\n")
			_, err = h.client.ResetCheckpointCount(ctx, &checkpt.FinMsg{TopicNames: input.FinalOutputTopicNames})
			if err != nil {
				debug.Fprintf(os.Stderr, "reset chkpt count err: %v\n", err)
				return common.GenErrFnOutput(err)
			}
			// debug.Fprintf(os.Stderr, "after reset chkpt count\n")
			startChkpt := time.Now()
			err = h.genChkpt(ctx, input)
			if err != nil {
				debug.Fprintf(os.Stderr, "gen chkpt err: %v\n", err)
				return common.GenErrFnOutput(err)
			}
			// debug.Fprintf(os.Stderr, "after genChkpt\n")
			should_exit, err := h.WaitForChkptFinish(ctx, input)
			if err != nil {
				debug.Fprintf(os.Stderr, "wait for chkpt finish err: %v\n", err)
				return common.GenErrFnOutput(err)
			}
			el := time.Since(startChkpt)
			chkptTimes.AddSample(el.Milliseconds())
			if should_exit {
				break
			}
			// debug.Fprintf(os.Stderr, "after wait for chkpt finish\n")
			now = time.Now()
		}
		chkptTimes.PrintRemainingStats()
	}
	debug.Fprintf(os.Stderr, "chkptmngr exits with success\n")
	return &common.FnOutput{Success: true}
}
