package transaction

import (
	"context"
	"fmt"
	"math"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/consume_seq_num_manager/con_types"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/env_config"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/txn_data"
	"sync"
	"sync/atomic"

	"cs.utexas.edu/zjia/faas/protocol"
	"cs.utexas.edu/zjia/faas/types"
	"github.com/rs/zerolog/log"
	"github.com/zhangyunhao116/skipmap"
	"github.com/zhangyunhao116/skipset"
	"golang.org/x/sync/errgroup"
)

const (
	TRANSACTION_LOG_TOPIC_NAME = "__transaction_log"
)

type WaitOrAppendedMeta uint8

const (
	None = iota
	WaitPrev
	AppendedNewPar
	WaitAndAppend
)

// assume each transactional_id correspond to one output partition
// transaction manager is not goroutine safe, it's assumed to be used by
// only one stream task and only one goroutine could update it
type TransactionManager struct {
	mu                    sync.Mutex // protect flush callback
	offsetRecordSerde     commtypes.SerdeG[txn_data.OffsetRecord]
	bgCtx                 context.Context
	txnMdSerde            commtypes.SerdeG[txn_data.TxnMetadata]
	topicPartitionSerde   commtypes.SerdeG[*txn_data.TopicPartition]
	txnMarkerSerde        commtypes.SerdeG[commtypes.EpochMarker]
	transactionLog        *sharedlog_stream.SharedLogStream
	bgErrg                *errgroup.Group
	currentTopicSubstream *skipmap.StringMap[*skipset.Uint32Set]
	topicStreams          map[string]*sharedlog_stream.ShardedSharedLogStream
	TransactionalId       string
	prodId                commtypes.ProducerId
	tranCompleteMarkerTag uint64
	txnLogTag             uint64
	txnFenceTag           uint64
	waitPrevTxn           stats.PrintLogStatsCollector[int64]
	appendTxnMeta         stats.PrintLogStatsCollector[int64]
	txnSndPhase           stats.PrintLogStatsCollector[int64]
	hasWaitForLastTxn     atomic.Bool
	addedNewTpPar         atomic.Bool
	serdeFormat           commtypes.SerdeFormat
	txnMdSerdeUseBuf      bool
}

func NewTransactionManager(ctx context.Context,
	transactional_id string,
	serdeFormat commtypes.SerdeFormat,
) (*TransactionManager, error) {
	log, err := sharedlog_stream.NewSharedLogStream(TRANSACTION_LOG_TOPIC_NAME+"_"+transactional_id, serdeFormat)
	if err != nil {
		return nil, err
	}
	tm := &TransactionManager{
		transactionLog:        log,
		TransactionalId:       transactional_id,
		currentTopicSubstream: skipmap.NewString[*skipset.Uint32Set](),
		topicStreams:          make(map[string]*sharedlog_stream.ShardedSharedLogStream),
		prodId:                commtypes.NewProducerId(),
		tranCompleteMarkerTag: txn_data.MarkerTag(log.TopicNameHash(), 0),
		txnLogTag:             sharedlog_stream.NameHashWithPartition(log.TopicNameHash(), 0),
		txnFenceTag:           txn_data.FenceTag(log.TopicNameHash(), 0),
		serdeFormat:           serdeFormat,
		waitPrevTxn:           stats.NewPrintLogStatsCollector[int64]("waitPrevTxn2pc"),
		appendTxnMeta:         stats.NewPrintLogStatsCollector[int64]("appendTxnMeta2pc"),
		txnSndPhase:           stats.NewPrintLogStatsCollector[int64]("txnSndPhase"),
	}
	tm.addedNewTpPar.Store(false)
	tm.hasWaitForLastTxn.Store(false)
	tm.bgErrg, tm.bgCtx = errgroup.WithContext(ctx)
	err = tm.setupSerde(serdeFormat)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "%s txnMdSerde %s, topicPartitionSerde %s, txnMarkerSerde %s, offsetRecordSerde %s\n",
		tm.TransactionalId, tm.txnMdSerde, tm.topicPartitionSerde, tm.txnMarkerSerde, tm.offsetRecordSerde)
	tm.txnMdSerdeUseBuf = tm.txnMdSerde.UsedBufferPool()
	return tm, nil
}

