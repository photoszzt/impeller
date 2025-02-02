package kafka_utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// CreateTransactionalProducer creates a transactional producer for the given
// input partition.
func CreateTransactionalProducerNoBatching(broker string,
	toppar kafka.TopicPartition,
) (*kafka.Producer, error) {
	producerConfig := &kafka.ConfigMap{
		"client.id":          fmt.Sprintf("txn-p%d", toppar.Partition),
		"bootstrap.servers":  broker,
		"transactional.id":   fmt.Sprintf("tran-p%d", int(toppar.Partition)),
		"acks":               "all",
		"batch.num.messages": 1,
		"batch.size":         1,
		"linger.ms":          0,

		"max.in.flight.requests.per.connection": 5,
	}

	producer, err := kafka.NewProducer(producerConfig)
	if err != nil {
		return nil, err
	}

	maxDuration, err := time.ParseDuration("10s")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()

	initBeg := time.Now()
	err = producer.InitTransactions(ctx)
	if err != nil {
		return nil, err
	}
	initTranElapsed := time.Since(initBeg)
	fmt.Fprintf(os.Stderr, "initTransaction elapsed: %d us\n",
		initTranElapsed.Microseconds())

	err = producer.BeginTransaction()
	if err != nil {
		return nil, err
	}
	return producer, nil
}

// DestroyTransactionalProducer aborts the current transaction and destroys the producer.
func DestroyTransactionalProducer(producer *kafka.Producer) error {
	maxDuration, err := time.ParseDuration("10s")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()

	err = producer.AbortTransaction(ctx)
	if err != nil {
		if err.(kafka.Error).Code() == kafka.ErrState {
			// No transaction in progress, ignore the error.
			err = nil
		} else {
			fmt.Fprintf(os.Stderr, "Failed to abort transaction for %v: %v",
				producer, err)
		}
	}

	producer.Close()

	return err
}

// groupRebalance is triggered on consumer rebalance.
//
// For each assigned partition a transactional producer is created, this is
// required to guarantee per-partition offset commit state prior to
// KIP-447 being supported.
func GroupRebalance(broker string, producers *map[int32]*kafka.Producer,
	consumer *kafka.Consumer, event kafka.Event,
) error {
	fmt.Fprintf(os.Stderr, "Processor: rebalance event %v", event)

	switch e := event.(type) {
	case kafka.AssignedPartitions:
		// Create a producer per input partition.
		for _, tp := range e.Partitions {
			p, err := CreateTransactionalProducerNoBatching(broker, tp)
			if err != nil {
				return err
			}
			(*producers)[tp.Partition] = p
		}

		err := consumer.Assign(e.Partitions)
		if err != nil {
			return err
		}

	case kafka.RevokedPartitions:
		// Abort any current transactions and close the
		// per-partition producers.
		for _, producer := range *producers {
			err := DestroyTransactionalProducer(producer)
			if err != nil {
				return err
			}
		}

		// Clear producer and intersection states
		*producers = make(map[int32]*kafka.Producer)

		err := consumer.Unassign()
		if err != nil {
			return err
		}
	}

	return nil
}

// rewindConsumerPosition rewinds the consumer to the last committed offset or
// the beginning of the partition if there is no committed offset.
// This is to be used when the current transaction is aborted.
func RewindConsumerPosition(consumer *kafka.Consumer, topic string, partition int32) error {
	committed, err := consumer.Committed([]kafka.TopicPartition{{Topic: &topic, Partition: partition}}, 10*1000 /* 10s */)
	if err != nil {
		return err
	}

	for _, tp := range committed {
		if tp.Offset < 0 {
			// No committed offset, reset to earliest
			tp.Offset = kafka.OffsetBeginning
		}

		fmt.Fprintf(os.Stderr, "Processor: rewinding input partition %v to offset %v",
			tp.Partition, tp.Offset)

		err = consumer.Seek(tp, -1)
		if err != nil {
			return err
		}
	}
	return nil
}

// getConsumerPosition gets the current position (next offset) for a given input partition.
func GetConsumerPosition(consumer *kafka.Consumer, topic string, partition int32,
) ([]kafka.TopicPartition, error) {
	position, err := consumer.Position([]kafka.TopicPartition{{Topic: &topic, Partition: partition}})
	if err != nil {
		return nil, err
	}

	return position, nil
}

// commitTransactionForInputPartition sends the consumer offsets for
// the given input partition and commits the current transaction.
// A new transaction will be started when done.
func CommitTransactionForInputPartition(ctx context.Context, producers map[int32]*kafka.Producer,
	consumer *kafka.Consumer, inputTopic string, partition int32,
) error {
	producer, found := producers[partition]
	if !found || producer == nil {
		return fmt.Errorf("BUG: No producer for input partition %v", partition)
	}

	position, err := GetConsumerPosition(consumer, inputTopic, partition)
	if err != nil {
		return err
	}
	consumerMetadata, err := consumer.GetConsumerGroupMetadata()
	if err != nil {
		return fmt.Errorf("Failed to get consumer group metadata: %v", err)
	}

	err = producer.SendOffsetsToTransaction(ctx, position, consumerMetadata)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Processor: Failed to send offsets to transaction for input partition %v: %s: aborting transaction",
			partition, err)

		err = producer.AbortTransaction(ctx)
		if err != nil {
			panic(err)
		}

		// Rewind this input partition to the last committed offset.
		err = RewindConsumerPosition(consumer, inputTopic, partition)
		if err != nil {
			panic(err)
		}
	} else {
		err = producer.CommitTransaction(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"Processor: Failed to commit transaction for input partition %v: %s",
				partition, err)

			err = producer.AbortTransaction(ctx)
			if err != nil {
				panic(err)
			}

			// Rewind this input partition to the last committed offset.
			err = RewindConsumerPosition(consumer, inputTopic, partition)
			if err != nil {
				panic(err)
			}
		}
	}

	// Start a new transaction
	err = producer.BeginTransaction()
	if err != nil {
		return err
	}
	return nil
}
