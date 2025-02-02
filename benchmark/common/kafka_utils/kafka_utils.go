package kafka_utils

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rs/zerolog/log"
)

func ProcessReturnEvents(p *kafka.Producer, replies *int32, stats_arr []string) {
	for e := range p.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Error().Msgf("Delivery failed: %v\n", ev.TopicPartition)
			} else {
				log.Debug().Msgf("Delivered message to %v, ts %v\n", ev.TopicPartition, ev.Timestamp)
			}
		case *kafka.Stats:
			stats_arr = append(stats_arr, ev.String())
		default:
		}
		atomic.AddInt32(replies, 1)
	}
}

func CreateProducer(broker string, flushMs int) (*kafka.Producer, error) {
	return kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":                     broker,
		"go.produce.channel.size":               100000,
		"go.events.channel.size":                100000,
		"acks":                                  "all",
		"batch.size":                            131072,
		"linger.ms":                             flushMs,
		"max.in.flight.requests.per.connection": 5,
		// "statistics.interval.ms":                5000,
	})
}

func CreateProducerNoBatching(broker string) (*kafka.Producer, error) {
	return kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":                     broker,
		"go.produce.channel.size":               100000,
		"go.events.channel.size":                100000,
		"acks":                                  "all",
		"batch.num.messages":                    1,
		"batch.size":                            1,
		"linger.ms":                             0,
		"max.in.flight.requests.per.connection": 5,
		// "statistics.interval.ms":                5000,
	})
}

func FlushAndWait(p *kafka.Producer, replies *int32) {
	remaining := p.Flush(30 * 1000)
	for remaining != 0 {
		remaining = p.Flush(30 * 1000)
	}
	ret := atomic.LoadInt32(replies)
	fmt.Fprintf(os.Stderr, "%d event acked\n", ret)
}

func CreateTopic(ctx context.Context, topics []kafka.TopicSpecification, bootstrapServer string) error {
	conf := kafka.ConfigMap{"bootstrap.servers": bootstrapServer}
	adminClient, err := kafka.NewAdminClient(&conf)
	if err != nil {
		return err
	}
	result, err := adminClient.CreateTopics(ctx, topics)
	if err != nil {
		return err
	}
	for _, res := range result {
		switch res.Error.Code() {
		case kafka.ErrTopicAlreadyExists:
			log.Error().Msgf("Failed to create topic %s: %v", res.Topic, res.Error)
		case kafka.ErrNoError:
			log.Error().Msgf("Succeed to create topic %s", res.Topic)
		default:
			return fmt.Errorf("failed to create topic %s: %v", res.Topic, res.Error)
		}
	}
	return nil
}

func CreateTopicSpecification(topic string, numPartition int) []kafka.TopicSpecification {
	return []kafka.TopicSpecification{
		{
			Topic:             topic,
			NumPartitions:     numPartition,
			ReplicationFactor: 3,
			Config: map[string]string{
				"min.insync.replicas": "3",
			},
		},
	}
}
