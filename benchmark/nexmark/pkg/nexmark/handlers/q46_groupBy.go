package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/stream_task"

	"cs.utexas.edu/zjia/faas/types"
)

type q46GroupByHandler struct {
	env         types.Environment
	funcName    string
	inMsgSerde  commtypes.MessageGSerdeG[string, *ntypes.Event]
	outMsgSerde commtypes.MessageGSerdeG[uint64, *ntypes.Event]
}

func NewQ46GroupByHandler(env types.Environment, funcName string) types.FuncHandler {
	return &q46GroupByHandler{
		env:      env,
		funcName: funcName,
	}
}

func (h *q46GroupByHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Q46GroupBy(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *q46GroupByHandler) setupSerde(sf uint8) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sf)
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("get event serde err: %v", err))
	}
	h.inMsgSerde, err = commtypes.GetMsgGSerdeG[string](serdeFormat, commtypes.StringSerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("get msg serde err: %v", err))
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("get msg serde err: %v", err))
	}
	return nil
}

func (h *q46GroupByHandler) Q46GroupBy(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	fn_out := h.setupSerde(sp.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	ectx, err := getExecutionCtx(ctx, sp, h.funcName)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	ectx.Consumers()[0].SetInitialSource(true)
	ectx.Producers()[0].SetName("aucsByIDSink")
	ectx.Producers()[1].SetName("bidsByAucIDSink")
	aucByIDProc := processor.NewStreamSelectKeyProcessorG[string, *ntypes.Event, uint64]("auctionsByIDMap",
		processor.SelectKeyFuncG[string, *ntypes.Event, uint64](
			func(key optional.Option[string], value optional.Option[*ntypes.Event]) (optional.Option[uint64], error) {
				event := value.Unwrap()
				return optional.Some(event.NewAuction.ID), nil
			}))
	groupByAucIDProc := processor.NewGroupByOutputProcessorG("aucProc", ectx.Producers()[0], &ectx, h.outMsgSerde)
	aucByIDProc.NextProcessor(groupByAucIDProc)

	bidsByAucIDProc := processor.NewStreamSelectKeyProcessorG[string, *ntypes.Event, uint64]("bidsByAuctionIDMap",
		processor.SelectKeyFuncG[string, *ntypes.Event, uint64](func(_ optional.Option[string], value optional.Option[*ntypes.Event]) (optional.Option[uint64], error) {
			event := value.Unwrap()
			return optional.Some(event.Bid.Auction), nil
		}))
	groupBidByAucIDProc := processor.NewGroupByOutputProcessorG("bidProc", ectx.Producers()[1], &ectx, h.outMsgSerde)
	bidsByAucIDProc.NextProcessor(groupBidByAucIDProc)

	task := stream_task.NewStreamTaskBuilder().AppProcessFunc(
		stream_task.CommonAppProcessFunc(
			func(ctx context.Context, msg commtypes.MessageG[string, *ntypes.Event]) error {
				event := msg.Value.Unwrap()
				if event.Etype == ntypes.AUCTION {
					err := aucByIDProc.Process(ctx, msg)
					if err != nil {
						return err
					}
				} else if event.Etype == ntypes.BID {
					err := bidsByAucIDProc.Process(ctx, msg)
					if err != nil {
						return err
					}
				}
				return nil
			}, h.inMsgSerde)).
		Build()
	transactionalID := fmt.Sprintf("%s-%s-%d", h.funcName, sp.InputTopicNames[0], sp.ParNum)
	streamTaskArgs, err := benchutil.UpdateStreamTaskArgs(sp,
		stream_task.NewStreamTaskArgsBuilder(&ectx, transactionalID)).Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback, func() {
		groupByAucIDProc.OutputRemainingStats()
		groupBidByAucIDProc.OutputRemainingStats()
	})
}
