package handlers

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/stream_task"

	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"

	"cs.utexas.edu/zjia/faas/types"
)

type query2Handler struct {
	env      types.Environment
	funcName string
	msgSerde commtypes.MessageGSerdeG[string, *ntypes.Event]
}

func NewQuery2(env types.Environment, funcName string) types.FuncHandler {
	return &query2Handler{
		env:      env,
		funcName: funcName,
	}
}

func (h *query2Handler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Query2(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func filterFunc(_ optional.Option[string], value optional.Option[*ntypes.Event]) (bool, error) {
	v := value.Unwrap()
	return v.Etype == ntypes.BID && v.Bid.Auction%123 == 0, nil
}

func (h *query2Handler) setupSerde(sf uint8) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sf)
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.msgSerde, err = commtypes.GetMsgGSerdeG[string](serdeFormat, commtypes.StringSerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *query2Handler) Query2(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	fn_out := h.setupSerde(sp.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks, err := getSrcSink(ctx, sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	srcs[0].SetInitialSource(true)
	sinks[0].MarkFinalOutput()
	srcsSinks := proc_interface.NewBaseSrcsSinks(srcs, sinks)
	ectx := processor.NewExecutionContextFromComponents(srcsSinks, proc_interface.NewBaseProcArgs(h.funcName,
		sp.ScaleEpoch, sp.ParNum))
	filterProc := processor.NewStreamFilterProcessorG[string, *ntypes.Event]("q2Filter",
		processor.PredicateFuncG[string, *ntypes.Event](filterFunc))
	outProc := processor.NewFixedSubstreamOutputProcessorG("subG1Proc", sinks[0], sp.ParNum, h.msgSerde)
	filterProc.NextProcessor(outProc)
	task := stream_task.NewStreamTaskBuilder().MarkFinalStage().
		AppProcessFunc(stream_task.CommonAppProcessFunc[string, *ntypes.Event](filterProc.Process, h.msgSerde)).
		Build()
	streamTaskArgs, err := streamArgsBuilder(&ectx, sp).
		FixedOutParNum(sp.ParNum).
		Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback, func() {
		outProc.OutputRemainingStats()
	})
}
