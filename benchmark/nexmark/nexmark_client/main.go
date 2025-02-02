package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	FLAGS_faas_gateway    string
	FLAGS_engine_1        string
	FLAGS_app_name        string
	FLAGS_duration        int
	FLAGS_events_num      int
	FLAGS_tps             int
	FLAGS_serdeFormat     string
	FLAGS_workload_config string
	FLAGS_guarantee       string
	FLAGS_commit_everyMs  uint64
	FLAGS_snapshot_everyS uint
	FLAGS_buf_max_size    uint
	FLAGS_stat_dir        string
	FLAGS_warmup_time     int
	FLAGS_local           bool
	FLAGS_flush_ms        int
	FLAGS_src_flush_ms    int
	FLAGS_additionalBytes int
	FLAGS_dump_dir        string
	FLAGS_test_src        string
	FLAGS_fail_spec       string
	FLAGS_waitForEndMark  bool
)

type invokeSource func(client *http.Client, srcInvokeConfig common.SrcInvokeConfig, response *common.FnOutput, wg *sync.WaitGroup, warmup bool)

func invokeTestSrcFunc(client *http.Client, srcInvokeConfig common.SrcInvokeConfig,
	response *common.FnOutput, wg *sync.WaitGroup, warmup bool,
) {
	defer wg.Done()
	serdeFormat := common.GetSerdeFormat(FLAGS_serdeFormat)
	src := common.TestSourceInput{
		FileName:    FLAGS_test_src,
		SerdeFormat: uint8(serdeFormat),
	}
	url := common.BuildFunctionUrl(FLAGS_faas_gateway, "testSrc")
	fmt.Printf("func source url is %v\n", url)
	if err := common.JsonPostRequest(client, url, srcInvokeConfig.NodeConstraint, &src, response); err != nil {
		log.Error().Err(err).Msg("source request failed")
	} else if !response.Success {
		log.Error().Msgf("source request failed: %s", response.Message)
	}
	fmt.Fprintf(os.Stderr, "source-%d invoke done\n", srcInvokeConfig.InstanceID)
}

