package handlers

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/execution"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type q4Avg struct {
	env types.Environment

	funcName      string
	inMsgSerde    commtypes.MessageGSerdeG[uint64, commtypes.ChangeG[uint64]]
	outMsgSerde   commtypes.MessageGSerdeG[uint64, float64]
	storeMsgSerde commtypes.MessageGSerdeG[uint64, commtypes.ValueTimestampG[ntypes.SumAndCount]]
}

func NewQ4Avg(env types.Environment, funcName string) *q4Avg {
	return &q4Avg{
		env:      env,
		funcName: funcName,
	}
}

func (h *q4Avg) getExecutionCtx(ctx context.Context, sp *common.QueryInput,
) (processor.BaseExecutionContext, error) {
	inputStream, outputStreams, err := benchutil.GetShardedInputOutputStreams(ctx, sp)
	if err != nil {
		return processor.BaseExecutionContext{}, err
	}
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	warmup := time.Duration(sp.WarmupS) * time.Second

	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(inputStream,
		&producer_consumer.StreamConsumerConfig{
			Timeout:     common.SrcConsumeTimeout,
			SerdeFormat: serdeFormat,
		}, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return processor.BaseExecutionContext{}, err
	}
	src := producer_consumer.NewMeteredConsumer(consumer, warmup)
	sink, err := producer_consumer.NewMeteredProducer(
		producer_consumer.NewShardedSharedLogStreamProducer(outputStreams[0],
			&producer_consumer.StreamSinkConfig{
				FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
				Format:        serdeFormat,
			}), warmup)
	if err != nil {
		return processor.BaseExecutionContext{}, err
	}
	sink.MarkFinalOutput()
	ectx := processor.NewExecutionContext([]*producer_consumer.MeteredConsumer{src},
		[]producer_consumer.MeteredProducerIntr{sink}, h.funcName, sp.ScaleEpoch, sp.ParNum)
	return ectx, nil
}

func (h *q4Avg) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Q4Avg(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *q4Avg) setupSerde(serdeFormat commtypes.SerdeFormat) *common.FnOutput {
	changeSerde, err := commtypes.GetChangeGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{})
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.inMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, changeSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG[uint64, float64](serdeFormat, commtypes.Uint64SerdeG{}, commtypes.Float64SerdeG{})
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	scSerde, err := ntypes.GetSumAndCountSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.storeMsgSerde, err = processor.MsgSerdeWithValueTsG[uint64](serdeFormat, commtypes.Uint64SerdeG{},
		scSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *q4Avg) Q4Avg(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	fn_out := h.setupSerde(serdeFormat)
	if fn_out != nil {
		return fn_out
	}
	ectx, err := h.getExecutionCtx(ctx, sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	kvstore, builder, snapfunc, err := setupKVStoreForAgg[uint64, ntypes.SumAndCount](
		ctx, sp,
		&execution.KVStoreParam[uint64, ntypes.SumAndCount]{
			Compare: store.Uint64LessFunc,
			CommonStoreParam: execution.CommonStoreParam[uint64, ntypes.SumAndCount]{
				StoreName:     "q4SumCountKVStore",
				SizeOfK:       nil,
				SizeOfV:       nil,
				MaxCacheBytes: q4SizePerStore,
				UseCache:      false,
				GuaranteeMth:  exactly_once_intr.GuaranteeMth(sp.GuaranteeMth),
			},
		},
		&ectx, h.storeMsgSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	tabAggProc := processor.NewMeteredProcessorG(
		processor.NewTableAggregateProcessorG[uint64, uint64, ntypes.SumAndCount]("sumCount", kvstore,
			processor.InitializerFuncG[ntypes.SumAndCount](func() optional.Option[ntypes.SumAndCount] {
				return optional.Some(ntypes.SumAndCount{
					Sum:   0,
					Count: 0,
				})
			}),
			processor.AggregatorFuncG[uint64, uint64, ntypes.SumAndCount](
				func(_ uint64, val uint64, agg optional.Option[ntypes.SumAndCount]) optional.Option[ntypes.SumAndCount] {
					aggVal := agg.Unwrap()
					return optional.Some(ntypes.SumAndCount{
						Sum:   aggVal.Sum + val,
						Count: aggVal.Count + 1,
					})
				}),
			processor.AggregatorFuncG[uint64, uint64, ntypes.SumAndCount](
				func(_ uint64, val uint64, agg optional.Option[ntypes.SumAndCount]) optional.Option[ntypes.SumAndCount] {
					aggVal := agg.Unwrap()
					return optional.Some(ntypes.SumAndCount{
						Sum:   aggVal.Sum - val,
						Count: aggVal.Count - 1,
					})
				}),
		))
	mapValProc := processor.NewTableMapValuesProcessorG[uint64, ntypes.SumAndCount, float64]("calcAvg",
		processor.ValueMapperWithKeyFuncG[uint64, ntypes.SumAndCount, float64](func(_ optional.Option[uint64], value optional.Option[ntypes.SumAndCount]) (optional.Option[float64], error) {
			sc := value.Unwrap()
			return optional.Some(float64(sc.Sum) / float64(sc.Count)), nil
		}))
	tabToStrProc := processor.NewTableToStreamProcessorG[uint64, float64]()
	outProc := processor.NewFixedSubstreamOutputProcessorG("subG4Proc", ectx.Producers()[0], sp.ParNum, h.outMsgSerde)
	tabAggProc.NextProcessor(mapValProc)
	mapValProc.NextProcessor(tabToStrProc)
	tabToStrProc.NextProcessor(outProc)
	task := stream_task.NewStreamTaskBuilder().
		MarkFinalStage().
		AppProcessFunc(stream_task.CommonAppProcessFunc(tabAggProc.Process, h.inMsgSerde)).
		Build()
	streamTaskArgs, err := builder.
		FixedOutParNum(sp.ParNum).
		Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs,
		snapfunc, func() { outProc.OutputRemainingStats() })
}
