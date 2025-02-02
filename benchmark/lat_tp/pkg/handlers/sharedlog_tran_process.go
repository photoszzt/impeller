package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedlog-stream/benchmark/common"
	datatype "sharedlog-stream/benchmark/lat_tp/pkg/data_type"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type sharedlogTranProcessHandler struct {
	env      types.Environment
	msgSerde commtypes.MessageGSerdeG[string, datatype.PayloadTs]
}

func NewSharedlogTranProcessHandler(env types.Environment) types.FuncHandler {
	return &sharedlogTranProcessHandler{
		env: env,
	}
}

func (h *sharedlogTranProcessHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.TranProcessBenchParam{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	output := h.sharedlogTranProcess(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *sharedlogTranProcessHandler) getSrcSink(sp *common.TranProcessBenchParam,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, *common.FnOutput) {
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	inStream, err := sharedlog_stream.NewShardedSharedLogStream(sp.InTopicName,
		sp.NumPartition, serdeFormat, sp.BufMaxSize)
	if err != nil {
		return nil, nil, common.GenErrFnOutput(err)
	}
	outStream, err := sharedlog_stream.NewShardedSharedLogStream(sp.OutTopicName,
		sp.NumPartition, serdeFormat, sp.BufMaxSize)
	if err != nil {
		return nil, nil, common.GenErrFnOutput(err)
	}
	inConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     common.SrcConsumeTimeout,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(100) * time.Millisecond,
		Format:        serdeFormat,
	}
	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(inStream, inConfig, 1, 0)
	if err != nil {
		return nil, nil, common.GenErrFnOutput(err)
	}
	src := producer_consumer.NewMeteredConsumer(consumer, 0)
	sink, err := producer_consumer.NewMeteredProducer(producer_consumer.NewShardedSharedLogStreamProducer(outStream, outConfig), 0)
	if err != nil {
		return nil, nil, common.GenErrFnOutput(err)
	}
	return []*producer_consumer.MeteredConsumer{src}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *sharedlogTranProcessHandler) getMsgSerde(sf uint8) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sf)
	var payloadTsSerde commtypes.SerdeG[datatype.PayloadTs]
	var err error
	if serdeFormat == commtypes.MSGP {
		payloadTsSerde = datatype.PayloadTsMsgpSerdeG{}
	} else {
		payloadTsSerde = datatype.PayloadTsJSONSerdeG{}
	}
	h.msgSerde, err = commtypes.GetMsgGSerdeG[string](serdeFormat, commtypes.StringSerdeG{}, payloadTsSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *sharedlogTranProcessHandler) sharedlogTranProcess(ctx context.Context, sp *common.TranProcessBenchParam) *common.FnOutput {
	srcs, sinks, fn_out := h.getSrcSink(sp)
	if fn_out != nil {
		return fn_out
	}
	srcs[0].SetInitialSource(true)
	sinks[0].MarkFinalOutput()
	fn_out = h.getMsgSerde(sp.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	ectx := processor.NewExecutionContextFromComponents(proc_interface.NewBaseSrcsSinks(srcs, sinks),
		proc_interface.NewBaseProcArgs("tranProcess", 1, 0))
	outProc := processor.NewFixedSubstreamOutputProcessorG("subG1", sinks[0], 0, h.msgSerde)
	task := stream_task.NewStreamTaskBuilder().MarkFinalStage().
		AppProcessFunc(stream_task.CommonAppProcessFunc(outProc.Process, h.msgSerde)).
		Build()
	streamTaskArgs, err := stream_task.NewStreamTaskArgsBuilder(&ectx,
		fmt.Sprintf("tranProcess-%s-%d-%s", sp.InTopicName,
			0, sp.OutTopicName)).
		Guarantee(exactly_once_intr.EPOCH_MARK).
		AppID(sp.AppId).
		Warmup(time.Duration(0) * time.Second).
		CommitEveryMs(sp.CommitEveryMs).
		FlushEveryMs(sp.FlushMs).
		Duration(sp.Duration).
		SerdeFormat(commtypes.SerdeFormat(sp.SerdeFormat)).
		BufMaxSize(sp.BufMaxSize).
		FaasGateway(sp.FaasGateway).
		WaitEndMark(false).FixedOutParNum(0).Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback,
		func() {
			outProc.OutputRemainingStats()
		})
}
