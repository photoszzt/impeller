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
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

const (
	q5SizePerStore = 5 * 1024 * 1024
)

type q5MaxBid struct {
	env         types.Environment
	funcName    string
	useCache    bool
	inMsgSerde  commtypes.MessageGSerdeG[ntypes.StartEndTime, ntypes.AuctionIdCount]
	outMsgSerde commtypes.MessageGSerdeG[ntypes.StartEndTime, ntypes.AuctionIdCntMax]
	msgSerde    commtypes.MessageGSerdeG[ntypes.StartEndTime, commtypes.ValueTimestampG[uint64]]
}

func NewQ5MaxBid(env types.Environment, funcName string) types.FuncHandler {
	useCache := common.CheckCacheConfig()
	return &q5MaxBid{
		env:      env,
		funcName: funcName,
		useCache: useCache,
	}
}

func (h *q5MaxBid) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.processQ5MaxBid(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *q5MaxBid) getSrcSink(ctx context.Context, sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	input_stream, output_streams, err := benchutil.GetShardedInputOutputStreams(ctx, sp)
	if err != nil {
		return nil, nil, err
	}
	debug.Assert(len(output_streams) == 1, "expected only one output stream")

	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	warmup := time.Duration(sp.WarmupS) * time.Second
	inConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     time.Duration(20) * time.Second,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		Format:        serdeFormat,
	}
	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(input_stream,
		inConfig, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src := producer_consumer.NewMeteredConsumer(consumer, warmup)
	sink, err := producer_consumer.NewConcurrentMeteredSyncProducer(producer_consumer.NewShardedSharedLogStreamProducer(output_streams[0], outConfig),
		warmup)
	if err != nil {
		return nil, nil, err
	}
	src.SetInitialSource(false)
	sink.MarkFinalOutput()
	return []*producer_consumer.MeteredConsumer{src}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *q5MaxBid) setupSerde(serdeFormat commtypes.SerdeFormat) *common.FnOutput {
	var seSerde commtypes.SerdeG[ntypes.StartEndTime]
	var aucIdCountSerde commtypes.SerdeG[ntypes.AuctionIdCount]
	var aucIdCountMaxSerde commtypes.SerdeG[ntypes.AuctionIdCntMax]
	var err error
	if serdeFormat == commtypes.JSON {
		seSerde = ntypes.StartEndTimeJSONSerdeG{}
		aucIdCountSerde = ntypes.AuctionIdCountJSONSerdeG{}
		aucIdCountMaxSerde = ntypes.AuctionIdCntMaxJSONSerdeG{}
	} else if serdeFormat == commtypes.MSGP {
		seSerde = ntypes.StartEndTimeMsgpSerdeG{}
		aucIdCountSerde = ntypes.AuctionIdCountMsgpSerdeG{}

		aucIdCountMaxSerde = ntypes.AuctionIdCntMaxMsgpSerdeG{}
	} else {
		return common.GenErrFnOutput(
			fmt.Errorf("serde format should be either json or msgp; but %v is given", serdeFormat))
	}
	h.inMsgSerde, err = commtypes.GetMsgGSerdeG(serdeFormat, seSerde, aucIdCountSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG(serdeFormat, seSerde, aucIdCountMaxSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.msgSerde, err = processor.MsgSerdeWithValueTsG[ntypes.StartEndTime, uint64](serdeFormat,
		seSerde, commtypes.Uint64SerdeG{})
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *q5MaxBid) processQ5MaxBid(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	fn_out := h.setupSerde(serdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks_arr, err := h.getSrcSink(ctx, sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	ectx := processor.NewExecutionContext(srcs,
		sinks_arr, h.funcName, sp.ScaleEpoch, sp.ParNum)
	// gua := exactly_once_intr.GuaranteeMth(sp.GuaranteeMth)
	// useCache := benchutil.UseCache(h.useCache, gua)
	aggStore, builder, snapfunc, err := setupKVStoreForAgg(ctx, sp,
		&execution.KVStoreParam[ntypes.StartEndTime, uint64]{
			Compare: compareStartEndTime,
			CommonStoreParam: execution.CommonStoreParam[ntypes.StartEndTime, uint64]{
				StoreName:     "q5MaxBidKVStore",
				SizeOfK:       ntypes.SizeOfStartEndTime,
				SizeOfV:       commtypes.SizeOfUint64,
				MaxCacheBytes: q5SizePerStore,
				UseCache:      h.useCache,
			},
		}, &ectx, h.msgSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	maxBid := processor.NewMeteredProcessorG(
		processor.NewStreamAggregateProcessorG[ntypes.StartEndTime, ntypes.AuctionIdCount, uint64]("maxBid",
			aggStore, processor.InitializerFuncG[uint64](func() optional.Option[uint64] {
				return optional.Some(uint64(0))
			}),
			processor.AggregatorFuncG[ntypes.StartEndTime, ntypes.AuctionIdCount, uint64](
				func(key ntypes.StartEndTime, v ntypes.AuctionIdCount, agg optional.Option[uint64]) optional.Option[uint64] {
					aggVal := agg.Unwrap()
					if v.Count > aggVal {
						return optional.Some(v.Count)
					}
					return agg
				}), h.useCache))
	stJoin := processor.NewMeteredProcessorG[ntypes.StartEndTime, ntypes.AuctionIdCount, ntypes.StartEndTime, ntypes.AuctionIdCntMax](
		processor.NewStreamTableJoinProcessorG[ntypes.StartEndTime, ntypes.AuctionIdCount, uint64, ntypes.AuctionIdCntMax](aggStore,
			processor.ValueJoinerWithKeyFuncG[ntypes.StartEndTime, ntypes.AuctionIdCount, uint64, ntypes.AuctionIdCntMax](
				func(readOnlyKey ntypes.StartEndTime, lv ntypes.AuctionIdCount, rv uint64) optional.Option[ntypes.AuctionIdCntMax] {
					return optional.Some(ntypes.AuctionIdCntMax{
						AucId:  lv.AucId,
						Count:  lv.Count,
						MaxCnt: rv,
					})
				})))
	chooseMaxCnt := processor.NewStreamFilterProcessorG[ntypes.StartEndTime, ntypes.AuctionIdCntMax]("chooseMaxCnt",
		processor.PredicateFuncG[ntypes.StartEndTime, ntypes.AuctionIdCntMax](func(key optional.Option[ntypes.StartEndTime], value optional.Option[ntypes.AuctionIdCntMax]) (bool, error) {
			v := value.Unwrap()
			return v.Count >= v.MaxCnt, nil
		}))
	outProc := processor.NewFixedSubstreamOutputProcessorG("subG3Proc", sinks_arr[0],
		ectx.SubstreamNum(), h.outMsgSerde)
	stJoin.NextProcessor(chooseMaxCnt)
	chooseMaxCnt.NextProcessor(outProc)
	task := stream_task.NewStreamTaskBuilder().
		AppProcessFunc(stream_task.CommonAppProcessFunc(
			func(ctx context.Context, msg commtypes.MessageG[ntypes.StartEndTime, ntypes.AuctionIdCount]) error {
				// fmt.Fprintf(os.Stderr, "got msg with key: %v, val: %v, ts: %v\n", msg.Msg.Key, msg.Msg.Value, msg.Msg.Timestamp)
				_, err := maxBid.ProcessAndReturn(ctx, msg)
				if err != nil {
					return fmt.Errorf("maxBid err: %v", err)
				}
				return stJoin.Process(ctx, msg)
			},
			h.inMsgSerde)).
		MarkFinalStage().
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
