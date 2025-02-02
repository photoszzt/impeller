package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/benchmark/common"
	"sharedlog-stream/benchmark/common/benchutil"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/store"
)

func NewQueryInput(serdeFormat commtypes.SerdeFormat) *common.QueryInput {
	table_type := store.IN_MEM
	fmt.Fprintf(os.Stderr, "warmup: %d\n", FLAGS_warmup_time)
	guarantee := benchutil.GetGuarantee(FLAGS_guarantee)
	var failSpec commtypes.FailSpec
	if FLAGS_fail_spec != "" {
		specBytes, err := os.ReadFile(FLAGS_fail_spec)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(specBytes, &failSpec)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "Fail spec is %+v\n", failSpec)
	}
	return &common.QueryInput{
		Duration:       uint32(FLAGS_duration),
		FaasGateway:    FLAGS_faas_gateway,
		Engine1:        FLAGS_engine_1,
		GuaranteeMth:   uint8(guarantee),
		CommitEveryMs:  FLAGS_commit_everyMs,
		SerdeFormat:    uint8(serdeFormat),
		AppId:          FLAGS_app_name,
		TableType:      uint8(table_type),
		FlushMs:        uint32(FLAGS_flush_ms),
		WarmupS:        uint32(FLAGS_warmup_time),
		TestParams:     failSpec.FailSpec,
		SnapEveryS:     uint32(FLAGS_snapshot_everyS),
		BufMaxSize:     uint32(FLAGS_buf_max_size),
		WaitForEndMark: FLAGS_waitForEndMark,
	}
}
