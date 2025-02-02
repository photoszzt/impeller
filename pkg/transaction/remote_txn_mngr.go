package transaction

import (
	"context"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/env_config"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/transaction/remote_txn_rpc"
	"sharedlog-stream/pkg/txn_data"
	"sync"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RemoteTxnManagerServer struct {
	remote_txn_rpc.UnimplementedRemoteTxnMngrServer
	RemoteTxnManager
	env types.Environment
}

func (r *RemoteTxnManagerServer) AppendTpPar(ctx context.Context, in *txn_data.TxnMetaMsg) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, commtypes.ENVID{}, r.env)
	err := r.RemoteTxnManager.AppendTpPar(ctx, in)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, err
}

func (s *RemoteTxnManagerServer) AbortTxn(ctx context.Context, in *txn_data.TxnMetaMsg) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, commtypes.ENVID{}, s.env)
	err := s.RemoteTxnManager.AbortTxn(ctx, in)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, err
}

func (s *RemoteTxnManagerServer) AppendConsumedOffset(ctx context.Context, in *remote_txn_rpc.ConsumedOffsets) (*emptypb.Empty, error) {
	ctx = context.WithValue(ctx, commtypes.ENVID{}, s.env)
	err := s.RemoteTxnManager.AppendConsumedOffset(ctx, in)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, err
}

func (s *RemoteTxnManagerServer) CommitTxnAsyncComplete(ctx context.Context, in *txn_data.TxnMetaMsg) (*remote_txn_rpc.CommitReply, error) {
	ctx = context.WithValue(ctx, commtypes.ENVID{}, s.env)
	return s.RemoteTxnManager.CommitTxnAsyncComplete(ctx, in)
}

func (s *RemoteTxnManagerServer) CommitTxn(ctx context.Context, in *txn_data.TxnMetaMsg) (*remote_txn_rpc.CommitReply, error) {
	ctx = context.WithValue(ctx, commtypes.ENVID{}, s.env)
	return s.RemoteTxnManager.CommitTxn(ctx, in)
}

func (s *RemoteTxnManagerServer) Init(ctx context.Context, in *remote_txn_rpc.InitArg) (*remote_txn_rpc.InitReply, error) {
	ctx = context.WithValue(ctx, commtypes.ENVID{}, s.env)
	return s.RemoteTxnManager.Init(ctx, in)
}

func NewRemoteTxnManagerServer(env types.Environment, serdeFormat commtypes.SerdeFormat) *RemoteTxnManagerServer {
	return &RemoteTxnManagerServer{
		RemoteTxnManager: RemoteTxnManager{
			serdeFormat: serdeFormat,
			tm_map:      make(map[string]*TransactionManager),
			prod_id_map: make(map[string]commtypes.ProducerId),
		},
		env: env,
	}
}

type RemoteTxnManager struct {
	mu          sync.Mutex // guard tm_map
	serdeFormat commtypes.SerdeFormat
	tm_map      map[string]*TransactionManager
	prod_id_map map[string]commtypes.ProducerId
}

func NewRemoteTxnManager() *RemoteTxnManager {
	tm := &RemoteTxnManager{
		tm_map:      make(map[string]*TransactionManager),
		prod_id_map: make(map[string]commtypes.ProducerId),
	}
	return tm
}

func (s *RemoteTxnManager) UpdateSerdeFormat(serdeFormat commtypes.SerdeFormat) {
	s.serdeFormat = serdeFormat
}

