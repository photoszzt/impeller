package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type q3GroupByHandler struct {
	env types.Environment

	funcName    string
	inMsgSerde  commtypes.MessageGSerdeG[string, *ntypes.Event]
	outMsgSerde commtypes.MessageGSerdeG[uint64, *ntypes.Event]
}

func NewQ3GroupByHandler(env types.Environment, funcName string) types.FuncHandler {
	return &q3GroupByHandler{
		env:      env,
		funcName: funcName,
	}
}

func (h *q3GroupByHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Q3GroupBy(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func getExecutionCtx(ctx context.Context, sp *common.QueryInput, funcName string,
) (processor.BaseExecutionContext, error) {
	input_stream, output_streams, err := benchutil.GetShardedInputOutputStreams(ctx, sp)
	if err != nil {
		return processor.BaseExecutionContext{}, err
	}
	debug.Assert(len(output_streams) == 2, "expected 2 output streams")
	var sinks []producer_consumer.MeteredProducerIntr
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	inConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     common.SrcConsumeTimeout,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		Format:        serdeFormat,
	}
	// fmt.Fprintf(os.Stderr, "output to %v\n", output_stream.TopicName())
	warmup := time.Duration(sp.WarmupS) * time.Second
	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(input_stream, inConfig,
		sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return processor.BaseExecutionContext{}, fmt.Errorf("get msg serde err: %v", err)
	}
	src := producer_consumer.NewMeteredConsumer(consumer, warmup)
	for _, output_stream := range output_streams {
		sink, err := producer_consumer.NewMeteredProducer(producer_consumer.NewShardedSharedLogStreamProducer(output_stream, outConfig),
			warmup)
		if err != nil {
			return processor.BaseExecutionContext{}, err
		}
		sinks = append(sinks, sink)
	}
	return processor.NewExecutionContext([]*producer_consumer.MeteredConsumer{src}, sinks,
		funcName, sp.ScaleEpoch, sp.ParNum), nil
}

func (h *q3GroupByHandler) setupSerde(sf uint8) *common.FnOutput {
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

func (h *q3GroupByHandler) Q3GroupBy(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	fn_out := h.setupSerde(sp.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	ectx, err := getExecutionCtx(ctx, sp, h.funcName)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	ectx.Consumers()[0].SetInitialSource(true)
	ectx.Producers()[0].SetName("aucSink")
	ectx.Producers()[1].SetName("perSink")
	aucBySellerProc := processor.NewStreamSelectKeyProcessorG[string, *ntypes.Event, uint64](
		"auctionsBySellerIDMap", processor.SelectKeyFuncG[string, *ntypes.Event, uint64](
			func(_ optional.Option[string], value optional.Option[*ntypes.Event]) (optional.Option[uint64], error) {
				return optional.Some(value.Unwrap().NewAuction.Seller), nil
			}))
	grouBySellerProc := processor.NewGroupByOutputProcessorG("aucProc", ectx.Producers()[0], &ectx, h.outMsgSerde)
	aucBySellerProc.NextProcessor(grouBySellerProc)
	perByIDProc := processor.NewStreamSelectKeyProcessorG[string, *ntypes.Event, uint64](
		"personsByIDMap", processor.SelectKeyFuncG[string, *ntypes.Event, uint64](
			func(_ optional.Option[string], value optional.Option[*ntypes.Event]) (optional.Option[uint64], error) {
				return optional.Some(value.Unwrap().NewPerson.ID), nil
			}))
	groupByPerIDProc := processor.NewGroupByOutputProcessorG("perProc", ectx.Producers()[1], &ectx, h.outMsgSerde)
	perByIDProc.NextProcessor(groupByPerIDProc)

	task := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(stream_task.CommonAppProcessFunc(
			func(ctx context.Context, msg commtypes.MessageG[string, *ntypes.Event]) error {
				event := msg.Value.Unwrap()
				if event.Etype == ntypes.PERSON && ((event.NewPerson.State == "OR") ||
					event.NewPerson.State == "ID" || event.NewPerson.State == "CA") {
					err = perByIDProc.Process(ctx, msg)
					if err != nil {
						return err
					}
				} else if event.Etype == ntypes.AUCTION && event.NewAuction.Category == 10 {
					err = aucBySellerProc.Process(ctx, msg)
					if err != nil {
						return err
					}
				}
				return nil
			}, h.inMsgSerde)).
		Build()
	streamTaskArgs, err := benchutil.UpdateStreamTaskArgs(sp,
		stream_task.NewStreamTaskArgsBuilder(&ectx,
			fmt.Sprintf("%s-%s-%d", h.funcName, sp.InputTopicNames[0], sp.ParNum))).Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, stream_task.EmptySetupSnapshotCallback, func() {
		grouBySellerProc.OutputRemainingStats()
		groupByPerIDProc.OutputRemainingStats()
	})
}
