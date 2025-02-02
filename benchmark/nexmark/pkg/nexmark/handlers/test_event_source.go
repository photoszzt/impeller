package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/sharedlog_stream"

	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"

	"cs.utexas.edu/zjia/faas/types"
)

type testEventSource struct {
	env      types.Environment
	funcName string
}

func NewTestEventSource(env types.Environment, funcName string) types.FuncHandler {
	return &testEventSource{
		env:      env,
		funcName: funcName,
	}
}

func (h *testEventSource) Call(ctx context.Context, input []byte) ([]byte, error) {
	inputConfig := &common.TestSourceInput{}
	err := json.Unmarshal(input, inputConfig)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.eventGeneration(ctx, inputConfig)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *testEventSource) eventGeneration(ctx context.Context, sp *common.TestSourceInput) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	stream, err := sharedlog_stream.NewShardedSharedLogStream("nexmark_src", 1, serdeFormat,
		sharedlog_stream.SINK_BUFFER_MAX_SIZE)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	eventSerde, err := ntypes.GetEventSerde(serdeFormat)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	msgSerde, err := commtypes.GetMsgSerde(serdeFormat, commtypes.StringSerde{}, eventSerde)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	events_byte, err := os.ReadFile(sp.FileName)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	events := ntypes.Events{}
	err = json.Unmarshal(events_byte, &events)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	useBuf := msgSerde.UsedBufferPool()
	for _, event := range events.EventsArr {
		msg := commtypes.Message{
			Key:   nil,
			Value: &event,
		}
		msgEncoded, b, err := msgSerde.Encode(&msg)
		if err != nil {
			return common.GenErrFnOutput(fmt.Errorf("msg serialization failed: %v", err))
		}
		_, err = stream.Push(ctx, msgEncoded, 0, sharedlog_stream.SingleDataRecordMeta,
			commtypes.EmptyProducerId)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		if useBuf && b != nil {
			*b = msgEncoded
			commtypes.PushBuffer(b)
		}
	}
	return &common.FnOutput{Success: true}
}