func (s *RemoteTxnManager) Init(ctx context.Context, in *remote_txn_rpc.InitArg) (*remote_txn_rpc.InitReply, error) {
	// debug.Fprintf(os.Stderr, "handle Init with input: %v\n", in)
	tm, err := NewTransactionManager(ctx, in.TransactionalId, s.serdeFormat)
	if err != nil {
		return nil, err
	}
	initRet, err := tm.InitTransaction(ctx)
	if err != nil {
		return nil, err
	}
	var offsetPairs []*remote_txn_rpc.OffsetPair
	for _, inputTopicInfo := range in.InputStreamInfos {
		inputTopicName := inputTopicInfo.GetTopicName()
		stream, err := sharedlog_stream.NewShardedSharedLogStream(inputTopicName,
			uint8(inputTopicInfo.NumPartition), s.serdeFormat, in.GetBufMaxSize())
		if err != nil {
			return nil, err
		}
		tm.RecordTopicStreams(inputTopicName, stream)
		err = tm.CreateOffsetTopic(inputTopicName,
			uint8(inputTopicInfo.GetNumPartition()), in.GetBufMaxSize())
		if err != nil {
			return nil, err
		}
		if initRet.HasRecentTxnMeta {
			offset, err := GetOffset(ctx, tm, inputTopicInfo.GetTopicName(),
				uint8(in.GetSubstreamNum()))
			if err != nil {
				return nil, err
			}
			offsetPairs = append(offsetPairs, &remote_txn_rpc.OffsetPair{
				TopicName: inputTopicName,
				Offset:    offset,
			})
		}
	}
	for _, outStreamInfo := range in.OutputStreamInfos {
		stream, err := sharedlog_stream.NewShardedSharedLogStream(outStreamInfo.GetTopicName(),
			uint8(outStreamInfo.NumPartition), s.serdeFormat, in.GetBufMaxSize())
		if err != nil {
			return nil, err
		}
		tm.RecordTopicStreams(outStreamInfo.GetTopicName(), stream)
	}
	for _, kvsInfo := range in.KVChangelogInfos {
		stream, err := sharedlog_stream.NewShardedSharedLogStream(kvsInfo.GetTopicName(),
			uint8(kvsInfo.GetNumPartition()), s.serdeFormat, in.GetBufMaxSize())
		if err != nil {
			return nil, err
		}
		tm.RecordTopicStreams(kvsInfo.GetTopicName(), stream)
	}
	for _, wsInfo := range in.WinChangelogInfos {
		stream, err := sharedlog_stream.NewShardedSharedLogStream(wsInfo.GetTopicName(),
			uint8(wsInfo.GetNumPartition()), s.serdeFormat, in.GetBufMaxSize())
		if err != nil {
			return nil, err
		}
		tm.RecordTopicStreams(wsInfo.GetTopicName(), stream)
	}
	s.mu.Lock()
	s.tm_map[in.TransactionalId] = tm
	s.prod_id_map[in.TransactionalId] = commtypes.ProducerId{
		TaskId:    tm.prodId.TaskId,
		TaskEpoch: tm.prodId.TaskEpoch,
	}
	s.mu.Unlock()
	// debug.Fprintf(os.Stderr, "[%d] done init %s\n", in.SubstreamNum, in.TransactionalId)
	return &remote_txn_rpc.InitReply{
		ProdId: &commtypes.ProdId{
			TaskId:    tm.prodId.TaskId,
			TaskEpoch: uint32(tm.prodId.TaskEpoch),
		},
		OffsetPairs: offsetPairs,
	}, nil
}

