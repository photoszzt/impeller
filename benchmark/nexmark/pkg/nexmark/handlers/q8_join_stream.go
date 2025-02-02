package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/execution"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/stream_task"
	"sharedlog-stream/pkg/utils"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type q8JoinStreamHandler struct {
	env         types.Environment
	funcName    string
	msgSerde    commtypes.MessageGSerdeG[uint64, *ntypes.Event]
	outMsgSerde commtypes.MessageGSerdeG[uint64, ntypes.PersonTime]
}

func NewQ8JoinStreamHandler(env types.Environment, funcName string) types.FuncHandler {
	return &q8JoinStreamHandler{
		env:      env,
		funcName: funcName,
	}
}

func (h *q8JoinStreamHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "inputParam: %+v\n", parsedInput)
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Query8JoinStream(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("query 3 output: %v\n", encodedOutput)
	return common.CompressData(encodedOutput), nil
}

func (h *q8JoinStreamHandler) getSrcSink(sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	stream1, stream2, outputStream, err := getInOutStreams(sp)
	if err != nil {
		return nil, nil, err
	}
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	// timeout := common.SrcConsumeTimeout
	warmup := time.Duration(sp.WarmupS) * time.Second
	srcConfig := producer_consumer.StreamConsumerConfig{
		Timeout:     common.SrcConsumeTimeout,
		SerdeFormat: serdeFormat,
	}
	consumer1, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream1,
		&srcConfig, sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	consumer2, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream2,
		&srcConfig, sp.NumSubstreamProducer[1], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src1 := producer_consumer.NewMeteredConsumer(consumer1, warmup)
	src2 := producer_consumer.NewMeteredConsumer(consumer2, warmup)
	sink, err := producer_consumer.NewConcurrentMeteredSyncProducer(producer_consumer.NewShardedSharedLogStreamProducer(outputStream,
		&producer_consumer.StreamSinkConfig{
			FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
			Format:        serdeFormat,
		}), warmup)
	if err != nil {
		return nil, nil, err
	}
	src1.SetInitialSource(false)
	src2.SetInitialSource(false)
	src1.SetName("auctionsBySellerIDSrc")
	src2.SetName("personsByIDSrc")
	sink.MarkFinalOutput()
	return []*producer_consumer.MeteredConsumer{src1, src2}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *q8JoinStreamHandler) setupSerde(sf uint8) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sf)
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.msgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	ptSerde, err := ntypes.GetPersonTimeSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, ptSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *q8JoinStreamHandler) setupJoin(sp *common.QueryInput) (
	proc_interface.ProcessAndReturnFunc[uint64, *ntypes.Event, uint64, ntypes.PersonTime],
	proc_interface.ProcessAndReturnFunc[uint64, *ntypes.Event, uint64, ntypes.PersonTime],
	*store.WinStoreOps,
	stream_task.SetupSnapshotCallbackFunc,
	*common.FnOutput,
) {
	joinWindows, err := commtypes.NewJoinWindowsWithGrace(time.Duration(10)*time.Second, time.Duration(5)*time.Second)
	if err != nil {
		return nil, nil, nil, nil, common.GenErrFnOutput(err)
	}
	windowSizeMs := int64(10 * 1000)
	joiner := processor.ValueJoinerWithKeyTsFuncG[uint64, *ntypes.Event, *ntypes.Event, ntypes.PersonTime](
		func(readOnlyKey uint64, leftValue *ntypes.Event, rightValue *ntypes.Event,
			leftTs int64, rightTs int64,
		) optional.Option[ntypes.PersonTime] {
			// fmt.Fprint(os.Stderr, "get into joiner\n")
			ts := rightValue.NewPerson.DateTime
			windowStart := (utils.MaxInt64(0, ts-windowSizeMs+windowSizeMs) / windowSizeMs) * windowSizeMs
			return optional.Some(ntypes.PersonTime{
				ID:        rightValue.NewPerson.ID,
				Name:      rightValue.NewPerson.Name,
				StartTime: windowStart,
			})
		})
	aucMp := getMaterializedParam[uint64, *ntypes.Event](
		"q8AuctionsBySellerIDWinTab", h.msgSerde, sp)
	perMp := getMaterializedParam[uint64, *ntypes.Event](
		"q8PersonsByIDWinTab", h.msgSerde, sp)
	aucJoinsPerFunc, perJoinsAucFunc, wsos, setupSnapCallbackFunc, err := execution.SetupSkipMapStreamStreamJoin(
		aucMp, perMp, store.IntegerCompare[uint64], joiner, joinWindows,
		exactly_once_intr.GuaranteeMth(sp.GuaranteeMth))
	if err != nil {
		return nil, nil, nil, nil, common.GenErrFnOutput(err)
	}
	return aucJoinsPerFunc, perJoinsAucFunc, wsos, setupSnapCallbackFunc, nil
}

func (h *q8JoinStreamHandler) Query8JoinStream(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	debug.Assert(sp.ScaleEpoch != 0, "scale epoch should start from 1")
	fn_out := h.setupSerde(sp.SerdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks_arr, err := h.getSrcSink(sp)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("getSrcSink err: %v\n", err))
	}
	aucJoinsPerFunc, perJoinsAucFunc, wsos, setupSnapCallbackFunc, fn_out := h.setupJoin(sp)
	if fn_out != nil {
		return fn_out
	}
	msgSerdePair := execution.NewMsgSerdePair(h.msgSerde, h.outMsgSerde)
	task, procArgs := execution.PrepareTaskWithJoin(ctx,
		execution.JoinWorkerFunc[uint64, *ntypes.Event, uint64, ntypes.PersonTime](aucJoinsPerFunc),
		execution.JoinWorkerFunc[uint64, *ntypes.Event, uint64, ntypes.PersonTime](perJoinsAucFunc),
		proc_interface.NewBaseSrcsSinks(srcs, sinks_arr),
		proc_interface.NewBaseProcArgs(h.funcName, sp.ScaleEpoch, sp.ParNum), true,
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
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, setupSnapCallbackFunc, func() {
		procArgs.OutputRemainingStats()
	})
}
