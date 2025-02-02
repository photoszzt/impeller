package main

import (
	"fmt"
	configscale "sharedlog-stream/benchmark/common/config_scale"
	"sharedlog-stream/benchmark/lat_tp/pkg/handlers"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type funcHandlerFactory struct{}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	switch funcName {
	case "produce":
		return handlers.NewSharedlogProduceBenchHandler(env), nil
	case "consume":
		return handlers.NewSharedlogConsumeBenchHandler(env), nil
	case "tranDataGen":
		return handlers.NewSharedlogTranDataGenHandler(env), nil
	case "tranProcess":
		return handlers.NewSharedlogTranProcessHandler(env), nil
	case "scale":
		return configscale.NewConfigScaleHandler(env), nil
	default:
		return nil, fmt.Errorf("unknown function name %v", funcName)
	}
}

func (f *funcHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	faas.Serve(&funcHandlerFactory{})
}