func main() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.StringVar(&FLAGS_engine_1, "engine1", "127.0.0.1:6060", "url for checkpt mngr on engine 1")
	flag.StringVar(&FLAGS_app_name, "app_name", "q1", "")
	flag.StringVar(&FLAGS_serdeFormat, "serde", "json", "serde format: json or msgp")
	flag.StringVar(&FLAGS_workload_config, "wconfig", "./wconfig.json", "path to a json file that stores workload config")
	flag.StringVar(&FLAGS_stat_dir, "stat_dir", "", "stats dir to dump")
	flag.StringVar(&FLAGS_dump_dir, "dumpdir", "", "output dir for dumps")
	flag.StringVar(&FLAGS_test_src, "testsrc", "", "test event source file")
	flag.StringVar(&FLAGS_fail_spec, "fail_spec", "", "fail spec")

	flag.IntVar(&FLAGS_duration, "duration", 60, "")
	flag.IntVar(&FLAGS_events_num, "events_num", 100000000, "events.num param for nexmark")
	flag.IntVar(&FLAGS_tps, "tps", 10000000, "tps param for nexmark")
	flag.IntVar(&FLAGS_warmup_time, "warmup_time", 0, "warmup time")
	flag.IntVar(&FLAGS_flush_ms, "flushms", 10, "flush the buffer every ms; for exactly once, please see commit_everyMs and commit_niter. They determine the flush interval. ")
	flag.IntVar(&FLAGS_src_flush_ms, "src_flushms", 5, "src flush ms")
	flag.IntVar(&FLAGS_additionalBytes, "extraBytes", 0, "additional bytes for bid, auc and person records")
	flag.UintVar(&FLAGS_snapshot_everyS, "snapshot_everyS", 0, "snapshot every s")
	flag.UintVar(&FLAGS_buf_max_size, "buf_max_size", 131072, "sink buffer max size")

	flag.Uint64Var(&FLAGS_commit_everyMs, "comm_everyMS", 10, "commit a transaction every (ms)")

	flag.StringVar(&FLAGS_guarantee, "guarantee", "alo",
		"none(no protocol), alo(at least once epoch), align_chkpt(aligned checkpoint), 2pc(two phase commit), remote_2pc(2 phase commit that talks to txn manager) or epoch(epoch marking)")
	flag.BoolVar(&FLAGS_local, "local", false, "local mode without setting node constraint")
	flag.BoolVar(&FLAGS_waitForEndMark, "waitForLast", false, "wait for the final mark of input; used in measuring throughput")

	flag.Parse()

	if FLAGS_stat_dir == "" {
		panic("should specify non empty stats dir")
	}
	if FLAGS_guarantee != "alo" && FLAGS_guarantee != "2pc" &&
		FLAGS_guarantee != "epoch" && FLAGS_guarantee != "none" &&
		FLAGS_guarantee != "align_chkpt" && FLAGS_guarantee != "remote_2pc" {
		fmt.Fprintf(os.Stderr, "expected guarantee is none, alo, 2pc and epoch")
		return
	}
	ctx := context.Background()
	fmt.Fprintf(os.Stderr, "wait for last: %v, sink max_buf_size: %v\n", FLAGS_waitForEndMark, FLAGS_buf_max_size)
	switch FLAGS_app_name {
	case "q1", "q2", "q3", "q4", "q5", "q6", "q7", "q8", "windowedAvg", "fanout":
		fixedMaxDur := uint32(0)
		serdeFormat := common.StringToSerdeFormat(FLAGS_serdeFormat)
		baseQueryInput := NewQueryInput(serdeFormat)
		gp := ntypes.GeneratorParams{
			EventsNum:      uint64(FLAGS_events_num),
			SerdeFormat:    serdeFormat,
			FaasGateway:    FLAGS_faas_gateway,
			Engine1:        FLAGS_engine_1,
			Duration:       uint32(FLAGS_duration),
			Tps:            uint32(FLAGS_tps),
			FlushMs:        uint32(FLAGS_src_flush_ms),
			CommitMs:       uint32(FLAGS_commit_everyMs),
			WaitForEndMark: FLAGS_waitForEndMark,
			BufMaxSize:     uint32(FLAGS_buf_max_size),
			GuaranteeMth:   baseQueryInput.GuaranteeMth,
		}
		var invokeSourceFunc_ invokeSource
		if FLAGS_test_src != "" {
			invokeSourceFunc_ = invokeTestSrcFunc
		} else {
			invokeSourceFunc_ = gp.InvokeSourceFunc
		}
		invokeFuncParam := common.InvokeFuncParam{
			ConfigFile:     FLAGS_workload_config,
			StatDir:        FLAGS_stat_dir,
			GatewayUrl:     FLAGS_faas_gateway,
			WarmupTime:     FLAGS_warmup_time,
			Local:          FLAGS_local,
			WaitForEndMark: FLAGS_waitForEndMark,
		}
		if FLAGS_app_name == "q7" {
			fixedMaxDur = uint32(450)
		}
		params, err := common.Invoke(ctx, invokeFuncParam, baseQueryInput, common.InvokeSrcFunc(invokeSourceFunc_),
			FLAGS_additionalBytes, fixedMaxDur)
		if err != nil {
			panic(err)
		}
		if FLAGS_dump_dir != "" {
			fmt.Fprintf(os.Stderr, "dumping log\n")
			err = os.MkdirAll(FLAGS_dump_dir, 0750)
			if err != nil {
				panic(err)
			}
			client := &http.Client{
				Transport: &http.Transport{
					IdleConnTimeout: 30 * time.Second,
				},
				Timeout: time.Duration(FLAGS_duration) * time.Second,
			}
			common.InvokeDumpFunc(client, FLAGS_dump_dir, FLAGS_app_name,
				common.GetSerdeFormat(FLAGS_serdeFormat), FLAGS_faas_gateway, params.ConfigScaleInput.Config)
		}
	default:
		panic("unrecognized app")
	}
}
