package store

import (
	"context"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/sharedlog_stream"
)

type StoreBackedByChangelog interface {
	Flush(ctx context.Context) (uint32, error)
	ConfigureExactlyOnce(rem exactly_once_intr.ReadOnlyExactlyOnceManager,
		guarantee exactly_once_intr.GuaranteeMth)
	ChangelogTopicName() string
	Stream() sharedlog_stream.Stream
	SetStreamFlushCallbackFunc(cb exactly_once_intr.FlushCallbackFunc)
	OutputRemainingStats()
}

type RestoreFromChangelog interface {
	ConsumeOneLogEntry(ctx context.Context, parNum uint8) (int, error)
}