func (s *RemoteTxnManager) AppendTpPar(ctx context.Context, in *txn_data.TxnMetaMsg) error {
	// debug.Fprintf(os.Stderr, "handle AppendTpPar taskId %#x, taskEpoch %#x, transactionalId %v, state %v, tps %v\n",
	// 	in.ProdId.GetTaskId(), in.ProdId.GetTaskEpoch(), in.TransactionalId, in.State, in.TopicPartitions)
	s.mu.Lock()
	prodId := s.prod_id_map[in.TransactionalId]
	if prodId.TaskEpoch != in.ProdId.GetTaskEpoch() || prodId.TaskId != in.ProdId.GetTaskId() {
		s.mu.Unlock()
		log.Error().Uint64("recorded task id", prodId.TaskId).
			Uint32("recorded task epoch", prodId.TaskEpoch).
			Uint64("got task id", in.ProdId.TaskId).
			Uint32("got task epoch", in.ProdId.TaskEpoch).
			Msg("producer id mismatch")
		return common_errors.ErrStaleProducer
	}
	tm := s.tm_map[in.TransactionalId]
	s.mu.Unlock()
	if env_config.ASYNC_SECOND_PHASE && !tm.hasWaitForLastTxn.Load() {
		tBeg := stats.TimerBegin()
		err := tm.bgErrg.Wait()
		if err != nil {
			return err
		}
		tm.waitPrevTxn.AddSample(stats.Elapsed(tBeg).Microseconds())
		tm.bgErrg, tm.bgCtx = errgroup.WithContext(ctx)
	}
	if len(in.TopicPartitions) > 0 {
		txnMeta := txn_data.TxnMetadata{
			State:           txn_data.TransactionState(in.State),
			TopicPartitions: in.TopicPartitions,
		}
		_, err := tm.appendToTransactionLog(ctx, txnMeta, []uint64{tm.txnLogTag})
		// debug.Fprint(os.Stderr, "done AppendTpPar\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *RemoteTxnManager) AbortTxn(ctx context.Context, in *txn_data.TxnMetaMsg) error {
	// debug.Fprintf(os.Stderr, "handle AbortTxn taskId %#x, taskEpoch %#x, transactionalId %v, state %v, tps %v\n",
	// 	in.ProdId.TaskId, in.ProdId.TaskEpoch, in.TransactionalId, in.State, in.TopicPartitions)
	s.mu.Lock()
	prodId := s.prod_id_map[in.TransactionalId]
	if prodId.TaskEpoch != in.ProdId.GetTaskEpoch() || prodId.TaskId != in.ProdId.GetTaskId() {
		s.mu.Unlock()
		log.Error().Uint64("recorded task id", prodId.TaskId).
			Uint32("recorded task epoch", prodId.TaskEpoch).
			Uint64("got task id", in.ProdId.TaskId).
			Uint32("got task epoch", in.ProdId.TaskEpoch).
			Msg("producer id mismatch")
		return common_errors.ErrStaleProducer
	}
	tm := s.tm_map[in.TransactionalId]
	s.mu.Unlock()
	txnMd := txn_data.TxnMetadata{
		State: txn_data.PREPARE_ABORT,
	}
	_, err := tm.appendToTransactionLog(ctx, txnMd, []uint64{tm.txnLogTag})
	if err != nil {
		return err
	}
	err = tm.completeTransaction(ctx, commtypes.ABORT, txn_data.COMPLETE_ABORT, in.TopicPartitions)
	// debug.Fprint(os.Stderr, "done AbortTxn\n")
	if err != nil {
		return err
	}
	return nil
}

func (s *RemoteTxnManager) AppendConsumedOffset(ctx context.Context, in *remote_txn_rpc.ConsumedOffsets) error {
	// debug.Fprintf(os.Stderr, "handle AppendConsumedOffset taskId: %#x, taskEpoch: %#x, transactionalId %v, offsetpairs %v, par_num %v\n",
	// 	in.ProdId.TaskId, in.ProdId.TaskEpoch, in.TransactionalId, in.OffsetPairs, in.ParNum)
	s.mu.Lock()
	prodId := s.prod_id_map[in.TransactionalId]
	if prodId.TaskEpoch != in.ProdId.GetTaskEpoch() || prodId.TaskId != in.ProdId.GetTaskId() {
		s.mu.Unlock()
		log.Error().Uint64("recorded task id", prodId.TaskId).
			Uint32("recorded task epoch", prodId.TaskEpoch).
			Uint64("got task id", in.ProdId.TaskId).
			Uint32("got task epoch", in.ProdId.TaskEpoch).
			Msg("producer id mismatch")
		return common_errors.ErrStaleProducer
	}
	tm := s.tm_map[in.TransactionalId]
	s.mu.Unlock()
	var tps []*txn_data.TopicPartition
	for _, op := range in.OffsetPairs {
		tps = append(tps, &txn_data.TopicPartition{
			Topic:  op.TopicName,
			ParNum: []byte{uint8(in.ParNum)},
		})
	}
	txnMeta := txn_data.TxnMetadata{
		State:           txn_data.BEGIN,
		TopicPartitions: tps,
	}
	_, err := tm.appendToTransactionLog(ctx, txnMeta, []uint64{tm.txnLogTag})
	if err != nil {
		return err
	}
	ofUseBuf := tm.offsetRecordSerde.UsedBufferPool()
	for _, op := range in.OffsetPairs {
		offsetLog := tm.topicStreams[op.TopicName]
		offsetRecord := txn_data.OffsetRecord{
			Offset: op.Offset,
		}
		encoded, b, err := tm.offsetRecordSerde.Encode(offsetRecord)
		if err != nil {
			if ofUseBuf && b != nil {
				*b = encoded
				commtypes.PushBuffer(b)
			}
			return err
		}
		_, err = offsetLog.Push(ctx, encoded, uint8(in.ParNum), sharedlog_stream.SingleDataRecordMeta,
			tm.prodId)
		if err != nil {
			if ofUseBuf && b != nil {
				*b = encoded
				commtypes.PushBuffer(b)
			}
			return err
		}
		if ofUseBuf && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
	}
	// debug.Fprint(os.Stderr, "done AppendConsumedOffset\n")
	return nil
}

func (s *RemoteTxnManager) CommitTxnAsyncComplete(ctx context.Context, in *txn_data.TxnMetaMsg) (*remote_txn_rpc.CommitReply, error) {
	// debug.Fprintf(os.Stderr, "handle CommitTxnAsyncComplete taskId %#x, taskEpoch %#x, transactionalId %v, state %v, tps %v\n",
	// 	in.ProdId.TaskId, in.ProdId.TaskEpoch, in.TransactionalId, in.State, in.TopicPartitions)
	s.mu.Lock()
	prodId := s.prod_id_map[in.TransactionalId]
	if prodId.TaskEpoch != in.ProdId.GetTaskEpoch() || prodId.TaskId != in.ProdId.GetTaskId() {
		s.mu.Unlock()
		log.Error().Uint64("recorded task id", prodId.TaskId).
			Uint32("recorded task epoch", prodId.TaskEpoch).
			Uint64("got task id", in.ProdId.TaskId).
			Uint32("got task epoch", in.ProdId.TaskEpoch).
			Msg("producer id mismatch")
		return nil, common_errors.ErrStaleProducer
	}
	tm := s.tm_map[in.TransactionalId]
	s.mu.Unlock()
	txnMd := txn_data.TxnMetadata{
		State: txn_data.PREPARE_COMMIT,
	}
	logOff, err := tm.appendToTransactionLog(ctx, txnMd, []uint64{tm.txnLogTag})
	if err != nil {
		return nil, err
	}
	tm.bgErrg.Go(func() error {
		// second phase of the commit
		err = tm.completeTransaction(tm.bgCtx, commtypes.EPOCH_END, txn_data.COMPLETE_COMMIT, in.TopicPartitions)
		if err != nil {
			return err
		}
		tm.hasWaitForLastTxn.Store(true)
		return nil
	})
	// debug.Fprint(os.Stderr, "done CommitTxnAsyncComplete\n")
	return &remote_txn_rpc.CommitReply{
		LogOffset: logOff,
	}, nil
}

func (s *RemoteTxnManager) CommitTxn(ctx context.Context, in *txn_data.TxnMetaMsg) (*remote_txn_rpc.CommitReply, error) {
	s.mu.Lock()
	prodId := s.prod_id_map[in.TransactionalId]
	if prodId.TaskEpoch != in.ProdId.GetTaskEpoch() || prodId.TaskId != in.ProdId.GetTaskId() {
		s.mu.Unlock()
		log.Error().Uint64("recorded task id", prodId.TaskId).
			Uint32("recorded task epoch", prodId.TaskEpoch).
			Uint64("got task id", in.ProdId.TaskId).
			Uint32("got task epoch", in.ProdId.TaskEpoch).
			Msg("producer id mismatch")
		return nil, common_errors.ErrStaleProducer
	}
	tm := s.tm_map[in.TransactionalId]
	s.mu.Unlock()
	txnMd := txn_data.TxnMetadata{
		State: txn_data.PREPARE_COMMIT,
	}
	logOff, err := tm.appendToTransactionLog(ctx, txnMd, []uint64{tm.txnLogTag})
	if err != nil {
		return nil, err
	}
	err = tm.completeTransaction(tm.bgCtx, commtypes.EPOCH_END, txn_data.COMPLETE_COMMIT, in.TopicPartitions)
	if err != nil {
		return nil, err
	}
	return &remote_txn_rpc.CommitReply{
		LogOffset: logOff,
	}, nil
}
