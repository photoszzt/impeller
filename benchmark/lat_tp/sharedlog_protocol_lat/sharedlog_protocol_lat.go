package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/benchmark/common"
	"sync"
	"time"
)

var (
	FLAGS_faas_gateway   string
	FLAGS_duration       int
	FLAGS_tps            int
	FLAGS_payload        string
	FLAGS_local          bool
	FLAGS_events_num     int
	FLAGS_npar           int
	FLAGS_nprod          int
	FLAGS_serdeFormat    string
	FLAGS_commit_everyMs uint64
	FLAGS_buf_max_size   uint
)

func main() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.IntVar(&FLAGS_duration, "duration", 60, "")
	flag.IntVar(&FLAGS_events_num, "events_num", 100000000, "events.num param for nexmark")
	flag.IntVar(&FLAGS_tps, "tps", 10000000, "tps param for nexmark")
	flag.StringVar(&FLAGS_payload, "payload", "", "payload path")
	flag.IntVar(&FLAGS_npar, "npar", 1, "number of partition")
	flag.IntVar(&FLAGS_nprod, "nprod", 1, "number of producer")
	flag.StringVar(&FLAGS_serdeFormat, "serde", "json", "serde format: json or msgp")
	flag.Uint64Var(&FLAGS_commit_everyMs, "comm_everyMS", 10, "commit a transaction every (ms)")
	flag.UintVar(&FLAGS_buf_max_size, "buf_max_size", 131072, "sink buffer max size")
	flag.BoolVar(&FLAGS_local, "local", false, "local mode without setting node constraint")

	flag.Parse()

	serdeFormat := common.StringToSerdeFormat(FLAGS_serdeFormat)
	spProd := &common.BenchSourceParam{
		TopicName:       "src",
		Duration:        uint32(FLAGS_duration),
		SerdeFormat:     uint8(serdeFormat),
		NumEvents:       uint32(FLAGS_events_num),
		FileName:        FLAGS_payload,
		NumOutPartition: uint8(FLAGS_npar),
		Tps:             uint32(FLAGS_tps),
		FlushMs:         uint32(FLAGS_commit_everyMs),
		BufMaxSize:      uint32(FLAGS_buf_max_size),
	}
	spTran := &common.TranProcessBenchParam{
		InTopicName:   "src",
		OutTopicName:  "out",
		SerdeFormat:   uint8(serdeFormat),
		NumPartition:  uint8(FLAGS_npar),
		Duration:      uint32(FLAGS_duration),
		CommitEveryMs: uint64(FLAGS_commit_everyMs),
		FlushMs:       uint32(FLAGS_commit_everyMs),
		AppId:         "protocol-lat",
		BufMaxSize:    uint32(FLAGS_buf_max_size),
		FaasGateway:   FLAGS_faas_gateway,
	}
	client := &http.Client{
		Transport: &http.Transport{
			IdleConnTimeout: 30 * time.Second,
		},
		Timeout: time.Duration(FLAGS_duration*2) * time.Second,
	}
	var wg sync.WaitGroup
	invoke := func(name string, response *common.FnOutput, sp interface{}, nodeConstraint string) {
		defer wg.Done()
		url := common.BuildFunctionUrl(FLAGS_faas_gateway, name)
		if err := common.JsonPostRequest(client, url, nodeConstraint, sp, response); err != nil {
			fmt.Fprintf(os.Stderr, "%s request failed: %v", name, err)
		} else if !response.Success {
			fmt.Fprintf(os.Stderr, "%s request failed: %s", name, response.Message)
		}
	}
	scaleParam := map[string]uint8{
		"nexmark_src": 1,
		"tran_out":    1,
		"tranDataGen": 1,
		"tranProcess": 1,
	}
	c := &common.ConfigScaleInput{
		Config:      scaleParam,
		AppId:       "protocol-lat",
		FuncNames:   []string{"tranDataGen", "tranProcess"},
		ScaleEpoch:  1,
		SerdeFormat: uint8(serdeFormat),
		Bootstrap:   true,
		BufMaxSize:  uint32(FLAGS_buf_max_size),
	}
	constraint := "1"
	if FLAGS_local {
		constraint = ""
	}
	var scaleResponse common.FnOutput
	wg.Add(1)
	go invoke("scale", &scaleResponse, c, constraint)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "done invoke initial scale\n")
	fmt.Fprintf(os.Stderr, "scale response: %#v\n", scaleResponse)
	prodCons := "1"
	if FLAGS_local {
		prodCons = ""
	}
	consumeCons := "2"
	if FLAGS_local {
		consumeCons = ""
	}
	var prodResponse common.FnOutput
	var consumeResponse common.FnOutput
	wg.Add(1)
	go invoke("tranDataGen", &prodResponse, spProd, prodCons)
	wg.Add(1)
	go invoke("tranProcess", &consumeResponse, spTran, consumeCons)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "tranDataGen: %#v\n", prodResponse)
	if consumeResponse.Success {
		fmt.Fprintf(os.Stderr, "tranProcess: counts: %#v, duration: %#v\n",
			consumeResponse.Counts, consumeResponse.Duration)
	} else {
		fmt.Fprintf(os.Stderr, "tranProcess failed: %s\n", consumeResponse.Message)
	}
}
