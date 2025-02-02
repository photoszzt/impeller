package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedlog-stream/benchmark/common"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/execution"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type q6JoinStreamHandler struct {
	env         types.Environment
	funcName    string
	msgSerde    commtypes.MessageGSerdeG[uint64, *ntypes.Event]
	outMsgSerde commtypes.MessageGSerdeG[ntypes.AuctionIdSeller, *ntypes.AuctionBid]
}

func NewQ6JoinStreamHandler(env types.Environment, funcName string) types.FuncHandler {
	return &q6JoinStreamHandler{
		env:      env,
		funcName: funcName,
	}
}

func (h *q6JoinStreamHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Q6JoinStream(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *q6JoinStreamHandler) getSrcSink(sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	stream1, stream2, outputStream, err := getInOutStreams(sp)
	if err != nil {
		return nil, nil, err
	}
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	timeout := time.Duration(10) * time.Millisecond
	srcConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     timeout,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		Format:        serdeFormat,
	}
	warmup := time.Duration(sp.WarmupS) * time.Second
	consumer1, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream1,
		srcConfig, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	consumer2, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream2,
		srcConfig, sp.NumSubstreamProducer[1], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src1 := producer_consumer.NewMeteredConsumer(consumer1, warmup)
	src2 := producer_consumer.NewMeteredConsumer(consumer2, warmup)
	sink, err := producer_consumer.NewConcurrentMeteredSyncProducer(
		producer_consumer.NewShardedSharedLogStreamProducer(outputStream, outConfig), warmup)
	if err != nil {
		return nil, nil, err
	}
	src1.SetInitialSource(false)
	src2.SetInitialSource(false)
	src1.SetName("aucsByIDSrc")
	src2.SetName("bidsByAucIDSrc")
	return []*producer_consumer.MeteredConsumer{src1, src2}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *q6JoinStreamHandler) setupSerde(serdeFormat commtypes.SerdeFormat) *common.FnOutput {
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.msgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	abSerde, err := ntypes.GetAuctionBidSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	asSerde, err := ntypes.GetAuctionIdSellerSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG(serdeFormat, asSerde, abSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *q6JoinStreamHandler) setupJoin(sp *common.QueryInput) (
	proc_interface.ProcessAndReturnFunc[uint64, *ntypes.Event, uint64, *ntypes.AuctionBid],
	proc_interface.ProcessAndReturnFunc[uint64, *ntypes.Event, uint64, *ntypes.AuctionBid],
	*store.WinStoreOps,
	stream_task.SetupSnapshotCallbackFunc,
	*common.FnOutput,
) {
	jw, err := commtypes.NewJoinWindowsNoGrace(auctionDurationUpper)
	if err != nil {
		return nil, nil, nil, nil, common.GenErrFnOutput(err)
	}
	joiner := processor.ValueJoinerWithKeyTsFuncG[uint64, *ntypes.Event, *ntypes.Event, *ntypes.AuctionBid](
		func(readOnlyKey uint64, value1 *ntypes.Event, value2 *ntypes.Event, leftTs, otherTs int64) optional.Option[*ntypes.AuctionBid] {
			auc := value1.NewAuction
			bid := value2.Bid
			return optional.Some(&ntypes.AuctionBid{
				BidDateTime: bid.DateTime,
				BidPrice:    bid.Price,
				AucDateTime: auc.DateTime,
				AucExpires:  auc.Expires,
				AucSeller:   auc.Seller,
				AucCategory: auc.Category,
			})
		})
	aucMp := getMaterializedParam[uint64, *ntypes.Event](
		"q6AuctionsByIDStore", h.msgSerde, sp)
	bidMp := getMaterializedParam[uint64, *ntypes.Event](
		"q6BidsByAuctionIDStore", h.msgSerde, sp)
	aucJoinBidsFunc, bidsJoinAucFunc, wsos, setupSnapCallbackFunc, err := execution.SetupSkipMapStreamStreamJoin(
		aucMp, bidMp, store.IntegerCompare[uint64], joiner, jw,
		exactly_once_intr.GuaranteeMth(sp.GuaranteeMth))
	if err != nil {
		return nil, nil, nil, nil, common.GenErrFnOutput(err)
	}
	return aucJoinBidsFunc, bidsJoinAucFunc, wsos, setupSnapCallbackFunc, nil
}

func (h *q6JoinStreamHandler) Q6JoinStream(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	fn_out := h.setupSerde(serdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks_arr, err := h.getSrcSink(sp)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("getSrcSink err: %v\n", err))
	}
	aucJoinBidsFunc, bidsJoinAucFunc, wsos, setupSnapCallbackFunc, fn_out := h.setupJoin(sp)
	if fn_out != nil {
		return fn_out
	}
	getValidBid := processor.NewMeteredProcessorG(
		processor.NewStreamFilterProcessorG[uint64, *ntypes.AuctionBid]("getValidBid",
			processor.PredicateFuncG[uint64, *ntypes.AuctionBid](
				func(key optional.Option[uint64], value optional.Option[*ntypes.AuctionBid]) (bool, error) {
					ab := value.Unwrap()
					return ab.BidDateTime >= ab.AucDateTime && ab.BidDateTime <= ab.AucExpires, nil
				})))
	filterAndGroupMsg := func(ctx context.Context,
		msgs []commtypes.MessageG[uint64, *ntypes.AuctionBid],
	) ([]commtypes.MessageG[ntypes.AuctionIdSeller, *ntypes.AuctionBid], error) {
		var newMsgs []commtypes.MessageG[ntypes.AuctionIdSeller, *ntypes.AuctionBid]
		for _, msg := range msgs {
			validBid, err := getValidBid.ProcessAndReturn(ctx, msg)
			if err != nil {
				return nil, err
			}
			if validBid != nil && validBid[0].Value.IsSome() {
				k := validBid[0].Key.Unwrap()
				v := validBid[0].Value.Unwrap()
				aic := ntypes.AuctionIdSeller{
					AucId:  k,
					Seller: v.AucSeller,
				}
				newMsg := commtypes.MessageG[ntypes.AuctionIdSeller, *ntypes.AuctionBid]{
					Key: optional.Some(aic), Value: validBid[0].Value,
					TimestampMs:   validBid[0].TimestampMs,
					StartProcTime: validBid[0].StartProcTime,
				}
				newMsgs = append(newMsgs, newMsg)
			}
		}
		return newMsgs, nil
	}
	aJoinB := execution.JoinWorkerFunc[uint64, *ntypes.Event, ntypes.AuctionIdSeller, *ntypes.AuctionBid](
		func(ctx context.Context, m commtypes.MessageG[uint64, *ntypes.Event]) ([]commtypes.MessageG[ntypes.AuctionIdSeller, *ntypes.AuctionBid], error) {
			joined, err := aucJoinBidsFunc(ctx, m)
			if err != nil {
				return nil, err
			}
			return filterAndGroupMsg(ctx, joined)
		})

	bJoinA := execution.JoinWorkerFunc[uint64, *ntypes.Event, ntypes.AuctionIdSeller, *ntypes.AuctionBid](
		func(ctx context.Context, m commtypes.MessageG[uint64, *ntypes.Event]) ([]commtypes.MessageG[ntypes.AuctionIdSeller, *ntypes.AuctionBid], error) {
			joined, err := bidsJoinAucFunc(ctx, m)
			if err != nil {
				return nil, err
			}
			return filterAndGroupMsg(ctx, joined)
		})
	msgSerdePair := execution.NewMsgSerdePair(h.msgSerde, h.outMsgSerde)
	task, procArgs := execution.PrepareTaskWithJoin(
		ctx, aJoinB, bJoinA, proc_interface.NewBaseSrcsSinks(srcs, sinks_arr),
		proc_interface.NewBaseProcArgs(h.funcName, sp.ScaleEpoch, sp.ParNum), false,
		msgSerdePair, msgSerdePair, "subG2")
	builder := streamArgsBuilderForJoin(procArgs, sp)
	builder = execution.StreamArgsSetWinStore(wsos, builder,
		exactly_once_intr.GuaranteeMth(sp.GuaranteeMth))
	streamTaskArgs, err := builder.
		FixedOutParNum(sp.ParNum).
		Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, setupSnapCallbackFunc, func() { procArgs.OutputRemainingStats() })
}
