package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/stream_task"
	"time"

	"cs.utexas.edu/zjia/faas/types"
)

type q3JoinTableHandler struct {
	env           types.Environment
	funcName      string
	inMsgSerde    commtypes.MessageGSerdeG[uint64, *ntypes.Event]
	outMsgSerde   commtypes.MessageGSerdeG[uint64, ntypes.NameCityStateId]
	storeMsgSerde commtypes.MessageGSerdeG[uint64, commtypes.ValueTimestampG[*ntypes.Event]]
}

func NewQ3JoinTableHandler(env types.Environment, funcName string) types.FuncHandler {
	return &q3JoinTableHandler{
		env:      env,
		funcName: funcName,
	}
}

func (h *q3JoinTableHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.Query3JoinTable(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("query 3 output: %v\n", encodedOutput)
	return common.CompressData(encodedOutput), nil
}

func getInOutStreams(
	input *common.QueryInput,
) (*sharedlog_stream.ShardedSharedLogStream, /* auction */
	*sharedlog_stream.ShardedSharedLogStream, /* person */
	*sharedlog_stream.ShardedSharedLogStream, /* output */
	error,
) {
	inputStream1, err := sharedlog_stream.NewShardedSharedLogStream(input.InputTopicNames[0],
		input.NumInPartition, commtypes.SerdeFormat(input.SerdeFormat), input.BufMaxSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("NewSharedlogStream for input stream failed: %v", err)
	}
	inputStream2, err := sharedlog_stream.NewShardedSharedLogStream(input.InputTopicNames[1], input.NumInPartition,
		commtypes.SerdeFormat(input.SerdeFormat), input.BufMaxSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("NewSharedlogStream for input stream failed: %v", err)
	}
	outputStream, err := sharedlog_stream.NewShardedSharedLogStream(input.OutputTopicNames[0], input.NumOutPartitions[0],
		commtypes.SerdeFormat(input.SerdeFormat), input.BufMaxSize)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("NewSharedlogStream for output stream failed: %v", err)
	}
	return inputStream1, inputStream2, outputStream, nil
}

