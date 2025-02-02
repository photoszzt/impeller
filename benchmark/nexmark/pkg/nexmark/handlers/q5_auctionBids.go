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
	"sharedlog-stream/pkg/execution"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type q5AuctionBids struct {
	env          types.Environment
	funcName     string
	useCache     bool
	srcMsgSerde  commtypes.MessageGSerdeG[uint64, *ntypes.Event]
	sinkMsgSerde commtypes.MessageGSerdeG[ntypes.StartEndTime, ntypes.AuctionIdCount]
	msgSerde     commtypes.MessageGSerdeG[uint64, commtypes.ValueTimestampG[uint64]]
}

func NewQ5AuctionBids(env types.Environment, funcName string) *q5AuctionBids {
	useCache := common.CheckCacheConfig()
	return &q5AuctionBids{
		env:      env,
		funcName: funcName,
		useCache: useCache,
	}
}

func (h *q5AuctionBids) Call(ctx context.Context, input []byte) ([]byte, error) {
	sp := &common.QueryInput{}
	err := json.Unmarshal(input, sp)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.processQ5AuctionBids(ctx, sp)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return common.CompressData(encodedOutput), nil
}

func (h *q5AuctionBids) getSrcSink(ctx context.Context, sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	input_stream, output_streams, err := benchutil.GetShardedInputOutputStreams(ctx, sp)
	if err != nil {
		return nil, nil, err
	}
	debug.Assert(len(output_streams) == 1, "expected only one output stream")
	warmup := time.Duration(sp.WarmupS) * time.Second
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(input_stream,
		&producer_consumer.StreamConsumerConfig{
			Timeout:     time.Duration(5) * time.Second,
			SerdeFormat: serdeFormat,
		}, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src := producer_consumer.NewMeteredConsumer(consumer, warmup)
	sink, err := producer_consumer.NewConcurrentMeteredSyncProducer(
		producer_consumer.NewShardedSharedLogStreamProducer(output_streams[0],
			&producer_consumer.StreamSinkConfig{
				FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
				Format:        serdeFormat,
			}), warmup)
	if err != nil {
		return nil, nil, err
	}
	src.SetInitialSource(false)
	return []*producer_consumer.MeteredConsumer{src}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *q5AuctionBids) getCountAggProc(ctx context.Context, sp *common.QueryInput,
	ectx *processor.BaseExecutionContext,
) (countProc *processor.MeteredProcessorG[uint64, *ntypes.Event, commtypes.WindowedKeyG[uint64], commtypes.ChangeG[uint64]],
	builder stream_task.BuildStreamTaskArgs,
	setSnapCallbackFunc stream_task.SetupSnapshotCallbackFunc,
	err error,
) {
	hopWindow, err := commtypes.NewTimeWindowsWithGrace(time.Duration(10)*time.Second, time.Duration(5)*time.Second)
	if err != nil {
		return nil, nil, nil, err
	}
	hopWindow, err = hopWindow.AdvanceBy(time.Duration(2) * time.Second)
	if err != nil {
		return nil, nil, nil, err
	}
	// useCache := benchutil.UseCache(h.useCache, exactly_once_intr.GuaranteeMth(sp.GuaranteeMth))
	cachedStore, builder, setSnapCallbackFunc, err := setupWinStoreForAgg(
		ctx, sp, &execution.WinStoreParam[uint64, uint64]{
			CmpFunc:    store.IntegerCompare[uint64],
			JoinWindow: hopWindow,
			CommonStoreParam: execution.CommonStoreParam[uint64, uint64]{
				StoreName:     "q5AuctionBidsCountStore",
				SizeOfK:       commtypes.SizeOfUint64,
				SizeOfV:       commtypes.SizeOfUint64,
				MaxCacheBytes: q5SizePerStore,
				UseCache:      h.useCache,
			},
		}, ectx, h.msgSerde)
	if err != nil {
		return nil, nil, nil, err
	}
	countProc = processor.NewMeteredProcessorG(processor.NewStreamWindowAggregateProcessorG[uint64, *ntypes.Event, uint64](
		"countProc", cachedStore,
		processor.InitializerFuncG[uint64](func() optional.Option[uint64] { return optional.Some(uint64(0)) }),
		processor.AggregatorFuncG[uint64, *ntypes.Event, uint64](func(key uint64, value *ntypes.Event, aggregate optional.Option[uint64]) optional.Option[uint64] {
			val := aggregate.Unwrap()
			return optional.Some(val + 1)
		}), hopWindow, h.useCache))
	return countProc, builder, setSnapCallbackFunc, nil
}

func (h *q5AuctionBids) setupSerde(sf uint8) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sf)
	var seSerde commtypes.SerdeG[ntypes.StartEndTime]
	var aucIdCountSerde commtypes.SerdeG[ntypes.AuctionIdCount]
	if serdeFormat == commtypes.JSON {
		seSerde = ntypes.StartEndTimeJSONSerdeG{}
		aucIdCountSerde = ntypes.AuctionIdCountJSONSerdeG{}
	} else if serdeFormat == commtypes.MSGP {
		seSerde = ntypes.StartEndTimeMsgpSerdeG{}
		aucIdCountSerde = ntypes.AuctionIdCountMsgpSerdeG{}
	} else {
		return common.GenErrFnOutput(fmt.Errorf("serde format should be either json or msgp; but %v is given", sf))
	}

	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.srcMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat,
		commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.sinkMsgSerde, err = commtypes.GetMsgGSerdeG(serdeFormat, seSerde, aucIdCountSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.msgSerde, err = processor.MsgSerdeWithValueTsG[uint64, uint64](serdeFormat,
		commtypes.Uint64SerdeG{}, commtypes.Uint64SerdeG{})
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *q5AuctionBids) processQ5AuctionBids(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	fn_out := h.setupSerde(sp.SerdeFormat)
	if fn_out != nil {
		return nil
	}
	srcs, sinks, err := h.getSrcSink(ctx, sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	ectx := processor.NewExecutionContext(srcs,
		sinks, h.funcName, sp.ScaleEpoch, sp.ParNum)
	countProc, builder, setSnapCallbackFunc, err := h.getCountAggProc(ctx, sp, &ectx)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	tabToStreamProc := processor.NewTableToStreamProcessorG[commtypes.WindowedKeyG[uint64], uint64]()
	mapProc := processor.NewStreamMapProcessorG[commtypes.WindowedKeyG[uint64], uint64, ntypes.StartEndTime, ntypes.AuctionIdCount]("groupByAuction",
		processor.MapperFuncG[commtypes.WindowedKeyG[uint64], uint64, ntypes.StartEndTime, ntypes.AuctionIdCount](
			func(key optional.Option[commtypes.WindowedKeyG[uint64]], value optional.Option[uint64]) (optional.Option[ntypes.StartEndTime], optional.Option[ntypes.AuctionIdCount], error) {
				k := key.Unwrap()
				v := value.Unwrap()
				newKey := ntypes.StartEndTime{
					StartTimeMs: k.Window.Start(),
					EndTimeMs:   k.Window.End(),
				}
				newVal := ntypes.AuctionIdCount{
					AucId: k.Key,
					Count: v,
				}
				return optional.Some(newKey), optional.Some(newVal), nil
			}))
	outProc := processor.NewGroupByOutputProcessorG("subG2Proc", sinks[0], &ectx, h.sinkMsgSerde)
	countProc.NextProcessor(tabToStreamProc)
	tabToStreamProc.NextProcessor(mapProc)
	mapProc.NextProcessor(outProc)

	task := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(stream_task.CommonAppProcessFunc(countProc.Process, h.srcMsgSerde)).
		Build()
	streamTaskArgs, err := builder.Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, setSnapCallbackFunc, func() { outProc.OutputRemainingStats() })
}
