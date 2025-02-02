package handlers

import (
	"context"
	"encoding/json"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"

	"cs.utexas.edu/zjia/faas/types"
)

type lastQueryEmpty struct {
	env      types.Environment
	funcName string
}

func NewLastEmptyQuery(env types.Environment, funcName string) types.FuncHandler {
	return &lastQueryEmpty{
		env:      env,
		funcName: funcName,
	}
}

func (q *lastQueryEmpty) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, q.env)
	output := q.queryEmpty(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (q *lastQueryEmpty) queryEmpty(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	return emptyFunc(ctx, sp, q.funcName, q.env, "subGLast", false, true)
}

type subG2Empty struct {
	env      types.Environment
	funcName string
}

func NewSubG2Empty(env types.Environment, funcName string) types.FuncHandler {
	return &subG2Empty{
		env:      env,
		funcName: funcName,
	}
}

func (q *subG2Empty) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, q.env)
	output := q.queryEmpty(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (q *subG2Empty) queryEmpty(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	return emptyFunc(ctx, sp, q.funcName, q.env, "subG2", false, false)
}

type subG3Empty struct {
	env      types.Environment
	funcName string
}

func NewSubG3Empty(env types.Environment, funcName string) types.FuncHandler {
	return &subG3Empty{
		env:      env,
		funcName: funcName,
	}
}

func (q *subG3Empty) Call(ctx context.Context, input []byte) ([]byte, error) {
	parsedInput := &common.QueryInput{}
	err := json.Unmarshal(input, parsedInput)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, q.env)
	output := q.queryEmpty(ctx, parsedInput)
	encodedOutput, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	return common.CompressData(encodedOutput), nil
}

func (q *subG3Empty) queryEmpty(ctx context.Context, sp *common.QueryInput) *common.FnOutput {
	return emptyFunc(ctx, sp, q.funcName, q.env, "subG3", false, false)
}
