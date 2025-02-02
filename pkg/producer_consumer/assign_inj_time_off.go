//go:build !stats
// +build !stats

package producer_consumer

import (
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/stats"
)

func assignInjTime(msg *commtypes.MessageSerialized) {
}

func extractProduceToConsumeTime(*commtypes.Message, bool, *stats.PrintLogStatsCollector[int64]) error {
	return nil
}

func extractProduceToConsumeTimeMsgG[K, V any](*commtypes.MessageG[K, V], bool, *stats.PrintLogStatsCollector[int64]) {
}