func (tm *TransactionManager) OutputRemainingStats() {
	tm.waitPrevTxn.PrintRemainingStats()
	tm.appendTxnMeta.PrintRemainingStats()
	tm.txnSndPhase.PrintRemainingStats()
}

func (tm *TransactionManager) GetCurrentEpoch() uint32             { return tm.prodId.TaskEpoch }
func (tm *TransactionManager) GetCurrentTaskId() uint64            { return tm.prodId.TaskId }
func (tm *TransactionManager) GetProducerId() commtypes.ProducerId { return tm.prodId }

func (tc *TransactionManager) setupSerde(serdeFormat commtypes.SerdeFormat) error {
	if serdeFormat == commtypes.JSON {
		tc.txnMdSerde = txn_data.TxnMetadataJSONSerdeG{}
		tc.topicPartitionSerde = txn_data.TopicPartitionJSONSerdeG{}
		tc.txnMarkerSerde = commtypes.EpochMarkerJSONSerdeG{}
		tc.offsetRecordSerde = txn_data.OffsetRecordJSONSerdeG{}
	} else if serdeFormat == commtypes.MSGP {
		tc.txnMdSerde = txn_data.TxnMetadataMsgpSerdeG{}
		tc.topicPartitionSerde = txn_data.TopicPartitionMsgpSerdeG{}
		tc.txnMarkerSerde = commtypes.EpochMarkerMsgpSerdeG{}
		tc.offsetRecordSerde = txn_data.OffsetRecordMsgpSerdeG{}
	} else {
		return fmt.Errorf("serde format should be either json or msgp; but %v is given", serdeFormat)
	}
	return nil
}

func (tc *TransactionManager) loadCurrentTopicPartitions(lastTopicPartitions []*txn_data.TopicPartition) {
	for _, tp := range lastTopicPartitions {
		pars, _ := tc.currentTopicSubstream.LoadOrStore(tp.Topic, skipset.NewUint32())
		for _, par := range tp.ParNum {
			pars.Add(uint32(par))
		}
	}
}

