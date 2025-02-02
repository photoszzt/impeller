package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/tests/pkg/tests/test_types"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"

	"cs.utexas.edu/zjia/faas/types"
)

type produceConsumeHandler struct {
	env types.Environment
}

func NewProducerConsumerTestHandler(env types.Environment) types.FuncHandler {
	return &produceConsumeHandler{
		env: env,
	}
}

func (h *produceConsumeHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &test_types.TestInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.tests(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *produceConsumeHandler) tests(ctx context.Context, sp *test_types.TestInput) *common.FnOutput {
	if sp.TestName == "multiProducer2pc" {
		debug.Fprintf(os.Stderr, "testing multi producer 2pc\n")
		h.testMultiProducer2pc(ctx)
		debug.Fprintf(os.Stderr, "done testing multi producer 2pc\n")
	} else if sp.TestName == "singleProducerEpoch" {
		debug.Fprintf(os.Stderr, "testing single producer EPOCH with JSON serde\n")
		h.testSingleProduceConsumeEpoch(ctx, commtypes.JSON, "test2JSON")
		debug.Fprintf(os.Stderr, "testing single producer EPOCH with MSGP serde\n")
		h.testSingleProduceConsumeEpoch(ctx, commtypes.MSGP, "test2MSGP")
	} else if sp.TestName == "multiProducerEpoch" {
		debug.Fprintf(os.Stderr, "testing multi producer EPOCH with JSON serde\n")
		h.testMultiProducerEpoch(ctx, commtypes.JSON, "test1JSON")
		debug.Fprintf(os.Stderr, "testing multi producer EPOCH with MSGP serde\n")
		h.testMultiProducerEpoch(ctx, commtypes.MSGP, "test1MSGP")
	} else {
		return &common.FnOutput{
			Success: false,
			Message: fmt.Sprintf("test not recognized: %s\n", sp.TestName),
		}
	}
	return &common.FnOutput{
		Success: true,
		Message: fmt.Sprintf("%s done", sp.TestName),
	}
}
