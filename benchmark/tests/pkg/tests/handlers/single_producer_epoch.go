package handlers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/epoch_manager"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"time"

	"golang.org/x/xerrors"
)

func getEpochManager(ctx context.Context,
	transactionalID string, serdeFormat commtypes.SerdeFormat,
) (*epoch_manager.EpochManager, exactly_once_intr.TrackProdSubStreamFunc, error) {
	em, err := epoch_manager.NewEpochManager(transactionalID, serdeFormat)
	if err != nil {
		return nil, nil, err
	}
	_, _, err = em.Init(ctx)
	if err != nil {
		return nil, nil, err
	}
	trackParFunc := exactly_once_intr.TrackProdSubStreamFunc(
		func(topicName string, substreamId uint8,
		) {
			em.AddTopicSubstream(topicName, substreamId)
		})
	return em, trackParFunc, nil
}

func getSrcSink(ctx context.Context, sp *common.TestParam,
) ([]*producer_consumer.MeteredConsumer, []producer_consumer.MeteredProducerIntr, error) {
	srcStreams, sinkStreams, err := benchutil.GetShardedInputOutputStreamsTest(ctx, sp)
	if err != nil {
		panic(err)
	}
	serdeFormat := commtypes.SerdeFormat(sp.SerdeFormat)
	timeout := time.Duration(10) * time.Millisecond
	srcConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     timeout,
		SerdeFormat: serdeFormat,
	}
	outConfig := &producer_consumer.StreamSinkConfig{
		FlushDuration: time.Duration(sp.FlushMs) * time.Millisecond,
		Format:        serdeFormat,
	}
	warmup := time.Duration(0) * time.Second
	var consumers []*producer_consumer.MeteredConsumer
	for _, srcStream := range srcStreams {
		consumer, err := producer_consumer.NewShardedSharedLogStreamConsumer(srcStream,
			srcConfig, 1, 0)
		if err != nil {
			return nil, nil, err
		}
		src := producer_consumer.NewMeteredConsumer(consumer, warmup)
		consumers = append(consumers, src)
	}
	sink, err := producer_consumer.NewMeteredProducer(
		producer_consumer.NewShardedSharedLogStreamProducer(sinkStreams[0], outConfig), warmup)
	if err != nil {
		return nil, nil, err
	}
	return consumers, []producer_consumer.MeteredProducerIntr{sink}, nil
}

