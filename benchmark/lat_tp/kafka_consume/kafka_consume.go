package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/pkg/stats"
	"sort"
	"time"

	datatype "sharedlog-stream/benchmark/lat_tp/pkg/data_type"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	FLAGS_broker       string
	FLAGS_topicName    string
	FLAGS_duration     int
	FLAGS_events_num   int
	FLAGS_warmup       int
	FLAGS_warmupEvents int
)

func main() {
	flag.StringVar(&FLAGS_broker, "broker", "127.0.0.1", "")
	flag.IntVar(&FLAGS_events_num, "events_num", 100000000, "number of events")
	flag.StringVar(&FLAGS_topicName, "topicName", "src", "topic name")
	flag.IntVar(&FLAGS_duration, "duration", 80, "")
	flag.IntVar(&FLAGS_warmup, "warmup_time", 0, "warm up time in sec")
	flag.IntVar(&FLAGS_warmupEvents, "warmup_events", 0, "number of events consumed for warmup")
	flag.Parse()
	if FLAGS_duration < FLAGS_warmup {
		fmt.Fprintf(os.Stderr, "warm up duration(%d) should be smaller then total duration(%d)\n", FLAGS_warmup, FLAGS_duration)
		os.Exit(1)
	}

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":      FLAGS_broker,
		"group.id":               "bench",
		"auto.offset.reset":      "earliest",
		"fetch.wait.max.ms":      5,
		"fetch.error.backoff.ms": 5,
		"enable.auto.commit":     false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	err = c.SubscribeTopics([]string{FLAGS_topicName}, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fail to subscribe to topic: %s\n", err)
		os.Exit(1)
	}
	commitEvery := time.Duration(100) * time.Millisecond
	duration := time.Duration(FLAGS_duration) * time.Second
	handleConsume := func(w http.ResponseWriter, req *http.Request) {
		var ptSerde datatype.PayloadTsMsgpSerde
		prod_to_con_lat := make([]int64, 0, 4096)
		// warmup first
		rest := FLAGS_events_num
		if FLAGS_warmup > 0 && FLAGS_warmupEvents > 0 {
			fmt.Fprintf(os.Stdout, "begin warmup")
			idx_consumed, err := runLoop(time.Duration(FLAGS_warmup)*time.Second, FLAGS_warmupEvents, c, commitEvery)
			if err != nil {
				panic(err)
			}
			rest = FLAGS_events_num - idx_consumed
			fmt.Fprintf(os.Stdout, "down warmup")
		}
		idx := 0
		commitTimer := time.NewTicker(commitEvery)
		start := time.Now()

		hasUncommitted := false
		for {
			select {
			case <-commitTimer.C:
				_, _ = c.Commit()
				hasUncommitted = false
			default:
			}
			if (duration > 0 && time.Since(start) > duration) ||
				(rest > 0 && idx >= rest) {
				commitTimer.Stop()
				break
			}
			ev := c.Poll(5)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				ptTmp, err := ptSerde.Decode(e.Value)
				if err != nil {
					panic(err)
				}
				pt := ptTmp.(datatype.PayloadTs)
				nowTs := time.Now().UnixMicro()
				lat := nowTs - pt.Ts
				prod_to_con_lat = append(prod_to_con_lat, lat)
				idx += 1
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
			}
		}
		if hasUncommitted {
			_, err := c.Commit()
			if err != nil {
				panic(err)
			}
		}
		totalTime := time.Since(start).Seconds()
		fmt.Fprintf(os.Stderr, "\n%v\n", prod_to_con_lat)
		ts := stats.Int64Slice(prod_to_con_lat)
		sort.Sort(ts)
		fmt.Fprintf(os.Stdout, "consumed %d events, time: %v, throughput: %v, p50: %d, p99: %d\n",
			idx, totalTime, float64(idx)/float64(totalTime), stats.P(ts, 0.5), stats.P(ts, 0.99))
		fmt.Fprint(w, "done consume")
	}
	http.HandleFunc("/consume", handleConsume)
	_ = http.ListenAndServe(":8090", nil)
}

func runLoop(duration time.Duration, warmup_events int, c *kafka.Consumer, commitEvery time.Duration) (int, error) {
	start := time.Now()
	commitTimer := time.Now()
	idx := 0
	for {
		if (duration > 0 && time.Since(start) > duration) ||
			(warmup_events > 0 && idx > warmup_events) {
			break
		}
		ev := c.Poll(5)
		if ev == nil {
			continue
		}

		switch ev.(type) {
		case *kafka.Message:
			if time.Since(commitTimer) >= commitEvery {
				_, err := c.Commit()
				if err != nil {
					return idx, err
				}
				commitTimer = time.Now()
			}
			idx += 1
		}
	}
	_, err := c.Commit()
	return idx, err
}
