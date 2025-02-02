//go:build stats
// +build stats

package producer_consumer

import (
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/stats"
	"sharedlog-stream/pkg/txn_data"
	"time"
)

func MsgIsScaleFence(msg *commtypes.Message) bool {
	ctrl, ok := msg.Key.(string)
	return ok && ctrl == txn_data.SCALE_FENCE_KEY
}

func assignInjTime(msg *commtypes.MessageSerialized) {
	nowMs := time.Now().UnixMilli()
	msg.UpdateInjectTime(nowMs)
}

func extractProduceToConsumeTime(msg *commtypes.Message, isInitialSrc bool, collector *stats.PrintLogStatsCollector[int64]) {
	if !isInitialSrc {
		ts := msg.ExtractInjectTimeMs()
		dur := time.Now().UnixMilli() - ts
		collector.AddSample(dur)
	}
}

func extractProduceToConsumeTimeMsgG[K, V any](msg *commtypes.MessageG[K, V], isInitialSrc bool, collector *stats.PrintLogStatsCollector[int64]) {
	if !isInitialSrc {
		ts := msg.ExtractInjectTimeMs()
		dur := time.Now().UnixMilli() - ts
		collector.AddSample(dur)
	}
}
