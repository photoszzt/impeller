package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/benchmark/common/kafka_utils"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	FLAGS_broker      string
	FLAGS_port        int
	FLAGS_input_fname string
)

func main() {
	flag.StringVar(&FLAGS_broker, "broker", "127.0.0.1", "")
	flag.StringVar(&FLAGS_input_fname, "input", "", "events json file")
	flag.IntVar(&FLAGS_port, "port", 8080, "port to listen")
	flag.Parse()

	if FLAGS_input_fname == "" {
		fmt.Fprintf(os.Stderr, "must specify input events payload file name\n")
		return
	}

	spec_bytes, err := os.ReadFile(FLAGS_input_fname)
	if err != nil {
		panic(err)
	}
	events := ntypes.Events{}
	err = json.Unmarshal(spec_bytes, &events)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	topic := "nexmark_src"
	newTopic := kafka_utils.CreateTopicSpecification(topic, 1)
	err = kafka_utils.CreateTopic(ctx, newTopic, FLAGS_broker)
	if err != nil {
		panic(fmt.Sprintf("Failed to create topic: %s", err))
	}

	p, err := kafka_utils.CreateProducer(FLAGS_broker, 5)
	if err != nil {
		panic(fmt.Sprintf("Failed to create producer: %s\n", err))
	}
	defer p.Close()
	eventSerde, err := ntypes.GetEventSerde(commtypes.JSON)
	if err != nil {
		panic(err)
	}

	handler := func(w http.ResponseWriter, req *http.Request) {
		replies := int32(0)
		stats_arr := make([]string, 0, 128)
		go kafka_utils.ProcessReturnEvents(p, &replies, stats_arr)
		for _, event := range events.EventsArr {
			encoded, _, err := eventSerde.Encode(&event)
			if err != nil {
				panic(err)
			}
			p.ProduceChannel() <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 0},
				Value:          encoded,
			}
		}
		kafka_utils.FlushAndWait(p, &replies)
		fmt.Fprintf(w, "done test_kproduce")
	}
	http.HandleFunc("/test_kproduce", handler)
	fmt.Fprintf(os.Stderr, "test_kproduce listening %d\n", FLAGS_port)
	_ = http.ListenAndServe(fmt.Sprintf(":%d", FLAGS_port), nil)
}
