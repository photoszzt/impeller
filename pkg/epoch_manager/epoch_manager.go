package epoch_manager

import (
	"context"
	"fmt"
	"math"
	"os"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/txn_data"

	"cs.utexas.edu/zjia/faas/types"
	"github.com/zhangyunhao116/skipmap"
	"github.com/zhangyunhao116/skipset"
)

const (
	EPOCH_LOG_TOPIC_NAME = "__epoch_log"
)

type EpochManager struct {
	epochMetaSerde        commtypes.SerdeG[commtypes.EpochMarker]
	currentTopicSubstream *skipmap.StringMap[*skipset.Uint32Set]
	epochLog              *sharedlog_stream.SharedLogStream
	tpHashes              map[string]uint64
	epochMngrName         string
	prodId                commtypes.ProducerId
	epochLogMarkerTag     uint64
	useBuf                bool
}

func NewEpochManager(epochMngrName string,
	serdeFormat commtypes.SerdeFormat,
) (*EpochManager, error) {
	log, err := sharedlog_stream.NewSharedLogStream(
		EPOCH_LOG_TOPIC_NAME+"_"+epochMngrName, serdeFormat)
	if err != nil {
		return nil, err
	}
	epochMetaSerde, err := commtypes.GetEpochMarkerSerdeG(serdeFormat)
	if err != nil {
		return nil, err
	}
	return &EpochManager{
		epochMngrName:         epochMngrName,
		prodId:                commtypes.NewProducerId(),
		currentTopicSubstream: skipmap.NewString[*skipset.Uint32Set](),
		epochLog:              log,
		epochMetaSerde:        epochMetaSerde,
		epochLogMarkerTag:     txn_data.MarkerTag(log.TopicNameHash(), 0),
		tpHashes:              make(map[string]uint64),
		useBuf:                epochMetaSerde.UsedBufferPool(),
	}, nil
}

func (em *EpochManager) RecordTpHashes(tpName string, stream *sharedlog_stream.ShardedSharedLogStream) {
	em.tpHashes[tpName] = stream.TopicNameHash()
}

var _ = exactly_once_intr.ReadOnlyExactlyOnceManager(&EpochManager{})

func (em *EpochManager) GetEpochManagerName() string {
	return em.epochMngrName
}
func (em *EpochManager) GetCurrentEpoch() uint32             { return em.prodId.TaskEpoch }
func (em *EpochManager) GetCurrentTaskId() uint64            { return em.prodId.TaskId }
func (em *EpochManager) GetProducerId() commtypes.ProducerId { return em.prodId }