func (h *produceConsumeHandler) testSingleProduceConsumeEpoch(ctx context.Context,
	serdeFormat commtypes.SerdeFormat, topicName string,
) {
	debug.Fprintf(os.Stderr, "single produce consume epoch test\n")
	default_max_buf := uint32(32 * 1024)
	sp := &common.TestParam{
		InStreamParam: []common.TestStreamParam{
			{TopicName: topicName, NumPartition: 1},
		},
		OutStreamParam: []common.TestStreamParam{
			{TopicName: topicName, NumPartition: 1},
		},
		AppId:         "testStreamStreamJoinMem",
		CommitEveryMs: 100,
		FlushMs:       100,
		BufMaxSize:    default_max_buf,
		SerdeFormat:   uint8(commtypes.JSON),
	}
	msgSerde, err := commtypes.GetMsgGSerdeG[int, string](commtypes.JSON, commtypes.IntSerdeG{}, commtypes.StringSerdeG{})
	if err != nil {
		panic(err)
	}
	srcs, sinks, err := getSrcSink(ctx, sp)
	if err != nil {
		panic(err)
	}
	ectx := processor.NewExecutionContextFromComponents(proc_interface.NewBaseSrcsSinks(srcs, sinks),
		proc_interface.NewBaseProcArgs("singleProdConsumeEpoch", 1, 0))
	em1, trackParFunc1, err := getEpochManager(ctx,
		"prod1_single"+serdeFormat.String(), serdeFormat)
	if err != nil {
		panic(err)
	}
	sinks[0].ConfigExactlyOnce(em1, exactly_once_intr.EPOCH_MARK)
	trackParFunc1(sinks[0].TopicName(), 0)
	msgForTm1 := []commtypes.MessageG[int, string]{
		{Key: optional.Some(1), Value: optional.Some("tm1_a")},
	}
	err = pushMsgsToSink(ctx, sinks[0], msgForTm1, msgSerde)
	if err != nil {
		panic(err)
	}
	sinks[0].Flush(ctx)
	msgForTm2 := []commtypes.MessageG[int, string]{
		{Key: optional.Some(2), Value: optional.Some("tm2_a")},
	}
	err = pushMsgsToSink(ctx, sinks[0], msgForTm2, msgSerde)
	if err != nil {
		panic(err)
	}
	_, err = sinks[0].Flush(ctx)
	if err != nil {
		panic(err)
	}
	epochMarker, err := epoch_manager.GenEpochMarker(ctx, em1, &ectx,
		nil, nil)
	if err != nil {
		panic(err)
	}
	epochMarkerTags, epochMarkerTopics := em1.GenTagsAndTopicsForEpochMarker()
	_, _, err = em1.MarkEpoch(ctx, epochMarker, epochMarkerTags, epochMarkerTopics)
	if err != nil {
		panic(err)
	}

	srcs[0].ConfigExactlyOnce(exactly_once_intr.EPOCH_MARK)
	got, err := readMsgsEpoch(ctx, srcs[0], msgSerde)
	if err != nil {
		panic(err)
	}
	expected_msgs := []commtypes.MessageG[int, string]{
		{Key: optional.Some(1), Value: optional.Some("tm1_a")},
		{Key: optional.Some(2), Value: optional.Some("tm2_a")},
	}
	if !reflect.DeepEqual(expected_msgs, got) {
		panic(fmt.Sprintf("should equal. expected: %v, got: %v", expected_msgs, got))
	}

	debug.Fprintf(os.Stderr, "single producer produces two epochs\n")
	// two epochs
	trackParFunc1(sinks[0].TopicName(), 0)
	msgForTm1 = []commtypes.MessageG[int, string]{
		{
			Key:   optional.Some(3),
			Value: optional.Some("tm1_c"),
		},
		{
			Key:   optional.Some(4),
			Value: optional.Some("tm1_d"),
		},
	}
	err = pushMsgsToSink(ctx, sinks[0], msgForTm1, msgSerde)
	if err != nil {
		panic(err)
	}
	epochMarker, err = epoch_manager.GenEpochMarker(ctx, em1, &ectx,
		nil, nil)
	if err != nil {
		panic(err)
	}
	epochMarkerTags, epochMarkerTopics = em1.GenTagsAndTopicsForEpochMarker()
	_, _, err = em1.MarkEpoch(ctx, epochMarker, epochMarkerTags, epochMarkerTopics)
	if err != nil {
		panic(err)
	}

	msgForTm1 = []commtypes.MessageG[int, string]{
		{Key: optional.Some(5), Value: optional.Some("tm1_e")},
		{Key: optional.Some(6), Value: optional.Some("tm1_f")},
	}
	err = pushMsgsToSink(ctx, sinks[0], msgForTm1, msgSerde)
	if err != nil {
		panic(err)
	}
	epochMarker, err = epoch_manager.GenEpochMarker(ctx, em1, &ectx,
		nil, nil)
	if err != nil {
		panic(err)
	}
	epochMarkerTags, epochMarkerTopics = em1.GenTagsAndTopicsForEpochMarker()
	_, _, err = em1.MarkEpoch(ctx, epochMarker, epochMarkerTags, epochMarkerTopics)
	if err != nil {
		panic(err)
	}

	got, err = readMsgsEpoch[int, string](ctx, srcs[0], msgSerde)
	if err != nil {
		panic(err)
	}
	expected_msgs = []commtypes.MessageG[int, string]{
		{Key: optional.Some(3), Value: optional.Some("tm1_c")},
		{Key: optional.Some(4), Value: optional.Some("tm1_d")},
		{Key: optional.Some(5), Value: optional.Some("tm1_e")},
		{Key: optional.Some(6), Value: optional.Some("tm1_f")},
	}
	if !reflect.DeepEqual(expected_msgs, got) {
		panic(fmt.Sprintf("should equal. expected: %v, got: %v", expected_msgs, got))
	}
}

func readMsgsEpoch[KIn, VIn any](ctx context.Context,
	consumer *producer_consumer.MeteredConsumer,
	msgSerde commtypes.MessageGSerdeG[KIn, VIn],
) ([]commtypes.MessageG[KIn, VIn], error) {
	ret := make([]commtypes.MessageG[KIn, VIn], 0)
	for {
		gotMsgs, err := consumer.Consume(ctx, 0)
		if xerrors.Is(err, common_errors.ErrStreamSourceTimeout) {
			return ret, nil
		} else if err != nil {
			return nil, err
		}
		decoded, err := commtypes.DecodeRawMsgSeqG(gotMsgs, msgSerde)
		if err != nil {
			return nil, err
		}
		if decoded.MsgArr != nil {
			ret = append(ret, decoded.MsgArr...)
		} else {
			ret = append(ret, decoded.Msg)
		}
	}
}
