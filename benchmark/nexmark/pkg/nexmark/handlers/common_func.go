package handlers

import (
	"context"
	"fmt"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/execution"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/store_with_changelog"
	"sharedlog-stream/pkg/stream_task"
	"time"
)

func only_bid(key optional.Option[string], value optional.Option[*ntypes.Event]) (bool, error) {
	v := value.Unwrap()
	return v.Etype == ntypes.BID, nil
}

func compareStartEndTime(a, b ntypes.StartEndTime) bool {
	return ntypes.CompareStartEndTime(a, b) < 0
}

func getMaterializedParam[K, V any](storeName string,
	kvMsgSerde commtypes.MessageGSerdeG[K, V],
	sp *common.QueryInput,
) *store_with_changelog.MaterializeParam[K, V] {
	r := store_with_changelog.NewMaterializeParamBuilder[K, V]().
		MessageSerde(kvMsgSerde).
		StoreName(storeName).
		ParNum(sp.ParNum).
		SerdeFormat(commtypes.SerdeFormat(sp.SerdeFormat)).
		ChangelogManagerParam(commtypes.CreateChangelogManagerParam{
			NumPartition:  sp.NumChangelogPartition,
			TimeOut:       common.SrcConsumeTimeout,
			FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		}).BufMaxSize(sp.BufMaxSize).Build()
	debug.Assert(r.ParNum() == sp.ParNum, "parNum should be set")
	debug.Assert(r.StoreName() == storeName, "storeName should be set")
	return r
}

func streamArgsBuilder(
	ectx *processor.BaseExecutionContext,
	sp *common.QueryInput,
) stream_task.BuildStreamTaskArgs {
	transactionalID := fmt.Sprintf("%s-%s-%d-%s", ectx.FuncName(), sp.InputTopicNames[0],
		sp.ParNum, sp.OutputTopicNames[0])
	return benchutil.UpdateStreamTaskArgs(sp,
		stream_task.NewStreamTaskArgsBuilder(ectx, transactionalID))
}

func streamArgsBuilderForJoin(
	ectx processor.ExecutionContext,
	sp *common.QueryInput,
) stream_task.BuildStreamTaskArgs {
	transactionalID := fmt.Sprintf("%s-%d", ectx.FuncName(), sp.ParNum)
	return benchutil.UpdateStreamTaskArgs(sp,
		stream_task.NewStreamTaskArgsBuilder(ectx, transactionalID))
}

func setupKVStoreForAgg[K comparable, V any](
	ctx context.Context,
	sp *common.QueryInput,
	p *execution.KVStoreParam[K, V],
	ectx *processor.BaseExecutionContext,
	msgSerde commtypes.MessageGSerdeG[K, commtypes.ValueTimestampG[V]],
) (
	store.CachedKeyValueStore[K, commtypes.ValueTimestampG[V]],
	stream_task.BuildStreamTaskArgs,
	stream_task.SetupSnapshotCallbackFunc,
	error,
) {
	p.CommonStoreParam.GuaranteeMth = exactly_once_intr.GuaranteeMth(sp.GuaranteeMth)
	mp := getMaterializedParam[K, commtypes.ValueTimestampG[V]](
		p.StoreName, msgSerde, sp)
	store, kvos, f, err := execution.GetKVStore(ctx, p, mp)
	if err != nil {
		return nil, nil, nil, err
	}
	builder := streamArgsBuilder(ectx, sp)
	builder = execution.StreamArgsSetKVStore(kvos, builder, p.GuaranteeMth)
	return store, builder, f, nil
}

