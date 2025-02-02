package redis_setup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/checkpt"

	"cs.utexas.edu/zjia/faas/types"
)

type RedisSetup struct {
	env types.Environment
}

func NewRedisSetupHandler(env types.Environment) types.FuncHandler {
	return &RedisSetup{
		env: env,
	}
}

func (h *RedisSetup) Setup(ctx context.Context, input *common.RedisSetupInput) *common.FnOutput {
	rcm := checkpt.NewRedisChkptManager()
	err := rcm.InitReqRes(ctx)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	err = rcm.ResetCheckPointCount(ctx, input.FinalOutputTopicNames)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	// mcs, err := snapshot_store.NewMinioChkptStore()
	// if err != nil {
	// 	return common.GenErrFnOutput(err)
	// }
	// err = mcs.CreateWorkloadBucket(ctx)
	// if err != nil {
	// 	return common.GenErrFnOutput(err)
	// }
	return &common.FnOutput{Success: true}
}

func (h *RedisSetup) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.RedisSetupInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}
	fmt.Fprintf(os.Stderr, "RedisSetupInput:\n")
	fmt.Fprintf(os.Stderr, "\tFinalOutputTopicNames: %v\n", parsedInput.FinalOutputTopicNames)
	output := h.Setup(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}
