//go:generate msgp
package remote_txn_rpc

import (
	"sharedlog-stream/pkg/commtypes"
	txn_data "sharedlog-stream/pkg/txn_data"
	"slices"
)

type RTxnRpcType uint8

const (
	Init RTxnRpcType = iota
	AppendTpPar
	AbortTxn
	CommitTxnAsync
	AppendConsumedOff
)

type RTxnArg struct {
	RpcType     RTxnRpcType          `json:"rpcType,omitempty" msg:"rpcType,omitempty"`
	SerdeFormat uint8                `json:"serdeFormat,omitempty" msg:"serdeFormat,omitempty"`
	Init        *InitArg             `json:"initArg,omitempty" msg:"initArg,omitempty"`
	MetaMsg     *txn_data.TxnMetaMsg `json:"txnMeta,omitempty" msg:"txnMeta,omitempty"`
	ConsumedOff *ConsumedOffsets     `json:"cOff,omitempty" msg:"cOff,omitempty"`
}

type RTxnReply struct {
	Success     bool         `json:"succ,omitempty" msg:"succ,omitempty"`
	Message     string       `json:"msg,omitempty" msg:"msg,omitempty"`
	InitReply   *InitReply   `json:"initRply,omitempty" msg:"initRply,omitempty"`
	CommitReply *CommitReply `json:"cmtRply,omitempty" msg:"cmtRply,omitempty"`
}

func EqualStreamInfos(a, b *StreamInfo) bool {
	if a != nil && b != nil {
		return a.TopicName == b.TopicName && a.NumPartition == b.NumPartition
	} else {
		return a == b
	}
}

func EqualOffsetPair(a, b *OffsetPair) bool {
	if a != nil && b != nil {
		return a.TopicName == b.TopicName && a.Offset == b.Offset
	} else {
		return a == b
	}
}

func (a *InitReply) Equal(b *InitReply) bool {
	return commtypes.EqualProdId(a.ProdId, b.ProdId) &&
		slices.EqualFunc(a.OffsetPairs, b.OffsetPairs, EqualOffsetPair)
}

func (a *CommitReply) Equal(b *CommitReply) bool {
	return a.LogOffset == b.LogOffset
}

func (a *RTxnArg) Equal(b *RTxnArg) bool {
	ret := a.RpcType == b.RpcType && a.SerdeFormat == b.SerdeFormat
	if ret {
		if a.Init != nil {
			ret = ret && a.Init.Equal(b.Init)
		} else {
			ret = ret && (a.Init == b.Init)
		}
		if a.MetaMsg != nil {
			ret = ret && a.MetaMsg.Equal(b.MetaMsg)
		} else {
			ret = ret && (a.MetaMsg == b.MetaMsg)
		}
		if a.ConsumedOff != nil {
			ret = ret && a.ConsumedOff.Equal(b.ConsumedOff)
		} else {
			ret = ret && (a.ConsumedOff == b.ConsumedOff)
		}
	}
	return ret
}

func (a *RTxnReply) Equal(b *RTxnReply) bool {
	ret := a.Success == b.Success && a.Message == b.Message
	if ret {
		if a.InitReply != nil {
			ret = ret && a.InitReply.Equal(b.InitReply)
		} else {
			ret = ret && (a.InitReply == b.InitReply)
		}
		if a.CommitReply != nil {
			ret = ret && a.CommitReply.Equal(b.CommitReply)
		} else {
			ret = ret && (a.CommitReply == b.CommitReply)
		}
	}
	return ret
}

func (a *ConsumedOffsets) Equal(b *ConsumedOffsets) bool {
	return a.TransactionalId == b.TransactionalId &&
		commtypes.EqualProdId(a.ProdId, b.ProdId) &&
		a.ParNum == b.ParNum &&
		slices.EqualFunc(a.OffsetPairs, b.OffsetPairs, EqualOffsetPair)
}

func (a *InitArg) Equal(b *InitArg) bool {
	return a.TransactionalId == b.TransactionalId &&
		a.SubstreamNum == b.SubstreamNum &&
		a.BufMaxSize == b.BufMaxSize &&
		slices.EqualFunc(a.InputStreamInfos, b.InputStreamInfos, EqualStreamInfos) &&
		slices.EqualFunc(a.OutputStreamInfos, b.OutputStreamInfos, EqualStreamInfos) &&
		slices.EqualFunc(a.KVChangelogInfos, b.KVChangelogInfos, EqualStreamInfos) &&
		slices.EqualFunc(a.WinChangelogInfos, b.WinChangelogInfos, EqualStreamInfos)
}
