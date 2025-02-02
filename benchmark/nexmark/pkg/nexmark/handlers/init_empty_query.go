package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/stream_task"

	"cs.utexas.edu/zjia/faas/types"
)

type initQueryEmpty struct {
	env      types.Environment
	funcName string
}

func NewInitEmptyQuery(env types.Environment, funcName string) types.FuncHandler {
	return &initQueryEmpty{
		env:      env,
		funcName: funcName,
	}
}

func (h *initQueryEmpty) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.queryEmpty(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func emptyFunc(ctx context.Context, sp *common.QueryInput,
	funcName string, env types.Environment, subGraphTag string,
	initial bool, last bool,
) *common.FnOutput {
	srcs, sinks, err := getSrcSink(ctx, sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	msgSerde, err := commtypes.GetMsgGSerdeG[string](serdeFormat, commtypes.StringSerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	if initial {
		srcs[0].SetInitialSource(true)
	}
	if last {
		sinks[0].MarkFinalOutput()
	}
	ectx := processor.NewExecutionContextFromComponents(proc_interface.NewBaseSrcsSinks(srcs, sinks),
		proc_interface.NewBaseProcArgs(funcName, sp.ScaleEpoch, sp.ParNum))
	outProc := processor.NewFixedSubstreamOutputProcessorG(subGraphTag, sinks[0], sp.ParNum, msgSerde)
	task := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(stream_task.CommonAppProcessFunc(outProc.Process, msgSerde)).
		Build()
	streamTaskArgs, err := benchutil.UpdateStreamTaskArgs(sp,
		stream_task.NewStreamTaskArgsBuilder(&ectx,
			fmt.Sprintf("%s-%s-%d-%s", funcName, sp.InputTopicNames[0],
				sp.ParNum, sp.OutputTopicNames[0]))).
		FixedOutParNum(sp.ParNum).
		Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback,
		func() {
			outProc.OutputRemainingStats()
		})
}

func (h *initQueryEmpty) queryEmpty(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	return emptyFunc(ctx, sp, h.funcName, h.env, "subG1", true, false)
}
