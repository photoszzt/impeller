package handlers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/epoch_manager"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
)

func (h *produceConsumeHandler) testMultiProducerEpoch(
	ctx context.Context,
	serdeFormat commtypes.SerdeFormat,
	topicName string,
) {
	debug.Fprintf(os.Stderr, "multi producer epoch test\n")
	default_max_buf := uint32(32 * 1024)
	sp := &common.TestParam{
		InStreamParam: []common.TestStreamParam{
			{TopicName: topicName, NumPartition: 1},
		},
		OutStreamParam: []common.TestStreamParam{
			{TopicName: topicName, NumPartition: 1},
			{TopicName: topicName, NumPartition: 1},
		},
		AppId:         "testStreamStreamJoinMem",
		CommitEveryMs: 100,
		FlushMs:       100,
		BufMaxSize:    default_max_buf,
		SerdeFormat:   uint8(serdeFormat),
	}
	srcs, sinks, err := getSrcSink(ctx, sp)
	if err != nil {
		panic(err)
	}
	meteredProducer1 := sinks[0]
	meteredProducer2 := sinks[1]
	msgSerde, err := commtypes.GetMsgGSerdeG[int, string](serdeFormat,
		commtypes.IntSerdeG{},
		commtypes.StringSerdeG{},
	)
	if err != nil {
		panic(err)
	}
	em1, trackParFunc1, err := getEpochManager(ctx,
		"prod1_multi"+serdeFormat.String(), serdeFormat)
	if err != nil {
		panic(err)
	}
	em2, trackParFunc2, err := getEpochManager(ctx,
		"prod2_multi"+serdeFormat.String(), serdeFormat)
	if err != nil {
		panic(err)
	}
	meteredProducer1.ConfigExactlyOnce(em1, exactly_once_intr.EPOCH_MARK)
	meteredProducer2.ConfigExactlyOnce(em2, exactly_once_intr.EPOCH_MARK)

	// producer1 push 1 msg to sink
	msgForTm1 := []commtypes.MessageG[int, string]{
		{Key: optional.Some(1), Value: optional.Some("tm1_a")},
	}
	err = pushMsgsToSink(ctx, meteredProducer1, msgForTm1, msgSerde)
	if err != nil {
		panic(err)
	}
	meteredProducer1.Flush(ctx)

	// then producer2 produce 1 msg to sink
	msgForTm2 := []commtypes.MessageG[int, string]{
		{Key: optional.Some(2), Value: optional.Some("tm2_a")},
	}
	err = pushMsgsToSink(ctx, meteredProducer2, msgForTm2, msgSerde)
	if err != nil {
		panic(err)
	}
	meteredProducer2.Flush(ctx)

	// producer1 push 1 msg to sink
	msgForTm1 = []commtypes.MessageG[int, string]{
		{Key: optional.Some(3), Value: optional.Some("tm1_b")},
	}
	err = pushMsgsToSink(ctx, meteredProducer1, msgForTm1, msgSerde)
	if err != nil {
		panic(err)
	}
	meteredProducer1.Flush(ctx)

	// producer1 mark
	trackParFunc1(sinks[0].TopicName(), 0)
	producers1 := []producer_consumer.MeteredProducerIntr{meteredProducer1}
	ectx1 := processor.NewExecutionContextFromComponents(proc_interface.NewBaseSrcsSinks(nil, producers1),
		proc_interface.NewBaseProcArgs("multiProdConsumeEpoch", 1, 0))
	epochMarker1, err := epoch_manager.GenEpochMarker(ctx, em1, &ectx1,
		nil, nil)
	if err != nil {
		panic(err)
	}
	epochMarkerTags1, epochMarkerTopics1 := em1.GenTagsAndTopicsForEpochMarker()
	_, _, err = em1.MarkEpoch(ctx, epochMarker1, epochMarkerTags1, epochMarkerTopics1)
	if err != nil {
		panic(err)
	}

	// producer2 outputs two message
	msgForTm2 = []commtypes.MessageG[int, string]{
		{Key: optional.Some(4), Value: optional.Some("tm2_b")},
		{Key: optional.Some(5), Value: optional.Some("tm2_c")},
	}
	err = pushMsgsToSink(ctx, meteredProducer2, msgForTm2, msgSerde)
	if err != nil {
		panic(err)
	}
	meteredProducer2.Flush(ctx)
	trackParFunc2(sinks[1].TopicName(), 0)
	producers2 := []producer_consumer.MeteredProducerIntr{meteredProducer2}
	ectx2 := processor.NewExecutionContextFromComponents(proc_interface.NewBaseSrcsSinks(nil, producers2),
		proc_interface.NewBaseProcArgs("multiProdConsumeEpoch2", 1, 0))
	epochMarker2, err := epoch_manager.GenEpochMarker(ctx, em2, &ectx2,
		nil, nil)
	if err != nil {
		panic(err)
	}
	epochMarkerTags2, epochMarkerTopics2 := em2.GenTagsAndTopicsForEpochMarker()
	_, _, err = em2.MarkEpoch(ctx, epochMarker2, epochMarkerTags2, epochMarkerTopics2)
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
		{Key: optional.Some(3), Value: optional.Some("tm1_b")},
		{Key: optional.Some(4), Value: optional.Some("tm2_b")},
		{Key: optional.Some(5), Value: optional.Some("tm2_c")},
	}
	if !reflect.DeepEqual(expected_msgs, got) {
		panic(fmt.Sprintf("should equal. expected: %v, got: %v", expected_msgs, got))
	}
}
