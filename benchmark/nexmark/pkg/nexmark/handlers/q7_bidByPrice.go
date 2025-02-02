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

type q7BidByPrice struct {
	env         types.Environment
	funcName    string
	inMsgSerde  commtypes.MessageGSerdeG[string, *ntypes.Event]
	outMsgSerde commtypes.MessageGSerdeG[uint64, *ntypes.Event]
}

func NewQ7BidByPriceHandler(env types.Environment, funcName string) types.FuncHandler {
	return &q7BidByPrice{
		env:      env,
		funcName: funcName,
	}
}

func (h *q7BidByPrice) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.q7BidByPrice(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *q7BidByPrice) setupSerde(serdeFormat commtypes.SerdeFormat) *common.FnOutput {
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

func (h *q7BidByPrice) q7BidByPrice(ctx context.Context, input *common.QueryInput) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(input.SerdeFormat)
	fn_out := h.setupSerde(serdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks_arr, err := getSrcSinkUint64Key(ctx, input)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	srcs[0].SetInitialSource(true)
	ectx := processor.NewExecutionContext(srcs, sinks_arr, h.funcName,
		input.ScaleEpoch, input.ParNum)
	filterProc := processor.NewStreamFilterProcessorG[string, *ntypes.Event]("filterBids",
		processor.PredicateFuncG[string, *ntypes.Event](
			func(key optional.Option[string], value optional.Option[*ntypes.Event]) (bool, error) {
				event := value.Unwrap()
				return event.Etype == ntypes.BID, nil
			}))
	selectKey := processor.NewStreamSelectKeyProcessorG[string, *ntypes.Event, uint64]("bidKeyedByPrice",
		processor.SelectKeyFuncG[string, *ntypes.Event, uint64](
			func(key optional.Option[string], value optional.Option[*ntypes.Event]) (optional.Option[uint64], error) {
				event := value.Unwrap()
				return optional.Some(event.Bid.Price), nil
			}))
	outProc := processor.NewGroupByOutputProcessorG("bidByPriceProc", sinks_arr[0], &ectx, h.outMsgSerde)
	filterProc.NextProcessor(selectKey)
	selectKey.NextProcessor(outProc)
	task := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(stream_task.CommonAppProcessFunc(filterProc.Process, h.inMsgSerde)).
		Build()
	transactionalID := fmt.Sprintf("%s-%s-%d-%s", h.funcName, input.InputTopicNames[0],
		input.ParNum, input.OutputTopicNames[0])
	streamTaskArgs, err := benchutil.UpdateStreamTaskArgs(input,
		stream_task.NewStreamTaskArgsBuilder(&ectx, transactionalID)).Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback,
		func() { outProc.OutputRemainingStats() })
}
