package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/benchmark/common/kafka_utils"
	datatype "sharedlog-stream/benchmark/lat_tp/pkg/data_type"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	FLAGS_broker       string
	FLAGS_topicName    string
	FLAGS_numPartition int
	FLAGS_events_num   int
	FLAGS_duration     int
	FLAGS_tps          int
	FLAGS_payloadFile  string
)

func main() {
	flag.IntVar(&FLAGS_events_num, "events_num", 100000000, "number of events")
	flag.IntVar(&FLAGS_duration, "duration", 60, "")
	flag.StringVar(&FLAGS_broker, "broker", "127.0.0.1", "")
	flag.StringVar(&FLAGS_topicName, "topicName", "src", "topic name")
	flag.IntVar(&FLAGS_numPartition, "npar", 1, "number of partition")
	flag.StringVar(&FLAGS_payloadFile, "payload", "", "payload file name")
	flag.IntVar(&FLAGS_tps, "tps", 1000, "events per second")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "duration: %d, events_num: %d, broker: %s, topicName: %s, nPar: %d, payload: %s\n",
		FLAGS_duration, FLAGS_events_num, FLAGS_broker, FLAGS_topicName, FLAGS_numPartition,
		FLAGS_payloadFile)
	if FLAGS_payloadFile == "" {
		fmt.Fprintf(os.Stderr, "payload filename cannot be empty\n")
		return
	}
	content, err := os.ReadFile(FLAGS_payloadFile)
	if err != nil {
		panic(fmt.Sprintf("fail to read file: %v", err))
	}
	newTopic := kafka_utils.CreateTopicSpecification(FLAGS_topicName, FLAGS_numPartition)
	ctx := context.Background()
	err = kafka_utils.CreateTopic(ctx, newTopic, FLAGS_broker)
	if err != nil {
		panic(fmt.Sprintf("Failed to create topic: %s", err))
	}
	p, err := kafka_utils.CreateProducerNoBatching(FLAGS_broker)
	if err != nil {
		panic(fmt.Sprintf("Failed to create producer: %s\n", err))
	}
	defer p.Close()
	var ptSerde datatype.PayloadTsMsgpSerde
	duration := time.Duration(FLAGS_duration) * time.Second
	timeGapUs := time.Duration(1000000/FLAGS_tps) * time.Microsecond

	produceHandler := func(w http.ResponseWriter, req *http.Request) {
		idx := int32(0)
		replies := int32(0)
		stats_arr := make([]string, 0, 128)
		go kafka_utils.ProcessReturnEvents(p, &replies, stats_arr)
		start := time.Now()
		next := time.Now()
		events_num := int32(FLAGS_events_num)
		num_par := int32(FLAGS_numPartition)
		for {
			if (duration != 0 && time.Since(start) >= duration) ||
				(FLAGS_events_num != 0 && idx >= events_num) {
				break
			}
			parNum := idx % num_par
			next = next.Add(timeGapUs)
			payloadTs := datatype.PayloadTs{
				Payload: content,
				Ts:      next.UnixMicro(),
			}
			encoded, _, err := ptSerde.Encode(&payloadTs)
			if err != nil {
				panic(err)
			}
			now := time.Now()
			if next.After(now) {
				time.Sleep(next.Sub(now))
			}
			p.ProduceChannel() <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &FLAGS_topicName, Partition: int32(parNum)},
				Value:          encoded,
			}
			idx += 1
		}
		kafka_utils.FlushAndWait(p, &replies)
		totalTime := time.Since(start).Seconds()
		fmt.Fprintf(os.Stderr, "produce %d events, time: %v, throughput: %v\n",
			idx, totalTime, float64(idx)/totalTime)
		for _, s := range stats_arr {
			fmt.Fprintf(os.Stderr, s+"\n")
		}
		fmt.Fprint(w, "done produce")
	}
	http.HandleFunc("/produce", produceHandler)
	_ = http.ListenAndServe(":8080", nil)
}
