package main

import (
	"context"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/transaction"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"

	"github.com/rs/zerolog/log"

	"cs.utexas.edu/zjia/faas"
	"cs.utexas.edu/zjia/faas/types"
)

type mngrFuncHandlerFactory struct{}

var r = transaction.NewRemoteTxnManager()

type mngrFuncHanlder struct {
	replySerde commtypes.SerdeG[*remote_txn_rpc.RTxnReply]
	env        types.Environment
}

func (h *mngrFuncHanlder) EncodeReply(reply *remote_txn_rpc.RTxnReply) []byte {
	ret, _, err := h.replySerde.Encode(reply)
	if err != nil {
		log.Error().Err(err).Msg("replySerde encode fail")
	}
	return ret
}

func (h *mngrFuncHanlder) GenErrOut(err error) []byte {
	reply := &remote_txn_rpc.RTxnReply{
		Success: false,
		Message: err.Error(),
	}
	ret, _, err := h.replySerde.Encode(reply)
	if err != nil {
		log.Error().Err(err).Msg("replySerde encode fail")
	}
	return ret
}

func (h *mngrFuncHanlder) GenEmptySucc() []byte {
	reply := &remote_txn_rpc.RTxnReply{
		Success: true,
	}
	ret, _, err := h.replySerde.Encode(reply)
	if err != nil {
		log.Error().Err(err).Msg("replySerde encode fail")
	}
	return ret
}

func (h *mngrFuncHanlder) Call(ctx context.Context, input []byte) ([]byte, error) {
	serde := remote_txn_rpc.RTxnArgMsgpSerdeG{}
	in, err := serde.Decode(input)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, commtypes.ENVID{}, h.env)
	switch in.RpcType {
	case remote_txn_rpc.Init:
		r.UpdateSerdeFormat(commtypes.SerdeFormat(in.SerdeFormat))
		ret, err := r.Init(ctx, in.Init)
		if err != nil {
			return h.GenErrOut(err), nil
		}
		reply := &remote_txn_rpc.RTxnReply{
			Success:   true,
			InitReply: ret,
		}
		return h.EncodeReply(reply), nil
	case remote_txn_rpc.CommitTxnAsync:
		ret, err := r.CommitTxnAsyncComplete(ctx, in.MetaMsg)
		if err != nil {
			return h.GenErrOut(err), nil
		}
		reply := &remote_txn_rpc.RTxnReply{
			Success:     true,
			CommitReply: ret,
		}
		return h.EncodeReply(reply), nil
	case remote_txn_rpc.AppendTpPar:
		err := r.AppendTpPar(ctx, in.MetaMsg)
		if err != nil {
			return h.GenErrOut(err), nil
		}
		return h.GenEmptySucc(), nil
	case remote_txn_rpc.AppendConsumedOff:
		err := r.AppendConsumedOffset(ctx, in.ConsumedOff)
		if err != nil {
			return h.GenErrOut(err), nil
		}
		return h.GenEmptySucc(), nil
	case remote_txn_rpc.AbortTxn:
		err := r.AbortTxn(ctx, in.MetaMsg)
		if err != nil {
			return h.GenErrOut(err), nil
		}
		return h.GenEmptySucc(), nil
	default:
		return nil, common_errors.ErrInvalidTxnMngrRpc
	}
}

func (f *mngrFuncHandlerFactory) New(env types.Environment, funcName string) (types.FuncHandler, error) {
	s, _ := remote_txn_rpc.GetRTxnReplySerdeG(commtypes.MSGP)
	return &mngrFuncHanlder{
		env:        env,
		replySerde: s,
	}, nil
}

func (f *mngrFuncHandlerFactory) GrpcNew(env types.Environment, service string) (types.GrpcFuncHandler, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	factory := &mngrFuncHandlerFactory{}
	faas.Serve(factory)
}
