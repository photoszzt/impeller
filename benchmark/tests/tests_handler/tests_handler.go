package main

import (
	"fmt"
	"sharedlog-stream/benchmark/tests/pkg/tests/handlers"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type funcHandlerFactory struct{}

func (f *funcHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	switch funcName {
	case "wintest":
		return handlers.NewWinTabTestsHandler(env), nil
		// case "restore":
		//	return handlers.NewTableRestoreHandler(env), nil
	case "join":
		return handlers.NewJoinHandler(env), nil
	case "prodConsume":
		return handlers.NewProducerConsumerTestHandler(env), nil
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
