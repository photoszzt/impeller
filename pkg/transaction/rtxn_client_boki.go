package transaction

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"

	"github.com/zhangyunhao116/skipmap"
	"github.com/zhangyunhao116/skipset"
)

type RemoteTxnMngrClientBoki struct {
	remote_txn_rpc.RTxnRpcClient
	RemoteTxnManagerClient
}

func NewRemoteTxnMngrClientBoki(faas_gateway string, nodeConstraint string,
	serdeFormat commtypes.SerdeFormat, transactionalId string,
) *RemoteTxnMngrClientBoki {
	c := &RemoteTxnMngrClientBoki{
		RTxnRpcClient: remote_txn_rpc.NewRTxnRpcClient(faas_gateway, nodeConstraint, serdeFormat),
		RemoteTxnManagerClient: RemoteTxnManagerClient{
			currentTopicSubstream: skipmap.NewString[*skipset.Uint32Set](),
			TransactionalId:       transactionalId,
			waitAndappendTxnMeta:  stats.NewPrintLogStatsCollector[int64]("waitAndappendTxnMeta"),
		},
	}
	c.addedNewTpPar.Store(false)
	return c
}

func (tc *RemoteTxnMngrClientBoki) AppendConsumedSeqNum(ctx context.Context, consumers []*producer_consumer.MeteredConsumer, parNum uint8) error {
	arg := tc.prepareAppendConsumedSeqNum(consumers, parNum)
	return tc.AppendConsumedOffset(ctx, arg)
}

func (tc *RemoteTxnMngrClientBoki) EnsurePrevTxnFinAndAppendMeta(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	sendRequest := false
	tBeg := stats.TimerBegin()
	txnMeta := tc.prepareEnsure(&sendRequest)
	if sendRequest {
		err := tc.AppendTpPar(ctx, txnMeta)
		if err != nil {
			return err
		}
	}
	tc.waitAndappendTxnMeta.AddSample(stats.Elapsed(tBeg).Microseconds())
	return nil
}

func (tc *RemoteTxnMngrClientBoki) AbortTransaction(ctx context.Context) error {
	txnMeta := tc.prepareAbortOrCommit()
	err := tc.AbortTxn(ctx, txnMeta)
	return err
}

func (tc *RemoteTxnMngrClientBoki) CommitTransactionAsyncComplete(ctx context.Context) (uint64, error) {
	txnMeta := tc.prepareAbortOrCommit()
	ret, err := tc.CommitTxnAsyncComplete(ctx, txnMeta)
	if err != nil {
		return 0, err
	}
	return ret.LogOffset, err
}
