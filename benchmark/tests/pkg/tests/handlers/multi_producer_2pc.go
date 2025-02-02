package handlers

import (
	"context"
	"fmt"
	"reflect"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/producer_consumer"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/stream_task"
	"sharedlog-stream/pkg/transaction"
)

func getProduceTransactionManager(
	ctx context.Context, transactionalID string,
	sink *producer_consumer.ShardedSharedLogStreamProducer,
	sp *common.QueryInput,
) *transaction.TransactionManager {
	p, err := producer_consumer.NewMeteredProducer(sink, 0)
	if err != nil {
		panic(err)
	}
	ectx := processor.NewExecutionContext([]*producer_consumer.MeteredConsumer{},
		[]producer_consumer.MeteredProducerIntr{p},
		"prodConsume", sp.ScaleEpoch, sp.ParNum)
	args1, err := benchutil.UpdateStreamTaskArgs(sp, stream_task.NewStreamTaskArgsBuilder(
		&ectx, transactionalID)).Build()
	if err != nil {
		panic(err)
	}
	tm1, err := stream_task.SetupManagersFor2pcTest(ctx, args1)
	if err != nil {
		panic(err)
	}
	return tm1
}

/*
func getConsumeTransactionManager(
	ctx context.Context, env types.Environment, transactionalID string,
	src *producer_consumer.ShardedSharedLogStreamConsumer, sp *common.QueryInput,
) (*transaction.TransactionManager, exactly_once_intr.TrackProdSubStreamFunc) {
	ectx := processor.NewExecutionContext(
		[]producer_consumer.MeteredConsumerIntr{producer_consumer.NewMeteredConsumer(src, 0)},
		nil, "prodConsume", sp.ScaleEpoch, sp.ParNum)
	args1 := benchutil.UpdateStreamTaskArgs(sp, stream_task.NewStreamTaskArgsBuilder(env, &ectx, transactionalID)).Build()
	tm1, err := stream_task.SetupTransactionManager(ctx, args1)
	if err != nil {
		panic(err)
	}
	trackParFunc1 := func(ctx context.Context,
		key interface{},
		keySerde commtypes.Serde,
		topicName string,
		substreamId uint8,
	) error {
		err := tm1.AddTopicSubstream(ctx, topicName, substreamId)
		if err != nil {
			return err
		}
		return err
	}
	return tm1, trackParFunc1
}
*/

func (h *produceConsumeHandler) beginTransaction(
	tm *transaction.TransactionManager, stream1 *sharedlog_stream.ShardedSharedLogStream,
) {
	tm.AddTopicSubstream(stream1.TopicName(), 0)
}

func (h *produceConsumeHandler) testMultiProducer2pc(ctx context.Context) {
	// two producer push to the same stream
	default_buf_max := uint32(32 * 1024)
	stream1, err := sharedlog_stream.NewShardedSharedLogStream("test", 1, commtypes.JSON, default_buf_max)
	if err != nil {
		panic(err)
	}
	stream1Copy, err := sharedlog_stream.NewShardedSharedLogStream("test", 1, commtypes.JSON, default_buf_max)
	if err != nil {
		panic(err)
	}

	msgSerdes, err := commtypes.GetMsgGSerdeG[int, string](commtypes.JSON, commtypes.IntSerdeG{}, commtypes.StringSerdeG{})
	if err != nil {
		panic(err)
	}
	produceSinkConfig := &producer_consumer.StreamSinkConfig{
		Format:        commtypes.JSON,
		FlushDuration: common.FlushDuration,
	}
	sp := &common.QueryInput{
		AppId:         "prodConsume",
		GuaranteeMth:  uint8(exactly_once_intr.TWO_PHASE_COMMIT),
		CommitEveryMs: 5,
		FlushMs:       5,
		WarmupS:       0,
		ParNum:        0,
		ScaleEpoch:    1,
	}

	produceSink := producer_consumer.NewShardedSharedLogStreamProducer(stream1, produceSinkConfig)
	produceSinkCopy := producer_consumer.NewShardedSharedLogStreamProducer(stream1Copy, produceSinkConfig)

	tm1 := getProduceTransactionManager(ctx, "prod1", produceSink, sp)
	tm2 := getProduceTransactionManager(ctx, "prod2", produceSinkCopy, sp)
	tm1.RecordTopicStreams(stream1.TopicName(), stream1)
	tm2.RecordTopicStreams(stream1Copy.TopicName(), stream1Copy)
	produceSink.ConfigExactlyOnce(tm1, exactly_once_intr.TWO_PHASE_COMMIT)
	produceSinkCopy.ConfigExactlyOnce(tm2, exactly_once_intr.TWO_PHASE_COMMIT)

	h.beginTransaction(tm1, stream1)
	h.beginTransaction(tm2, stream1Copy)

	// producer1 push 1 msg to sink
	msgForTm1 := []commtypes.MessageG[int, string]{
		{Key: optional.Some(1), Value: optional.Some("tm1_a")},
	}
	err = pushMsgsToSink(ctx, produceSink, msgForTm1, msgSerdes)
	if err != nil {
		panic(err)
	}
	produceSink.Flush(ctx)

	// then producer2 produce 1 msg to sink
	msgForTm2 := []commtypes.MessageG[int, string]{
		{Key: optional.Some(2), Value: optional.Some("tm2_a")},
	}
	err = pushMsgsToSink(ctx, produceSinkCopy, msgForTm2, msgSerdes)
	if err != nil {
		panic(err)
	}
	produceSinkCopy.Flush(ctx)

	// producer1 push 1 msg to sink
	msgForTm1 = []commtypes.MessageG[int, string]{
		{Key: optional.Some(3), Value: optional.Some("tm1_b")},
	}
	err = pushMsgsToSink(ctx, produceSink, msgForTm1, msgSerdes)
	if err != nil {
		panic(err)
	}
	produceSink.Flush(ctx)

	// producer1 commits
	if _, _, err = tm1.CommitTransaction(ctx); err != nil {
		panic(err)
	}

	// producer2 outputs two message
	msgForTm2 = []commtypes.MessageG[int, string]{
		{Key: optional.Some(4), Value: optional.Some("tm2_b")},
		{Key: optional.Some(5), Value: optional.Some("tm2_c")},
	}
	err = pushMsgsToSink(ctx, produceSinkCopy, msgForTm2, msgSerdes)
	if err != nil {
		panic(err)
	}
	produceSinkCopy.Flush(ctx)

	// producer2 commits
	if _, _, err = tm2.CommitTransaction(ctx); err != nil {
		panic(err)
	}

	stream1ForRead, err := sharedlog_stream.NewShardedSharedLogStream("test", 1, commtypes.JSON, default_buf_max)
	if err != nil {
		panic(err)
	}

	srcConfig := &producer_consumer.StreamConsumerConfig{
		Timeout:     common.SrcConsumeTimeout,
		SerdeFormat: commtypes.JSON,
	}

	src1, err := producer_consumer.NewShardedSharedLogStreamConsumer(stream1ForRead, srcConfig, 1, 0)
	if err != nil {
		panic(err)
	}
	src1.ConfigExactlyOnce(exactly_once_intr.TWO_PHASE_COMMIT)
	got, err := readMsgs[int, string](ctx, msgSerdes, commtypes.JSON, stream1ForRead)
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
