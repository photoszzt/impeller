package handlers

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"

	"cs.utexas.edu/zjia/faas/types"
)

type dump struct {
	env types.Environment
}

func NewDump(env types.Environment) types.FuncHandler {
	return &dump{
		env: env,
	}
}

func (h *dump) Call(ctx context.Context, input []byte) ([]byte, error) {
	di := &common.DumpStreams{}
	err := json.Unmarshal(input, di)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	output := h.process(ctx, di)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return common.CompressData(encodedOutput), nil
}

func (h *dump) process(ctx context.Context, di *common.DumpStreams) *common.FnOutput {
	serdeFormat := commtypes.SerdeFormat(di.SerdeFormat)
	for _, streamParam := range di.StreamParams {
		keySerde, err := GetSerdeFromString(streamParam.KeySerde, serdeFormat)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		valSerde, err := GetSerdeFromString(streamParam.ValueSerde, serdeFormat)
		if err != nil {
			return common.GenErrFnOutput(err)
		}
		msgSerde, err := commtypes.GetMsgSerde(serdeFormat, keySerde, valSerde)
		if err != nil {
			return common.GenErrFnOutput(err)
		}

		err = common.DumpOutputStream(ctx, common.DumpOutputStreamConfig{
			OutputDir:     di.DumpDir,
			TopicName:     streamParam.TopicName,
			SerdeFormat:   serdeFormat,
			NumPartitions: streamParam.NumPartitions,
			MsgSerde:      msgSerde,
		})
		if err != nil {
			return common.GenErrFnOutput(err)
		}
	}
	return &common.FnOutput{
		Success: true,
	}
}
