package handlers

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/stream_task"

	"cs.utexas.edu/zjia/faas/types"
)

type fanoutHandler struct {
	env      types.Environment
	funcName string
	msgSerde commtypes.MessageGSerdeG[string, *ntypes.Event]
}

func NewFanout(env types.Environment, funcName string) types.FuncHandler {
	return &fanoutHandler{
		env:      env,
		funcName: funcName,
	}
}

func (h *fanoutHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.fanout(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *fanoutHandler) setupSerde(sf uint8) *common.FnOutput {
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

func (h *fanoutHandler) fanout(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
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
	sinks[0].SetName("fanoutSink")
	ectx := processor.NewExecutionContextFromComponents(proc_interface.NewBaseSrcsSinks(srcs, sinks),
		proc_interface.NewBaseProcArgs(h.funcName, sp.ScaleEpoch, sp.ParNum))
	outProc := processor.NewRoundRobinOutputProcessorG("rbProc", &ectx, sinks[0], h.msgSerde)
	task := stream_task.NewStreamTaskBuilder().MarkFinalStage().
		AppProcessFunc(stream_task.CommonAppProcessFunc[string, *ntypes.Event](outProc.Process, h.msgSerde)).
		Build()
	streamTaskArgs, err := streamArgsBuilder(&ectx, sp).
		Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback, func() {
		outProc.OutputRemainingStats()
	})
}