func (em *EpochManager) appendToEpochLog(ctx context.Context,
	meta commtypes.EpochMarker, tags []uint64, additionalTopic []string,
) (uint64, int, error) {
	debug.Assert(meta.Mark != commtypes.EMPTY, "mark should not be empty")
	encoded, b, err := em.epochMetaSerde.Encode(meta)
	defer func() {
		if em.useBuf && b != nil {
			*b = encoded
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return 0, 0, err
	}
	// debug.Fprintf(os.Stderr, "encoded epochMeta: %v\n", string(encoded))
	debug.Assert(tags != nil, "tags should not be null")
	producerId := commtypes.ProducerId{
		TaskId:    em.prodId.TaskId,
		TaskEpoch: em.prodId.TaskEpoch,
	}
	off, err := em.epochLog.PushWithTag(ctx, encoded, 0, tags, additionalTopic,
		sharedlog_stream.ControlRecordMeta, producerId)
	// debug.Fprintf(os.Stderr, "epochOff: 0x%x, output range: %v, consume: ", off, meta.OutputRanges)
	// for s, c := range meta.ConSeqNums {
	// 	debug.Fprintf(os.Stderr, "%s:0x%x", s, c)
	// }
	// debug.Fprint(os.Stderr, "\n")
	return off, len(encoded), err
}

func (em *EpochManager) SyncToRecent(ctx context.Context) ([]commtypes.RawMsg, error) {
	meta := sharedlog_stream.SyncToRecentMeta()
	// making the empty entry with the fence tag so that the fence record or the
	// empty entry will be read and skipping the progress marker entries.
	tags := []uint64{
		sharedlog_stream.NameHashWithPartition(em.epochLog.TopicNameHash(), 0),
		txn_data.FenceTag(em.epochLog.TopicNameHash(), 0),
	}
	producerId := commtypes.ProducerId{
		TaskId:    em.prodId.TaskId,
		TaskEpoch: em.prodId.TaskEpoch,
	}
	off, err := em.epochLog.PushWithTag(ctx, nil, 0, tags, nil,
		meta, producerId)
	if err != nil {
		return nil, fmt.Errorf("PushWithTag: %v", err)
	}
	return em.epochLog.ReadNextWithTagUntil(ctx, 0, tags[1], off)
}

func (em *EpochManager) SyncToRecentNoRead(ctx context.Context) (uint64, error) {
	meta := sharedlog_stream.SyncToRecentMeta()
	// making the empty entry with the fence tag so that the fence record or the
	// empty entry will be read and skipping the progress marker entries.
	tags := []uint64{
		sharedlog_stream.NameHashWithPartition(em.epochLog.TopicNameHash(), 0),
		txn_data.FenceTag(em.epochLog.TopicNameHash(), 0),
	}
	producerId := commtypes.ProducerId{
		TaskId:    em.prodId.TaskId,
		TaskEpoch: em.prodId.TaskEpoch,
	}
	off, err := em.epochLog.PushWithTag(ctx, nil, 0, tags, nil,
		meta, producerId)
	if err != nil {
		return 0, fmt.Errorf("PushWithTag: %v", err)
	}
	return off, nil
}

// multiple goroutines might call this function
func (em *EpochManager) AddTopicSubstream(
	topic string, substreamNum uint8,
) {
	parSet, _ := em.currentTopicSubstream.LoadOrStore(topic, skipset.NewUint32())
	parSet.Add(uint32(substreamNum))
}

func GenEpochMarker(
	ctx context.Context,
	em *EpochManager,
	ectx processor.ExecutionContext,
	kvTabs map[string]store.KeyValueStoreOpWithChangelog,
	winTabs map[string]store.WindowStoreOpWithChangelog,
) (commtypes.EpochMarker, error) {
	outputRanges := make(map[string][]commtypes.ProduceRange)
	for _, producer := range ectx.Producers() {
		topicName := producer.TopicName()
		parSet, ok := em.currentTopicSubstream.Load(producer.TopicName())
		if ok {
			ranges := make([]commtypes.ProduceRange, 0, 32)
			parSet.Range(func(par uint32) bool {
				ranges = append(ranges, commtypes.ProduceRange{
					SubStreamNum: uint8(par),
					Start:        producer.GetInitialProdSeqNum(uint8(par)),
				})
				return true
			})
			outputRanges[topicName] = ranges
		}
	}
	for _, kvTab := range kvTabs {
		if !kvTab.ChangelogIsSrc() {
			ranges := []commtypes.ProduceRange{
				{
					SubStreamNum: kvTab.SubstreamNum(),
					Start:        kvTab.GetInitialProdSeqNum(),
				},
			}
			outputRanges[kvTab.ChangelogTopicName()] = ranges
		}
	}
	for _, winTab := range winTabs {
		ranges := []commtypes.ProduceRange{
			{
				SubStreamNum: winTab.SubstreamNum(),
				Start:        winTab.GetInitialProdSeqNum(),
			},
		}
		outputRanges[winTab.ChangelogTopicName()] = ranges
	}
	consumeSeqNums := make(map[string]uint64)
	for _, consumer := range ectx.Consumers() {
		consumeSeqNums[consumer.TopicName()] = consumer.CurrentConsumedSeqNum()
	}
	epochMeta := commtypes.EpochMarker{
		Mark:         commtypes.EPOCH_END,
		ConSeqNums:   consumeSeqNums,
		OutputRanges: outputRanges,
		ProdIndex:    ectx.SubstreamNum(),
	}
	return epochMeta, nil
}

func (em *EpochManager) GenTagsAndTopicsForEpochMarker() ([]uint64, []string) {
	tags := []uint64{em.epochLogMarkerTag}
	// debug.Fprintf(os.Stderr, "marker with tag: 0x%x\n", em.epochLogMarkerTag)
	var additionalTopic []string
	em.currentTopicSubstream.Range(func(tp string, parSet *skipset.Uint32Set) bool {
		// debug.Fprintf(os.Stderr, "tpSubstream tp: %s, parSet: %v\n", tp, parSet)
		additionalTopic = append(additionalTopic, tp)

		nameHash, ok := em.tpHashes[tp]
		if !ok {
			nameHash = hashfuncs.NameHash(tp)
			em.tpHashes[tp] = nameHash
		}
		parSet.Range(func(sub uint32) bool {
			tag := sharedlog_stream.NameHashWithPartition(nameHash, uint8(sub))
			// debug.Fprintf(os.Stderr, "marker with tag: 0x%x\n", tag)
			tags = append(tags, tag)
			return true
		})
		return true
	})
	return tags, additionalTopic
}

func (em *EpochManager) MarkEpoch(ctx context.Context,
	epochMeta commtypes.EpochMarker, tags []uint64, additionalTopic []string,
) (uint64, int, error) {
	// debug.Fprintf(os.Stderr, "epochMeta to push: %+v\n", epochMeta)
	return em.appendToEpochLog(ctx, epochMeta, tags, additionalTopic)
}

func (em *EpochManager) Init(ctx context.Context) (*commtypes.EpochMarker, *commtypes.RawMsg, error) {
	env := ctx.Value(commtypes.ENVID{}).(types.Environment)
	debug.Assert(env != nil, "env should be set")
	off, err := em.SyncToRecentNoRead(ctx)
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(os.Stderr, "%s sync to recent, off: %#x\n", em.epochLog.TopicName(), off)
	recentMeta, metaMsg, err := em.getMostRecentCommitEpoch(ctx, off)
	if err != nil {
		return nil, nil, err
	}
	if recentMeta == nil {
		em.prodId.InitTaskId(env)
		em.prodId.TaskEpoch = 0
	} else {
		em.prodId.TaskEpoch = metaMsg.ProdId.TaskEpoch
		em.prodId.TaskId = metaMsg.ProdId.TaskId
	}
	if recentMeta != nil && metaMsg.ProdId.TaskEpoch == math.MaxUint16 {
		em.prodId.InitTaskId(env)
		em.prodId.TaskEpoch = 0
	}
	em.prodId.TaskEpoch += 1
	meta := commtypes.EpochMarker{
		Mark: commtypes.FENCE,
	}
	elHash := em.epochLog.TopicNameHash()
	tags := []uint64{
		sharedlog_stream.NameHashWithPartition(elHash, 0),
		txn_data.FenceTag(elHash, 0),
	}
	_, _, err = em.appendToEpochLog(ctx, meta, tags, nil)
	if err != nil {
		return nil, nil, err
	}
	return recentMeta, metaMsg, nil
}

func (em *EpochManager) getMostRecentCommitEpoch(ctx context.Context, recentOff uint64) (*commtypes.EpochMarker, *commtypes.RawMsg, error) {
	rawMsg, err := em.epochLog.ReadBackwardWithTag(ctx, recentOff, 0, em.epochLogMarkerTag)
	if err != nil {
		if common_errors.IsStreamEmptyError(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	meta, err := em.epochMetaSerde.Decode(rawMsg.Payload)
	if err != nil {
		return nil, nil, err
	}
	return &meta, rawMsg, nil
}

func (em *EpochManager) FindLastEpochMetaThatHasConsumed(
	ctx context.Context, seqNum uint64,
) (meta *commtypes.EpochMarker, rawMsg *commtypes.RawMsg, err error) {
	tailSeqNum := seqNum
	for {
		rawMsg, err := em.epochLog.ReadBackwardWithTag(ctx, tailSeqNum, 0, em.epochLogMarkerTag)
		if err != nil {
			if common_errors.IsStreamEmptyError(err) {
				return nil, nil, nil
			}
			return nil, nil, err
		}
		meta, err := em.epochMetaSerde.Decode(rawMsg.Payload)
		if err != nil {
			return nil, nil, err
		}
		if len(meta.ConSeqNums) != 0 {
			return &meta, rawMsg, nil
		}
		tailSeqNum = rawMsg.LogSeqNum - 1
	}
}

func (em *EpochManager) FindLastEpochMetaWithAuxData(ctx context.Context, seqNum uint64) (auxData []byte, metaSeqNum uint64, err error) {
	tailSeqNum := seqNum
	for {
		rawMsg, err := em.epochLog.ReadBackwardWithTag(ctx, tailSeqNum, 0, em.epochLogMarkerTag)
		if err != nil {
			if common_errors.IsStreamEmptyError(err) {
				return nil, 0, nil
			}
			return nil, 0, err
		}
		if len(rawMsg.AuxData) != 0 {
			return rawMsg.AuxData, rawMsg.LogSeqNum, nil
		}
		tailSeqNum = rawMsg.LogSeqNum - 1
	}
}

func CleanupState(em *EpochManager, producers []producer_consumer.MeteredProducerIntr,
	kvTabs map[string]store.KeyValueStoreOpWithChangelog, winTabs map[string]store.WindowStoreOpWithChangelog,
) {
	em.currentTopicSubstream = skipmap.NewString[*skipset.Uint32Set]()
	for _, producer := range producers {
		producer.ResetInitialProd()
	}
	for _, kvTab := range kvTabs {
		if !kvTab.ChangelogIsSrc() {
			kvTab.ResetInitialProd()
		}
	}
	for _, winTab := range winTabs {
		winTab.ResetInitialProd()
	}
}
