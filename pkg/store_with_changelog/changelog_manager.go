package store_with_changelog

import (
	"context"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/sharedlog_stream"
	"time"

	"cs.utexas.edu/zjia/faas/protocol"
	"golang.org/x/xerrors"
)

// a changelog substream has only one producer. Each store would only produce to one substream.
type ChangelogManager[K, V any] struct {
	msgSerde         commtypes.MessageGSerdeG[K, V]
	epochMetaSerde   commtypes.SerdeG[commtypes.EpochMarker]
	producer         *producer_consumer.ShardedSharedLogStreamProducer
	restoreConsumers []*producer_consumer.ShardedSharedLogStreamConsumer
	timeout          time.Duration
	numProduced      uint32
	changelogIsSrc   bool
	serdeFormat      commtypes.SerdeFormat
	instanceId       uint8
}

func NewChangelogManager[K, V any](stream *sharedlog_stream.ShardedSharedLogStream,
	msgSerde commtypes.MessageGSerdeG[K, V],
	timeout time.Duration,
	flushDuration time.Duration,
	serdeFormat commtypes.SerdeFormat,
	instanceId uint8,
) (*ChangelogManager[K, V], error) {
	numSubStreams := stream.NumPartition()
	consumers := make([]*producer_consumer.ShardedSharedLogStreamConsumer, numSubStreams)
	for i := uint8(0); i < numSubStreams; i++ {
		consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream,
			&producer_consumer.StreamConsumerConfig{
				Timeout:     timeout,
				SerdeFormat: serdeFormat,
			}, 1, instanceId)
		if err != nil {
			return nil, err
		}
		consumers[i] = consumer
	}
	epochMetaSerde, err := commtypes.GetEpochMarkerSerdeG(serdeFormat)
	if err != nil {
		return nil, err
	}
	return &ChangelogManager[K, V]{
		epochMetaSerde:   epochMetaSerde,
		restoreConsumers: consumers,
		producer: producer_consumer.NewShardedSharedLogStreamProducer(stream,
			&producer_consumer.StreamSinkConfig{
				FlushDuration: flushDuration,
				Format:        serdeFormat,
			}),
		changelogIsSrc: false,
		msgSerde:       msgSerde,
		numProduced:    0,
		timeout:        timeout,
		serdeFormat:    serdeFormat,
		instanceId:     instanceId,
	}, nil
}

func (cm *ChangelogManager[K, V]) ChangelogIsSrc() bool {
	return cm.changelogIsSrc
}

func (cm *ChangelogManager[K, V]) OutputRemainingStats() {
	cm.producer.OutputRemainingStats()
}

func (cm *ChangelogManager[K, V]) TopicName() string {
	return cm.restoreConsumers[0].TopicName()
}

func (cm *ChangelogManager[K, V]) NumPartition() uint8 {
	return cm.restoreConsumers[0].Stream().NumPartition()
}

func (cm *ChangelogManager[K, V]) ConfigExactlyOnce(
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	if !cm.changelogIsSrc {
		cm.producer.ConfigExactlyOnce(rem, guarantee)
		for _, c := range cm.restoreConsumers {
			c.ConfigExactlyOnce(guarantee)
		}
	}
}

func (cm *ChangelogManager[K, V]) Stream() sharedlog_stream.Stream {
	return cm.restoreConsumers[0].Stream()
}

// when changelog is src, there's nothing to flush
func (cm *ChangelogManager[K, V]) Flush(ctx context.Context) (uint32, error) {
	if !cm.changelogIsSrc {
		// debug.Fprintf(os.Stderr, "flushing changelog manager, current produce: %d\n", cm.numProduced)
		return cm.producer.FlushNoLock(ctx)
	}
	return 0, nil
}

func (cm *ChangelogManager[K, V]) produce(ctx context.Context, msgSer commtypes.MessageSerialized, parNum uint8) error {
	if !cm.changelogIsSrc {
		cm.numProduced += 1
		return cm.producer.ProduceDataNoLock(ctx, msgSer, parNum)
	}
	return nil
}

func (cm *ChangelogManager[K, V]) findLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error) {
	// in this case, previous changelog has more substreams than current changelog
	if int(parNum) >= len(cm.restoreConsumers) {
		consumers := make([]*producer_consumer.ShardedSharedLogStreamConsumer, parNum+1)
		curLen := len(cm.restoreConsumers)
		for i := 0; i < curLen; i++ {
			consumers[i] = cm.restoreConsumers[i]
		}
		for i := uint8(curLen); i < parNum+1; i++ {
			consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(
				cm.producer.Stream().(*sharedlog_stream.ShardedSharedLogStream),
				&producer_consumer.StreamConsumerConfig{
					Timeout:     cm.timeout,
					SerdeFormat: cm.serdeFormat,
				}, 1, cm.instanceId)
			if err != nil {
				return nil, 0, err
			}
			consumers[i] = consumer
		}
		cm.restoreConsumers = consumers
	}
	stream := cm.restoreConsumers[parNum].Stream()
	tag := sharedlog_stream.NameHashWithPartition(stream.TopicNameHash(), parNum)
	tailSeqNum := protocol.MaxLogSeqnum
	for {
		rawMsg, err := stream.ReadBackwardWithTag(ctx, tailSeqNum, parNum, tag)
		if xerrors.Is(err, common_errors.ErrStreamEmpty) {
			return nil, 0, nil
		}
		if err != nil {
			return nil, 0, err
		}
		if rawMsg.IsControl && len(rawMsg.AuxData) != 0 {
			return rawMsg.AuxData, rawMsg.LogSeqNum, nil
		} else {
			tailSeqNum = rawMsg.LogSeqNum - 1
		}
	}
}

func (cm *ChangelogManager[K, V]) ProduceCount() uint32 {
	return cm.numProduced
}

func (cm *ChangelogManager[K, V]) SetChangelogConsumeCursor(seqNum uint64, parNum uint8) {
	cm.restoreConsumers[parNum].SetCursor(seqNum, parNum)
}

func (cm *ChangelogManager[K, V]) Consume(ctx context.Context, parNum uint8) (*commtypes.MsgAndSeqG[K, V], error) {
	rawMsg, err := cm.restoreConsumers[parNum].Consume(ctx, parNum)
	if err != nil {
		return nil, err
	}
	return commtypes.DecodeRawMsgSeqG(rawMsg, cm.msgSerde)
}

func CreateChangelog(tabName string,
	numPartition uint8, serdeFormat commtypes.SerdeFormat,
	bufMaxSize uint32,
) (*sharedlog_stream.ShardedSharedLogStream, error) {
	changelog_name := tabName + "-changelog"
	return sharedlog_stream.NewShardedSharedLogStream(changelog_name, numPartition,
		serdeFormat, bufMaxSize)
}
