package transaction

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/consume_seq_num_manager/con_types"
	"sharedlog-stream/pkg/env_config"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"
	"sharedlog-stream/pkg/txn_data"
	"sync"
	"sync/atomic"

	"github.com/zhangyunhao116/skipmap"
	"github.com/zhangyunhao116/skipset"
	"google.golang.org/grpc"
)

type RemoteTxnMngrClientGrpc struct {
	remote_txn_rpc.RemoteTxnMngrClient
	RemoteTxnManagerClient
}

type RemoteTxnManagerClient struct {
	mu                    sync.Mutex // protect the flush callback
	prodId                commtypes.ProducerId
	currentTopicSubstream *skipmap.StringMap[*skipset.Uint32Set]
	TransactionalId       string
	addedNewTpPar         atomic.Bool
	waitAndappendTxnMeta  stats.PrintLogStatsCollector[int64]
}

func NewRemoteTxnMngrClientGrpc(cc grpc.ClientConnInterface, transactionalId string) *RemoteTxnMngrClientGrpc {
	c := &RemoteTxnMngrClientGrpc{
		RemoteTxnMngrClient: remote_txn_rpc.NewRemoteTxnMngrClient(cc),
		RemoteTxnManagerClient: RemoteTxnManagerClient{
			currentTopicSubstream: skipmap.NewString[*skipset.Uint32Set](),
			TransactionalId:       transactionalId,
			waitAndappendTxnMeta:  stats.NewPrintLogStatsCollector[int64]("waitAndappendTxnMeta"),
		},
	}
	c.addedNewTpPar.Store(false)
	return c
}

var (
	_ = exactly_once_intr.ReadOnlyExactlyOnceManager(&RemoteTxnManagerClient{})
	_ = exactly_once_intr.ReadOnlyExactlyOnceManager(&RemoteTxnMngrClientBoki{})
	_ = exactly_once_intr.ReadOnlyExactlyOnceManager(&RemoteTxnMngrClientGrpc{})
)

func (tm *RemoteTxnManagerClient) GetCurrentEpoch() uint32             { return tm.prodId.TaskEpoch }
func (tm *RemoteTxnManagerClient) GetCurrentTaskId() uint64            { return tm.prodId.TaskId }
func (tm *RemoteTxnManagerClient) GetProducerId() commtypes.ProducerId { return tm.prodId }
func (tm *RemoteTxnManagerClient) UpdateProducerId(prodId *commtypes.ProdId) {
	tm.prodId.TaskId = prodId.TaskId
	tm.prodId.TaskEpoch = prodId.TaskEpoch
}

func (tm *RemoteTxnManagerClient) OutputRemainingStats() {
	tm.waitAndappendTxnMeta.PrintRemainingStats()
}

func (tc *RemoteTxnManagerClient) collectTopicSubstreams() []*txn_data.TopicPartition {
	topicSubstreams := make([]*txn_data.TopicPartition, 0, tc.currentTopicSubstream.Len())
	tc.currentTopicSubstream.Range(func(tp string, parSet *skipset.Uint32Set) bool {
		tpPar := &txn_data.TopicPartition{
			Topic:  tp,
			ParNum: make([]byte, 0, parSet.Len()),
		}
		parSet.Range(func(par uint32) bool {
			tpPar.ParNum = append(tpPar.ParNum, uint8(par))
			return true
		})
		topicSubstreams = append(topicSubstreams, tpPar)
		return true
	})
	return topicSubstreams
}

func (tc *RemoteTxnManagerClient) AddTopicSubstream(topic string, subStreamNum uint8) {
	// debug.Fprintf(os.Stderr, "tracking topic %s par %v\n", topic, partitions)
	parSet, loaded := tc.currentTopicSubstream.LoadOrStore(topic, skipset.NewUint32())
	needToAppendToLog := !loaded
	hasPar := parSet.Contains(uint32(subStreamNum))
	if !hasPar {
		needToAppendToLog = true
		parSet.Add(uint32(subStreamNum))
	}
	if needToAppendToLog {
		tc.addedNewTpPar.Store(true)
	}
}

func (tc *RemoteTxnManagerClient) AddTopicTrackConsumedSeqs(topicToTrack string, partition uint8) {
	offsetTopic := con_types.OffsetTopic(topicToTrack)
	tc.AddTopicSubstream(offsetTopic, partition)
}

