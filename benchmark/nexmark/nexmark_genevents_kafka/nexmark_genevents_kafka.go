package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/benchmark/common/kafka_utils"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/generator"
	"sharedlog-stream/pkg/commtypes"
	"strconv"
	"time"

	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rs/zerolog/log"
)

var (
	FLAGS_events_num      int
	FLAGS_duration        int
	FLAGS_broker          string
	FLAGS_stream_prefix   string
	FLAGS_serdeFormat     string
	FLAGS_numPartition    int
	FLAGS_tps             int
	FLAGS_srcInstance     int
	FLAGS_port            int
	FLAGS_flushms         int
	FLAGS_additionalBytes int
	FLAGS_disableBatching bool
)

func main() {
	flag.IntVar(&FLAGS_events_num, "events_num", 100000000, "events.num param for nexmark")
	flag.IntVar(&FLAGS_duration, "duration", 60, "")
	flag.StringVar(&FLAGS_broker, "broker", "127.0.0.1", "")
	flag.StringVar(&FLAGS_stream_prefix, "stream_prefix", "nexmark", "")
	flag.StringVar(&FLAGS_serdeFormat, "serde", "json", "serde format: json or msgp")
	flag.IntVar(&FLAGS_numPartition, "npar", 1, "number of partition")
	flag.IntVar(&FLAGS_tps, "tps", 10000000, "tps param for nexmark")
	flag.IntVar(&FLAGS_srcInstance, "srcIns", 1, "number of source instance")
	flag.IntVar(&FLAGS_port, "port", 8080, "port to listen")
	flag.IntVar(&FLAGS_flushms, "flushms", 100, "flush inverval in ms")
	flag.BoolVar(&FLAGS_disableBatching, "disableBatching", false, "disable batching")
	flag.IntVar(&FLAGS_additionalBytes, "extraBytes", 0, "additional bytes")
	flag.Parse()

	var serdeFormat commtypes.SerdeFormat
	var valueEncoder commtypes.SerdeG[*ntypes.Event]
	if FLAGS_serdeFormat == "json" {
		serdeFormat = commtypes.JSON
		valueEncoder = ntypes.EventJSONSerdeG{}
	} else if FLAGS_serdeFormat == "msgp" {
		serdeFormat = commtypes.MSGP
		valueEncoder = ntypes.EventMsgpSerdeG{}
	} else {
		log.Error().Msgf("serde format is not recognized; default back to JSON")
		serdeFormat = commtypes.JSON
		valueEncoder = ntypes.EventJSONSerdeG{}
	}

	iid_str := os.Getenv("IID")
	instanceId := 0
	var err error
	if iid_str != "" {
		instanceId, err = strconv.Atoi(iid_str)
		if err != nil {
			panic(err)
		}
	}
	aucBytes := ntypes.DEFAULT_AVG_AUC_SIZE + FLAGS_additionalBytes
	bidBytes := ntypes.DEFAULT_AVG_BID_SIZE + FLAGS_additionalBytes
	personBytes := ntypes.DEFAULT_AVG_PERSON_SIZE + FLAGS_additionalBytes
	fmt.Fprintf(os.Stderr, "duration: %d, events_nm: %d, serde: %s, nPar: %d, sourceInstances: %d, "+
		"tps: %d, port: %d, flushMs: %d, instanceID: %d, avgAucBytes: %d, avgBidBytes: %d, avgPersonBytes: %d\n",
		FLAGS_duration, FLAGS_events_num, FLAGS_serdeFormat, FLAGS_numPartition,
		FLAGS_srcInstance, FLAGS_tps, FLAGS_port, FLAGS_flushms, instanceId,
		aucBytes, bidBytes, personBytes)

	ctx := context.Background()

	topic := FLAGS_stream_prefix + "_src"
	newTopic := kafka_utils.CreateTopicSpecification(topic, FLAGS_numPartition)
	err = kafka_utils.CreateTopic(ctx, newTopic, FLAGS_broker)
	if err != nil {
		log.Fatal().Msgf("Failed to create topic: %s", err)
	}
	nexmarkConfigInput := ntypes.NewNexMarkConfigInput(topic, serdeFormat)
	nexmarkConfigInput.Duration = uint32(FLAGS_duration)
	nexmarkConfigInput.FirstEventRate = uint32(FLAGS_tps)
	nexmarkConfigInput.NextEventRate = uint32(FLAGS_tps)
	nexmarkConfigInput.EventsNum = uint64(FLAGS_events_num)
	nexmarkConfigInput.NumOutPartition = uint8(FLAGS_numPartition)
	nexmarkConfigInput.NumSrcInstance = uint8(FLAGS_srcInstance)
	nexmarkConfigInput.BidAvgSize = uint32(bidBytes)
	nexmarkConfigInput.PersonAvgSize = uint32(personBytes)
	nexmarkConfigInput.AuctionAvgSize = uint32(aucBytes)
	nexmarkConfig, err := ntypes.ConvertToNexmarkConfiguration(nexmarkConfigInput)
	if err != nil {
		log.Fatal().Msgf("Failed to convert to nexmark configuration: %s", err)
	}

	var p *kafka.Producer
	if FLAGS_disableBatching {
		p, err = kafka_utils.CreateProducerNoBatching(FLAGS_broker)
		if err != nil {
			log.Fatal().Msgf("Failed to create producer: %s\n", err)
		}
	} else {
		p, err = kafka_utils.CreateProducer(FLAGS_broker, FLAGS_flushms)
		if err != nil {
			log.Fatal().Msgf("Failed to create producer: %s\n", err)
		}
	}
	defer p.Close()
	duration := time.Duration(FLAGS_duration) * time.Second
	num_par := int32(FLAGS_numPartition)
	parNum := instanceId % int(num_par)

	handler := func(w http.ResponseWriter, req *http.Request) {
		generatorConfig := generator.NewGeneratorConfig(nexmarkConfig, time.Now().UnixMilli(), 1, uint64(nexmarkConfig.NumEvents), 1)
		eventGenerator := generator.NewSimpleNexmarkGenerator(generatorConfig, instanceId)
		eventsPerGen := uint64(FLAGS_events_num) / uint64(generatorConfig.Configuration.NumEventGenerators)
		fmt.Fprintf(os.Stderr, "eventsPerGen: %d, duration: %v\n", eventsPerGen, duration)
		channel_url_cache := make(map[uint32]*generator.ChannelUrl)

		idx := int32(0)
		replies := int32(0)
		stats_arr := make([]string, 0, 128)
		go kafka_utils.ProcessReturnEvents(p, &replies, stats_arr)
		start := time.Now()
		for {
			elapsed := time.Since(start)
			// fmt.Fprintf(os.Stderr, "elapsed: %v, idx: %v\n", elapsed, idx)
			if (duration != 0 && elapsed >= duration) ||
				(eventsPerGen != 0 && idx >= int32(eventsPerGen)) {
				break
			}
			nextEvent, err := eventGenerator.NextEvent(ctx, channel_url_cache)
			if err != nil {
				log.Fatal().Msgf("next event failed: %s", err)
			}
			encoded, _, err := valueEncoder.Encode(nextEvent.Event)
			if err != nil {
				log.Fatal().Msgf("event serialization failed: %s", err)
			}
			nowMs := time.Now().UnixMilli()
			if nextEvent.WallclockTimestamp > nowMs {
				time.Sleep(time.Duration(nextEvent.WallclockTimestamp-nowMs) * time.Millisecond)
			}

			idx += 1
			p.ProduceChannel() <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: int32(parNum)},
				Value:          encoded,
			}
		}
		kafka_utils.FlushAndWait(p, &replies)
		totalTime := time.Since(start).Seconds()
		fmt.Fprintf(os.Stderr, "source produced %d events, time %v, throughput %v\n",
			idx, totalTime, float64(idx)/totalTime)
		for _, s := range stats_arr {
			fmt.Fprintf(os.Stderr, s+"\n")
		}
		fmt.Fprintf(w, "done kproduce")
	}
	http.HandleFunc("/kproduce", handler)
	fmt.Fprintf(os.Stderr, "kproduce listening %d\n", FLAGS_port)
	_ = http.ListenAndServe(fmt.Sprintf(":%d", FLAGS_port), nil)
}
