package configscale

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/control_channel"

	"cs.utexas.edu/zjia/faas/types"
)

type ConfigScaleHandler struct {
	env types.Environment
}

func NewConfigScaleHandler(env types.Environment) types.FuncHandler {
	return &ConfigScaleHandler{
		env: env,
	}
}

func (h *ConfigScaleHandler) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.ConfigScaleInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.ConfigScale(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (h *ConfigScaleHandler) ConfigScale(ctx context.Context, input *common.ConfigScaleInput) *common.FnOutput {
	cmm, err := control_channel.NewControlChannelManager(input.AppId,
		commtypes.SerdeFormat(input.SerdeFormat), input.BufMaxSize, input.ScaleEpoch-1, 0)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	// append rescale config will add one to epoch; so subtract one when set up the cmm
	err = cmm.AppendRescaleConfig(ctx, input.Config)
	if err != nil {
		return common.GenErrFnOutput(err)
	}
	if input.Bootstrap {
		for _, fn := range input.FuncNames {
			ins := input.Config[fn]
			for i := uint8(0); i < ins; i++ {
				err = cmm.RecordPrevInstanceFinish(ctx, fn, i, 0)
				if err != nil {
					return common.GenErrFnOutput(err)
				}
			}
		}
	}
	return &common.FnOutput{Success: true}
}