func (tc *TransactionManager) getMostRecentTransactionState(ctx context.Context) (mostRecentTxnMetadata *txn_data.TxnMetadata, rawMsg *commtypes.RawMsg, err error) {
	// debug.Fprintf(os.Stderr, "load transaction log\n")
	mostRecentTxnMetadata = &txn_data.TxnMetadata{
		TopicPartitions: make([]*txn_data.TopicPartition, 0),
		State:           txn_data.EMPTY,
	}

	// debug.Fprintf(os.Stderr, "getMostRecentTransactionState: ReadBackwardWithTag 1\n")
	// find the most recent completed transaction
	rawMsg, err = tc.transactionLog.ReadBackwardWithTag(ctx, protocol.MaxLogSeqnum, 0, tc.tranCompleteMarkerTag)
	if err != nil {
		if common_errors.IsStreamEmptyError(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	begin := uint64(0)
	// try to find one before the last one
	// debug.Fprintf(os.Stderr, "getMostRecentTransactionState: ReadBackwardWithTag 2\n")
	rawMsg2, err := tc.transactionLog.ReadBackwardWithTag(ctx, rawMsg.LogSeqNum, 0, tc.tranCompleteMarkerTag)
	if err != nil {
		if !common_errors.IsStreamEmptyError(err) {
			return nil, nil, err
		}
	} else {
		begin = rawMsg2.LogSeqNum
	}
	tc.transactionLog.SetCursor(begin+1, 0)

	for {
		// debug.Fprintf(os.Stderr, "getMostRecentTransactionState: ReadNext\n")
		msg, err := tc.transactionLog.ReadNext(ctx, 0)
		if common_errors.IsStreamEmptyError(err) {
			break
		} else if err != nil {
			return nil, nil, err
		}
		txnMeta, err := tc.txnMdSerde.Decode(msg.Payload)
		if err != nil {
			return nil, nil, err
		}

		if txnMeta.TopicPartitions != nil {
			mostRecentTxnMetadata.TopicPartitions = append(mostRecentTxnMetadata.TopicPartitions, txnMeta.TopicPartitions...)
			mostRecentTxnMetadata.State = txnMeta.State
		} else {
			mostRecentTxnMetadata.State = txnMeta.State
		}
	}
	return mostRecentTxnMetadata, rawMsg, nil
}

// each transaction id corresponds to a separate transaction log; we only have one transaction id per serverless function
func (tc *TransactionManager) loadAndFixTransaction(ctx context.Context, mostRecentTxnMetadata *txn_data.TxnMetadata) error {
	// check the last status of the transaction
	switch mostRecentTxnMetadata.State {
	case txn_data.EMPTY, txn_data.COMPLETE_COMMIT, txn_data.COMPLETE_ABORT:
		log.Info().Msgf("examed previous transaction with no error")
	case txn_data.BEGIN:
		// need to abort
		tc.loadCurrentTopicPartitions(mostRecentTxnMetadata.TopicPartitions)
		err := tc.AbortTransaction(ctx)
		if err != nil {
			return err
		}
	case txn_data.PREPARE_ABORT:
		// need to abort

		// the transaction is aborted but the marker might not pushed to the relevant partitions yet
		tc.loadCurrentTopicPartitions(mostRecentTxnMetadata.TopicPartitions)
		tps := tc.collectTopicSubstreams()
		err := tc.completeTransaction(ctx, commtypes.ABORT, txn_data.COMPLETE_ABORT, tps)
		if err != nil {
			return err
		}
		tc.cleanupState()
	case txn_data.PREPARE_COMMIT:
		// the transaction is commited but the marker might not pushed to the relevant partitions yet
		// need to commit
		// debug.Fprintf(os.Stderr, "In repair: Transition to %s to restore\n", tc.currentStatus)

		tc.loadCurrentTopicPartitions(mostRecentTxnMetadata.TopicPartitions)
		tps := tc.collectTopicSubstreams()
		err := tc.completeTransaction(ctx, commtypes.EPOCH_END, txn_data.COMPLETE_COMMIT, tps)
		if err != nil {
			return err
		}
		tc.cleanupState()

	case txn_data.FENCE:
		log.Info().Msgf("Last operation in the log is fence to update the epoch. We are updating the epoch again.")
	}
	return nil
}

type InitTxnRet struct {
	HasRecentTxnMeta bool
	RecentTxnAuxData []byte
	RecentTxnLogSeq  uint64
}

// call at the beginning of function. Expected to execute in a single thread
func (tc *TransactionManager) InitTransaction(ctx context.Context) (*InitTxnRet, error) {
	recentTxnMeta, recentCompleteTxn, err := tc.getMostRecentTransactionState(ctx)
	if err != nil {
		return nil, fmt.Errorf("getMostRecentTransactionState failed: %v", err)
	}
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	// debug.Fprintf(os.Stderr, "Init transaction: Transition to %s\n", tc.currentStatus)
	if recentTxnMeta == nil {
		tc.prodId.InitTaskId(env)
		tc.prodId.TaskEpoch = 0
	} else {
		tc.prodId = recentCompleteTxn.ProdId
	}
	if recentTxnMeta != nil && recentCompleteTxn.ProdId.TaskEpoch == math.MaxUint32 {
		tc.prodId.InitTaskId(env)
		tc.prodId.TaskEpoch = 0
	}
	tc.prodId.TaskEpoch += 1
	txnMeta := txn_data.TxnMetadata{
		State: txn_data.FENCE,
	}
	tags := []uint64{tc.txnLogTag, tc.txnFenceTag}
	_, err = tc.appendToTransactionLog(ctx, txnMeta, tags)
	if err != nil {
		return nil, fmt.Errorf("appendToTransactionLog failed: %v", err)
	}

	ret := &InitTxnRet{
		HasRecentTxnMeta: recentTxnMeta != nil,
	}

	if recentTxnMeta != nil {
		err = tc.loadAndFixTransaction(ctx, recentTxnMeta)
		if err != nil {
			return nil, fmt.Errorf("loadTransactinoFromLog failed: %v", err)
		}
		ret.RecentTxnLogSeq = recentCompleteTxn.LogSeqNum
		ret.RecentTxnAuxData = recentCompleteTxn.AuxData
	}
	return ret, nil
}

func (tc *TransactionManager) appendToTransactionLog(ctx context.Context,
	tm txn_data.TxnMetadata, tags []uint64,
) (uint64, error) {
	encoded, b, err := tc.txnMdSerde.Encode(tm)
	if err != nil {
		return 0, fmt.Errorf("txnMdSerde enc err: %v", tm)
	}
	prodId := tc.prodId
	var r uint64
	if tags != nil {
		r, err = tc.transactionLog.PushWithTag(ctx, encoded, 0, tags, nil,
			sharedlog_stream.SingleDataRecordMeta, prodId)
	} else {
		r, err = tc.transactionLog.Push(ctx, encoded, 0,
			sharedlog_stream.SingleDataRecordMeta, prodId)
	}
	if tc.txnMdSerdeUseBuf && b != nil {
		*b = encoded
		commtypes.PushBuffer(b)
	}
	return r, err
}

func (tc *TransactionManager) collectTopicSubstreams() []*txn_data.TopicPartition {
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

func (tc *TransactionManager) appendTxnMarkerToStreams(ctx context.Context, marker commtypes.EpochMark,
	topicSubstreams []*txn_data.TopicPartition,
) error {
	tm := commtypes.EpochMarker{
		Mark: marker,
	}
	encoded, b, err := tc.txnMarkerSerde.Encode(tm)
	if err != nil {
		return err
	}
	producerId := tc.prodId
	bg, bgCtx := errgroup.WithContext(ctx)
	for _, tpParNum := range topicSubstreams {
		stream := tc.topicStreams[tpParNum.Topic]
		topicNameHash := stream.TopicNameHash()
		for _, par := range tpParNum.ParNum {
			parNum := uint8(par)
			tag := txn_data.MarkerTag(topicNameHash, parNum)
			tag2 := sharedlog_stream.NameHashWithPartition(topicNameHash, parNum)
			bg.Go(func() error {
				_, err := stream.PushWithTag(bgCtx, encoded, parNum, []uint64{tag, tag2},
					nil, sharedlog_stream.ControlRecordMeta, producerId)
				// debug.Fprintf(os.Stderr, "append marker %#v to stream %s off %x tag %x\n",
				// 	marker, stream.TopicName(), off, tag)
				return err
			})
		}
	}
	err = bg.Wait()
	if tc.txnMarkerSerde.UsedBufferPool() && b != nil {
		*b = encoded
		commtypes.PushBuffer(b)
	}
	return err
}

func (tc *TransactionManager) checkTopicExistsInTopicStream(topic string) bool {
	_, ok := tc.topicStreams[topic]
	return ok
}

// this function could be called by multiple goroutine.
func (tc *TransactionManager) AddTopicSubstream(topic string, subStreamNum uint8) {
	debug.Assert(tc.checkTopicExistsInTopicStream(topic), fmt.Sprintf("topic %s's stream should be tracked", topic))
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

func (tc *TransactionManager) CreateOffsetTopic(topicToTrack string, numPartition uint8, bufMaxSize uint32) error {
	offsetTopic := con_types.OffsetTopic(topicToTrack)
	debug.Assert(tc.topicStreams != nil, "topic streams should be initialized")
	_, ok := tc.topicStreams[offsetTopic]
	if ok {
		// already exists
		return nil
	}
	off, err := sharedlog_stream.NewShardedSharedLogStream(offsetTopic, numPartition, tc.serdeFormat, bufMaxSize)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "adding %s to topic streams\n", offsetTopic)
	tc.topicStreams[offsetTopic] = off
	return nil
}

func (tc *TransactionManager) RecordTopicStreams(topicToTrack string, stream *sharedlog_stream.ShardedSharedLogStream) {
	tc.topicStreams[topicToTrack] = stream
	debug.Fprintf(os.Stderr, "tracking stream %s, stream ptr %v\n", topicToTrack, stream)
}

func (tc *TransactionManager) AddTopicTrackConsumedSeqs(topicToTrack string, partition uint8) {
	offsetTopic := con_types.OffsetTopic(topicToTrack)
	tc.AddTopicSubstream(offsetTopic, partition)
}

// finding the last commited marker and gets the marker's seq number
// used in restore and in one thread
func GetOffset(ctx context.Context, tm *TransactionManager,
	topic string, parNum uint8,
) (uint64, error) {
	offset, err := tm.FindLastConsumedSeqNum(ctx, topic, parNum)
	if err != nil {
		if !common_errors.IsStreamEmptyError(err) {
			return 0, err
		}
	}
	return offset, nil
}

// func CollectOffsetRecords(consumers []*producer_consumer.MeteredConsumer) map[string]txn_data.OffsetRecord {
// 	ret := make(map[string]txn_data.OffsetRecord)
// 	for _, consumer := range consumers {
// 		topic := consumer.TopicName()
// 		offset := consumer.CurrentConsumedSeqNum()
// 		offsetTopic := con_types.OffsetTopic(topic)
// 		offsetRecord := txn_data.OffsetRecord{
// 			Offset: offset,
// 		}
// 		ret[offsetTopic] = offsetRecord
// 	}
// 	return ret
// }

func (tc *TransactionManager) AppendConsumedSeqNum(ctx context.Context, consumers []*producer_consumer.MeteredConsumer, parNum uint8) error {
	tps := make([]*txn_data.TopicPartition, 0, len(consumers))
	for _, consumer := range consumers {
		offsetTopic := con_types.OffsetTopic(consumer.TopicName())
		tps = append(tps, &txn_data.TopicPartition{
			Topic:  offsetTopic,
			ParNum: []byte{parNum},
		})
	}
	txnMeta := txn_data.TxnMetadata{
		State:           txn_data.BEGIN,
		TopicPartitions: tps,
	}
	_, err := tc.appendToTransactionLog(ctx, txnMeta, []uint64{tc.txnLogTag})
	if err != nil {
		return err
	}
	for _, consumer := range consumers {
		topic := consumer.TopicName()
		offset := consumer.CurrentConsumedSeqNum()
		offsetTopic := con_types.OffsetTopic(topic)
		offsetRecord := txn_data.OffsetRecord{
			Offset: offset,
		}
		offsetLog := tc.topicStreams[offsetTopic]

		encoded, b, err := tc.offsetRecordSerde.Encode(offsetRecord)
		if err != nil {
			return err
		}
		_, err = offsetLog.Push(ctx, encoded, parNum, sharedlog_stream.SingleDataRecordMeta,
			tc.prodId)
		if err != nil {
			return err
		}
		if tc.offsetRecordSerde.UsedBufferPool() {
			*b = encoded
			commtypes.PushBuffer(b)
		}
		// debug.Fprintf(os.Stderr, "consumed offset 0x%x for %s\n", offsetRecord.Offset, offsetTopic)
	}
	return nil
}

func (tc *TransactionManager) FindLastConsumedSeqNum(ctx context.Context, topicToTrack string, parNum uint8) (uint64, error) {
	offsetTopic := con_types.OffsetTopic(topicToTrack)
	offsetLog := tc.topicStreams[offsetTopic]
	debug.Assert(offsetTopic == offsetLog.TopicName(), fmt.Sprintf("expected offset log tp: %s, got %s",
		offsetTopic, offsetLog.TopicName()))
	// debug.Fprintf(os.Stderr, "looking at offsetlog %s, offsetLog tp: %s\n", offsetTopic, offsetLog.TopicName())

	// find the most recent transaction marker
	txnMarkerTag := txn_data.MarkerTag(offsetLog.TopicNameHash(), parNum)
	var txnMkRawMsg *commtypes.RawMsg
	var err error
	txnMkRawMsg, err = offsetLog.ReadBackwardWithTag(ctx, protocol.MaxLogSeqnum, parNum, txnMarkerTag)
	if err != nil {
		return 0, err
	}
	debug.Fprintf(os.Stderr, "offlog got entry off %x, control %v\n",
		txnMkRawMsg.LogSeqNum, txnMkRawMsg.IsControl)
	tag := sharedlog_stream.NameHashWithPartition(offsetLog.TopicNameHash(), parNum)
	debug.Fprintf(os.Stderr, "most recent commit seqNumber 0x%x\n", txnMkRawMsg.LogSeqNum)

	// read the previous item which should record the offset number
	rawMsg, err := offsetLog.ReadBackwardWithTag(ctx, txnMkRawMsg.LogSeqNum, parNum, tag)
	if err != nil {
		return 0, err
	}
	offsetRecord, err := tc.offsetRecordSerde.Decode(rawMsg.Payload)
	if err != nil {
		return 0, err
	}
	return offsetRecord.Offset, nil
}

func (tc *TransactionManager) EnsurePrevTxnFinAndAppendMeta(ctx context.Context) (WaitOrAppendedMeta, error) {
	var waited WaitOrAppendedMeta = None
	tc.mu.Lock()
	if env_config.ASYNC_SECOND_PHASE && !tc.hasWaitForLastTxn.Load() {
		// fmt.Fprintf(os.Stderr, "waiting for previous txn to finish\n")
		tBeg := stats.TimerBegin()
		err := tc.bgErrg.Wait()
		if err != nil {
			tc.mu.Unlock()
			return None, err
		}
		tc.waitPrevTxn.AddSample(stats.Elapsed(tBeg).Microseconds())
		waited = WaitPrev
		tc.bgErrg, tc.bgCtx = errgroup.WithContext(ctx)
		// fmt.Fprintf(os.Stderr, "previous txn finished\n")
	}
	if tc.addedNewTpPar.Load() {
		if waited == None {
			waited = AppendedNewPar
		} else {
			waited = WaitAndAppend
		}
		// debug.Fprintf(os.Stderr, "appending tp par to txn log\n")
		tBeg := stats.TimerBegin()
		tps := make([]*txn_data.TopicPartition, 0, tc.currentTopicSubstream.Len())
		tc.currentTopicSubstream.Range(func(key string, value *skipset.Uint32Set) bool {
			pars := make([]uint8, 0, value.Len())
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
		txnMeta := txn_data.TxnMetadata{
			State:           txn_data.BEGIN,
			TopicPartitions: tps,
		}
		_, err := tc.appendToTransactionLog(ctx, txnMeta, []uint64{tc.txnLogTag})
		if err != nil {
			tc.mu.Unlock()
			return None, err
		}
		tc.addedNewTpPar.Store(false)
		// debug.Fprintf(os.Stderr, "done appending tp par to txn log\n")
		tc.appendTxnMeta.AddSample(stats.Elapsed(tBeg).Microseconds())
	}
	tc.mu.Unlock()
	return waited, nil
}

// second phase of the 2-phase commit protocol
func (tc *TransactionManager) completeTransaction(ctx context.Context,
	trMark commtypes.EpochMark,
	trState txn_data.TransactionState,
	topicSubStreams []*txn_data.TopicPartition,
) error {
	tBeg := stats.TimerBegin()
	err := tc.appendTxnMarkerToStreams(ctx, trMark, topicSubStreams)
	if err != nil {
		return err
	}
	// debug.Fprintf(os.Stderr, "appended txn marker to streams\n")
	txnMd := txn_data.TxnMetadata{
		State: trState,
	}
	_, err = tc.appendToTransactionLog(ctx, txnMd, []uint64{tc.txnLogTag, tc.tranCompleteMarkerTag})
	// debug.Fprintf(os.Stderr, "appended txn complete to streams\n")
	el := stats.Elapsed(tBeg).Microseconds()
	tc.txnSndPhase.AddSample(el)
	return err
}

func (tc *TransactionManager) CommitTransaction(ctx context.Context) (uint64, bool, error) {
	// if !txn_data.PREPARE_COMMIT.IsValidPreviousState(tc.currentStatus) {
	// 	debug.Fprintf(os.Stderr, "Fail to transition from %s to PREPARE_COMMIT\n", tc.currentStatus.String())
	// 	return 0, false, common_errors.ErrInvalidStateTransition
	// }

	// rawMsgs, err := tc.SyncToRecent(ctx)
	// if err != nil {
	// 	return 0, false, fmt.Errorf("SyncToRecent: %v", err)
	// }
	// for _, rawMsg := range rawMsgs {
	// 	if rawMsg.Mark == commtypes.FENCE {
	// 		if (rawMsg.ProdId.TaskId == tc.GetCurrentTaskId() && rawMsg.ProdId.TaskEpoch > tc.GetCurrentEpoch()) ||
	// 			rawMsg.ProdId.TaskId != tc.GetCurrentTaskId() {
	// 			return 0, true, nil
	// 		}
	// 	}
	// }

	// first phase of the commit
	// debug.Fprintf(os.Stderr, "Transition to %s\n", tc.currentStatus)
	txnMd := txn_data.TxnMetadata{
		State: txn_data.PREPARE_COMMIT,
	}
	logOff, err := tc.appendToTransactionLog(ctx, txnMd, []uint64{tc.txnLogTag})
	if err != nil {
		return 0, false, err
	}
	tps := tc.collectTopicSubstreams()
	tc.cleanupState()
	// second phase of the commit
	err = tc.completeTransaction(ctx, commtypes.EPOCH_END, txn_data.COMPLETE_COMMIT, tps)
	if err != nil {
		return 0, false, err
	}
	tc.hasWaitForLastTxn.Store(true)
	return logOff, false, nil
}

func (tc *TransactionManager) SyncToRecent(ctx context.Context) ([]commtypes.RawMsg, error) {
	meta := sharedlog_stream.SyncToRecentMeta()
	// making the empty entry with the fence tag so that the fence record or the
	// empty entry will be read and skipping the progress marker entries.
	tags := []uint64{
		tc.txnLogTag,
		txn_data.FenceTag(tc.transactionLog.TopicNameHash(), 0),
	}
	producerId := tc.prodId
	off, err := tc.transactionLog.PushWithTag(ctx, nil, 0, tags, nil,
		meta, producerId)
	if err != nil {
		return nil, fmt.Errorf("PushWithTag: %v", err)
	}
	return tc.transactionLog.ReadNextWithTagUntil(ctx, 0, tags[1], off)
}

func (tc *TransactionManager) CommitTransactionAsyncComplete(ctx context.Context) (uint64, bool, error) {
	// if !txn_data.PREPARE_COMMIT.IsValidPreviousState(tc.currentStatus) {
	// 	debug.Fprintf(os.Stderr, "Fail to transition from %s to PREPARE_COMMIT\n", tc.currentStatus.String())
	// 	return 0, false, common_errors.ErrInvalidStateTransition
	// }
	// rawMsgs, err := tc.SyncToRecent(ctx)
	// if err != nil {
	// 	return 0, false, fmt.Errorf("SyncToRecent: %v", err)
	// }
	// for _, rawMsg := range rawMsgs {
	// 	if rawMsg.Mark == commtypes.FENCE {
	// 		if (rawMsg.ProdId.TaskId == tc.GetCurrentTaskId() && rawMsg.ProdId.TaskEpoch > tc.GetCurrentEpoch()) ||
	// 			rawMsg.ProdId.TaskId != tc.GetCurrentTaskId() {
	// 			return 0, true, nil
	// 		}
	// 	}
	// }

	// first phase of the commit
	// debug.Fprintf(os.Stderr, "Transition to %s\n", tc.currentStatus)
	txnMd := txn_data.TxnMetadata{
		State: txn_data.PREPARE_COMMIT,
	}
	logOff, err := tc.appendToTransactionLog(ctx, txnMd, []uint64{tc.txnLogTag})
	if err != nil {
		return 0, false, err
	}
	tps := tc.collectTopicSubstreams()
	tc.cleanupState()
	tc.hasWaitForLastTxn.Store(false)
	tc.bgErrg.Go(func() error {
		// second phase of the commit
		err = tc.completeTransaction(tc.bgCtx, commtypes.EPOCH_END, txn_data.COMPLETE_COMMIT, tps)
		if err != nil {
			return err
		}
		tc.hasWaitForLastTxn.Store(true)
		return nil
	})
	return logOff, false, nil
}

func (tc *TransactionManager) AbortTransaction(ctx context.Context) error {
	// if !txn_data.PREPARE_ABORT.IsValidPreviousState(tc.currentStatus) {
	// 	debug.Fprintf(os.Stderr, "fail to transition state from %d to PRE_ABORT", tc.currentStatus)
	// 	return common_errors.ErrInvalidStateTransition
	// }
	// debug.Fprintf(os.Stderr, "Transition to %s\n", tc.currentStatus)
	txnMd := txn_data.TxnMetadata{
		State: txn_data.PREPARE_ABORT,
	}
	_, err := tc.appendToTransactionLog(ctx, txnMd, []uint64{tc.txnLogTag})
	if err != nil {
		return err
	}
	tps := tc.collectTopicSubstreams()
	tc.cleanupState()
	err = tc.completeTransaction(ctx, commtypes.ABORT, txn_data.COMPLETE_ABORT, tps)
	if err != nil {
		return err
	}
	return nil
}

func (tc *TransactionManager) cleanupState() {
	// debug.Fprintf(os.Stderr, "Transition to %s\n", tc.currentStatus)
	tc.currentTopicSubstream = skipmap.NewString[*skipset.Uint32Set]()
	tc.addedNewTpPar.Store(false)
}