func (h *q3JoinTableHandler) getSrcSink(sp *common.QueryInput,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	stream1, stream2, outputStream, err := getInOutStreams(sp)
	if err != nil {
		return nil, nil, err
	}
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)

	timeout := time.Duration(4) * time.Millisecond
	warmup := time.Duration(sp.WarmupS) * time.Second
	inConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     timeout,
		SerdeFormat: serdeFormat,
	}
	consumer1, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream1, inConfig,
		sp.NumSubstreamProducer[0], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	consumer2, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream2, inConfig,
		sp.NumSubstreamProducer[1], sp.ParNum)
	if err != nil {
		return nil, nil, err
	}
	src1 := producer_consumer.NewMeteredConsumer(consumer1, warmup)
	src2 := producer_consumer.NewMeteredConsumer(consumer2, warmup)
	src1.SetInitialSource(false)
	src2.SetInitialSource(false)
	sink, err := producer_consumer.NewConcurrentMeteredSyncProducer(producer_consumer.NewShardedSharedLogStreamProducer(outputStream,
		&producer_consumer.StreamSinkConfig{
			FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
			Format:        serdeFormat,
		}), warmup)
	if err != nil {
		return nil, nil, err
	}
	sink.MarkFinalOutput()
	return []*producer_consumer.MeteredConsumer{src1, src2}, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *q3JoinTableHandler) setupSerde(serdeFormat commtypes.SerdeFormat) *common.FnOutput {
	eventSerde, err := ntypes.GetEventSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("get event serde err: %v", err))
	}
	h.inMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("get msg serde err: %v", err))
	}
	ncsiSerde, err := ntypes.GetNameCityStateIdSerdeG(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.outMsgSerde, err = commtypes.GetMsgGSerdeG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, ncsiSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	h.storeMsgSerde, err = processor.MsgSerdeWithValueTsG[uint64](serdeFormat, commtypes.Uint64SerdeG{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return nil
}

func (h *q3JoinTableHandler) setupQ3Join(sp *common.QueryInput) (
	proc_interface.ProcessAndReturnFunc[uint64, *ntypes.Event, uint64, commtypes.ChangeG[ntypes.NameCityStateId]],
	proc_interface.ProcessAndReturnFunc[uint64, *ntypes.Event, uint64, commtypes.ChangeG[ntypes.NameCityStateId]],
	*store.KVStoreOps,
	stream_task.SetupSnapshotCallbackFunc,
	error,
) {
	joiner := processor.ValueJoinerWithKeyFuncG[uint64, *ntypes.Event, *ntypes.Event, ntypes.NameCityStateId](
		func(_ uint64, _ *ntypes.Event, rightVal *ntypes.Event) optional.Option[ntypes.NameCityStateId] {
			ncsi := ntypes.NameCityStateId{
				Name:  rightVal.NewPerson.Name,
				City:  rightVal.NewPerson.City,
				State: rightVal.NewPerson.State,
				ID:    rightVal.NewPerson.ID,
			}
			// debug.Fprintf(os.Stderr, "join outputs: %v\n", ncsi)
			return optional.Some(ncsi)
		})
	mpAuc := getMaterializedParam[uint64, commtypes.ValueTimestampG[*ntypes.Event]]("q3AuctionsBySellerIDStore", h.storeMsgSerde, sp)
	mpPer := getMaterializedParam[uint64, commtypes.ValueTimestampG[*ntypes.Event]]("q3PersonsByIDStore", h.storeMsgSerde, sp)
	return execution.SetupTableTableJoinWithSkipmap(
		mpAuc, mpPer, store.Uint64LessFunc, joiner, exactly_once_intr.GuaranteeMth(sp.GuaranteeMth))
}

func (h *q3JoinTableHandler) Query3JoinTable(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	fn_out := h.setupSerde(serdeFormat)
	if fn_out != nil {
		return fn_out
	}
	srcs, sinks_arr, err := h.getSrcSink(sp)
	if err != nil {
		return common.GenErrFnOutput(fmt.Errorf("getSrcSink err: %v\n", err))
	}
	srcs[0].SetName("auctionsSrc")
	srcs[1].SetName("personsSrc")
	debug.Assert(len(sp.NumOutPartitions) == 1 && len(sp.OutputTopicNames) == 1,
		"expected only one output stream")
	debug.Assert(sp.ScaleEpoch != 0, "scale epoch should start from 1")
	aucJoinsPerFunc, perJoinsAucFunc, kvos, setupSnapFunc, err := h.setupQ3Join(sp)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	toStream := processor.NewTableToStreamProcessorG[uint64, ntypes.NameCityStateId]()

	aJoinP := execution.JoinWorkerFunc[uint64, *ntypes.Event, uint64, ntypes.NameCityStateId](
		func(ctx context.Context, m commtypes.MessageG[uint64, *ntypes.Event]) (
			[]commtypes.MessageG[uint64, ntypes.NameCityStateId], error,
		) {
			// msg is auction
			ret, err := aucJoinsPerFunc(ctx, m)
			if err != nil {
				return nil, err
			}
			if ret != nil {
				return toStream.ProcessAndReturn(ctx, ret[0])
			}
			return nil, nil
		})
	pJoinA := execution.JoinWorkerFunc[uint64, *ntypes.Event, uint64, ntypes.NameCityStateId](
		func(ctx context.Context, m commtypes.MessageG[uint64, *ntypes.Event]) (
			[]commtypes.MessageG[uint64, ntypes.NameCityStateId], error,
		) {
			// msg is person
			ret, err := perJoinsAucFunc(ctx, m)
			if err != nil {
				return nil, fmt.Errorf("ToTabP err: %v", err)
			}
			if ret != nil {
				return toStream.ProcessAndReturn(ctx, ret[0])
			}
			return nil, nil
		})
	msgSerdePair := execution.NewMsgSerdePair(h.inMsgSerde, h.outMsgSerde)
	task, procArgs := execution.PrepareTaskWithJoin(ctx, aJoinP, pJoinA,
		proc_interface.NewBaseSrcsSinks(srcs, sinks_arr),
		proc_interface.NewBaseProcArgs(h.funcName, sp.ScaleEpoch, sp.ParNum),
		true, msgSerdePair, msgSerdePair, "subG2")
	builder := streamArgsBuilderForJoin(procArgs, sp)
	builder = execution.StreamArgsSetKVStore(kvos, builder, exactly_once_intr.GuaranteeMth(sp.GuaranteeMth))
	streamTaskArgs, err := builder.
		FixedOutParNum(sp.ParNum).
		Build()
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	return stream_task.ExecuteApp(ctx, task, streamTaskArgs, setupSnapFunc, func() {
		procArgs.OutputRemainingStats()
	})
}