func setupWinStoreForAgg[K comparable, V any](
	ctx context.Context,
	sp *common.QueryInput,
	p *execution.WinStoreParam[K, V],
	ectx *processor.BaseExecutionContext,
	msgSerde commtypes.MessageGSerdeG[K, commtypes.ValueTimestampG[V]],
) (
	store.CachedWindowStateStore[K, commtypes.ValueTimestampG[V]],
	stream_task.BuildStreamTaskArgs,
	stream_task.SetupSnapshotCallbackFunc,
	error,
) {
	p.RetainDuplicates = false
	p.CommonStoreParam.GuaranteeMth = exactly_once_intr.GuaranteeMth(sp.GuaranteeMth)
	mp := getMaterializedParam[K, commtypes.ValueTimestampG[V]](
		p.StoreName, msgSerde, sp)
	store, wsos, f, err := execution.GetWinStore(ctx, p, mp)
	if err != nil {
		return nil, nil, nil, err
	}
	builder := streamArgsBuilder(ectx, sp)
	builder = execution.StreamArgsSetWinStore(wsos, builder, p.GuaranteeMth)
	return store, builder, f, nil
}

func getSrcSink(ctx context.Context, sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	input_stream, output_streams, err := benchutil.GetShardedInputOutputStreams(ctx, sp)
	if err != nil {
		return nil, nil, err
	}
	debug.Assert(len(output_streams) == 1, "expected only one output stream")
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	inConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     common.SrcConsumeTimeout,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		Format:        serdeFormat,
	}
	warmup := time.Duration(sp.WarmupS) * time.Second
	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(input_stream, inConfig, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src := producer_consumer.NewMeteredConsumer(consumer, warmup)
	sink, err := producer_consumer.NewMeteredProducer(producer_consumer.NewShardedSharedLogStreamProducer(output_streams[0], outConfig), warmup)
	if err != nil {
		return nil, nil, err
	}
	return []*producer_consumer.MeteredConsumer{src}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func getSrcSinkUint64Key(
	ctx context.Context,
	sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	input_stream, output_streams, err := benchutil.GetShardedInputOutputStreams(ctx, sp)
	if err != nil {
		return nil, nil, err
	}
	debug.Assert(len(output_streams) == 1, "expected only one output stream")
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	warmup := time.Duration(sp.WarmupS) * time.Second
	inConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     common.SrcConsumeTimeout,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		Format:        serdeFormat,
	}
	consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(input_stream, inConfig, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src := producer_consumer.NewMeteredConsumer(consumer, warmup)
	sink, err := producer_consumer.NewMeteredProducer(producer_consumer.NewShardedSharedLogStreamProducer(output_streams[0], outConfig), warmup)
	if err != nil {
		return nil, nil, err
	}
	return []*producer_consumer.MeteredConsumer{src}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func GetSerdeFromString(serdeStr string, serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeStr {
	case "StartEndTime":
		return ntypes.GetStartEndTimeSerde(serdeFormat)
	case "AuctionIdCount":
		return ntypes.GetAuctionIdCountSerde(serdeFormat)
	case "AuctionIdCntMax":
		return ntypes.GetAuctionIdCntMaxSerde(serdeFormat)
	case "AuctionIdCategory":
		return ntypes.GetAuctionIdCategorySerde(serdeFormat)
	case "AuctionBid":
		return ntypes.GetAuctionBidSerde(serdeFormat)
	case "AuctionIdSeller":
		return ntypes.GetAuctionIdSellerSerde(serdeFormat)
	case "ChangeUint64":
		return commtypes.GetChangeSerde(serdeFormat, commtypes.Uint64Serde{})
	case "ChangePriceTime":
		ptSerde, err := ntypes.GetPriceTimeSerde(serdeFormat)
		if err != nil {
			return nil, err
		}
		return commtypes.GetChangeSerde(serdeFormat, ptSerde)
	case "BidAndMax":
		return ntypes.GetBidAndMaxSerde(serdeFormat)
	case "Event":
		return ntypes.GetEventSerde(serdeFormat)
	case "Uint64":
		return commtypes.Uint64Serde{}, nil
	case "Float64":
		return commtypes.Float64Serde{}, nil
	case "String":
		return commtypes.StringSerde{}, nil
	case "PersonTime":
		return ntypes.GetPersonTimeSerde(serdeFormat)
	case "PriceTime":
		return ntypes.GetPriceTimeSerde(serdeFormat)
	case "NameCityStateId":
		return ntypes.GetNameCityStateIdSerde(serdeFormat)
	default:
		return nil, fmt.Errorf("Unrecognized serde string %s", serdeStr)
	}
}
