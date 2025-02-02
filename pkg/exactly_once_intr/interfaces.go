//go:generate stringer -type=GuaranteeMth
package exactly_once_intr

import (
	"context"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
)

type GuaranteeMth uint8

const (
	NO_GUARANTEE GuaranteeMth = iota
	AT_LEAST_ONCE
	TWO_PHASE_COMMIT
	REMOTE_2PC
	EPOCH_MARK
	ALIGN_CHKPT
)

type ReadOnlyExactlyOnceManager interface {
	GetCurrentEpoch() uint32
	GetCurrentTaskId() uint64
	GetProducerId() commtypes.ProducerId
}

type ExactlyOnceManagerLogMonitor interface {
	ErrChan() chan error
	SendQuit()
	// StartMonitorLog(ctx context.Context, cancel context.CancelFunc)
}

type TopicSubstreamTracker interface {
	AddTopicSubstream(name string, subNum uint8)
}

type TrackProdSubStreamFunc func(
	topicName string,
	substreamId uint8,
)

type FlushCallbackFunc func(ctx context.Context) error

func DefaultTrackProdSubstreamFunc(
	topicName string,
	substreamId uint8,
) {
}

type RecordPrevInstanceFinishFunc func(ctx context.Context, appId string, instanceID uint8) error

func DefaultRecordPrevInstanceFinishFunc(ctx context.Context,
	appId string, instanceId uint8,
) error {
	debug.Fprintf(os.Stderr, "Running empty default record finish func\n")
	return nil
}