func (tc *RemoteTxnManagerClient) prepareAppendConsumedSeqNum(consumers []*producer_consumer.MeteredConsumer, parNum uint8) *remote_txn_rpc.ConsumedOffsets {
	arg := remote_txn_rpc.ConsumedOffsets{
		TransactionalId: tc.TransactionalId,
		ProdId: &commtypes.ProdId{
			TaskEpoch: tc.prodId.TaskEpoch,
			TaskId:    tc.prodId.TaskId,
		},
		ParNum:      uint32(parNum),
		OffsetPairs: make([]*remote_txn_rpc.OffsetPair, 0, len(consumers)),
	}
	for _, consumer := range consumers {
		topic := consumer.TopicName()
		offset := consumer.CurrentConsumedSeqNum()
		offsetTopic := con_types.OffsetTopic(topic)
		arg.OffsetPairs = append(arg.OffsetPairs, &remote_txn_rpc.OffsetPair{
			TopicName: offsetTopic,
			Offset:    offset,
		})
	}
	return &arg
}

func (tc *RemoteTxnMngrClientGrpc) AppendConsumedSeqNum(ctx context.Context, consumers []*producer_consumer.MeteredConsumer, parNum uint8) error {
	arg := tc.prepareAppendConsumedSeqNum(consumers, parNum)
	_, err := tc.AppendConsumedOffset(ctx, arg)
	return err
}

func (tc *RemoteTxnManagerClient) prepareEnsure(sendRequest *bool) *txn_data.TxnMetaMsg {
	txnMeta := txn_data.TxnMetaMsg{
		TransactionalId: tc.TransactionalId,
		ProdId: &commtypes.ProdId{
			TaskEpoch: tc.prodId.TaskEpoch,
			TaskId:    tc.prodId.TaskId,
		},
		State: uint32(txn_data.BEGIN),
	}
	if env_config.ASYNC_SECOND_PHASE {
		*sendRequest = true
	}
	if tc.addedNewTpPar.Load() {
		*sendRequest = true
		tps := make([]*txn_data.TopicPartition, 0, tc.currentTopicSubstream.Len())
		tc.currentTopicSubstream.Range(func(key string, value *skipset.Uint32Set) bool {
			pars := make([]byte, 0, value.Len())
			value.Range(func(par uint32) bool {
				pars = append(pars, uint8(par))
				return true
			})
			tps = append(tps, &txn_data.TopicPartition{
				Topic:  key,
				ParNum: pars,
			})
			return true
		})
		txnMeta.TopicPartitions = tps
	}
	return &txnMeta
}

func (tc *RemoteTxnMngrClientGrpc) EnsurePrevTxnFinAndAppendMeta(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	sendRequest := false
	tBeg := stats.TimerBegin()
	txnMeta := tc.prepareEnsure(&sendRequest)
	if sendRequest {
		_, err := tc.AppendTpPar(ctx, txnMeta)
		if err != nil {
			return err
		}
	}
	tc.waitAndappendTxnMeta.AddSample(stats.Elapsed(tBeg).Microseconds())
	return nil
}

func (tc *RemoteTxnManagerClient) cleanupState() {
	// debug.Fprintf(os.Stderr, "Transition to %s\n", tc.currentStatus)
	tc.currentTopicSubstream = skipmap.NewString[*skipset.Uint32Set]()
	tc.addedNewTpPar.Store(false)
}

func (tc *RemoteTxnManagerClient) prepareAbortOrCommit() *txn_data.TxnMetaMsg {
	tps := tc.collectTopicSubstreams()
	tc.cleanupState()
	txnMeta := txn_data.TxnMetaMsg{
		TransactionalId: tc.TransactionalId,
		ProdId: &commtypes.ProdId{
			TaskEpoch: tc.prodId.TaskEpoch,
			TaskId:    tc.prodId.TaskId,
		},
		TopicPartitions: tps,
	}
	return &txnMeta
}

func (tc *RemoteTxnMngrClientGrpc) AbortTransaction(ctx context.Context) error {
	txnMeta := tc.prepareAbortOrCommit()
	_, err := tc.AbortTxn(ctx, txnMeta)
	return err
}

func (tc *RemoteTxnMngrClientGrpc) CommitTransactionAsyncComplete(ctx context.Context) (uint64, error) {
	txnMeta := tc.prepareAbortOrCommit()
	ret, err := tc.CommitTxnAsyncComplete(ctx, txnMeta)
	if err != nil {
		return 0, err
	}
	return ret.LogOffset, err
}

func (tc *RemoteTxnMngrClientGrpc) CommitTransaction(ctx context.Context) (uint64, error) {
	txnMeta := tc.prepareAbortOrCommit()
	ret, err := tc.CommitTxn(ctx, txnMeta)
	if err != nil {
		return 0, err
	}
	return ret.LogOffset, err
}
