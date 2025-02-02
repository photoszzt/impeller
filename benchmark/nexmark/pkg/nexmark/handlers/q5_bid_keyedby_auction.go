package handlers

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/stream_task"

	"cs.utexas.edu/zjia/faas/types"
)

type bidByAuction struct {
	env         types.Environment
	funcName    string
	inMsgSerde  commtypes.MessageGSerdeG[string, *ntypes.Event]
	outMsgSerde commtypes.MessageGSerdeG[uint64, *ntypes.Event]
}

func NewBidByAuctionHandler(env types.Environment, funcName string) types.FuncHandler {
	return &bidByAuction{
		env:      env,
		funcName: funcName,
	}
}

func (h *bidByAuction) Call(ctx context.Context, input []byte) ([]byte, error) {
	sp := &common.QueryInput{}
	err := json.Unmarshal(input, sp)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.processBidKeyedByAuction(ctx, sp)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return common.CompressData(encodedOutput), nil
}

func (h *bidByAuction) setupSerde(sf uint8) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sf)
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.inMsgSerde, err = commtypes.GetMsgGSerdeG[string](serdeFormat, commtypes.StringSerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *bidByAuction) processBidKeyedByAuction(ctx context.Context,
	sp *common.QueryInput,
) *common.FnOutput {
	fn_out := h.setupSerde(sp.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks, err := getSrcSinkUint64Key(ctx, sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	srcs[0].SetInitialSource(true)
	ectx := processor.NewExecutionContext(srcs,
		sinks, h.funcName, sp.ScaleEpoch, sp.ParNum)
	filterProc := processor.NewStreamFilterProcessorG[string, *ntypes.Event]("filterBid", processor.PredicateFuncG[string, *ntypes.Event](
		func(key optional.Option[string], value optional.Option[*ntypes.Event]) (bool, error) {
			event := value.Unwrap()
			return event.Etype == ntypes.BID, nil
		}))

	mapProc := processor.NewStreamSelectKeyProcessorG[string, *ntypes.Event, uint64]("selectKey",
		processor.SelectKeyFuncG[string, *ntypes.Event, uint64](
			func(key optional.Option[string], value optional.Option[*ntypes.Event]) (optional.Option[uint64], error) {
				event := value.Unwrap()
				return optional.Some(event.Bid.Auction), nil
			}))
	outProc := processor.NewGroupByOutputProcessorG("subG1Proc", sinks[0], &ectx, h.outMsgSerde)
	filterProc.NextProcessor(mapProc)
	mapProc.NextProcessor(outProc)
	task := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(
			stream_task.CommonAppProcessFunc(filterProc.Process, h.inMsgSerde)).
		Build()
	streamTaskArgs, err := streamArgsBuilder(&ectx, sp).Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback, func() { outProc.OutputRemainingStats() })
}
