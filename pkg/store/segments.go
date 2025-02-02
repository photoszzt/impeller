package store

import "context"

type Segments interface {
	SegmentId(ts int64) int64
	SegmentName(segmentId int64) string
	GetSegmentForTimestamp(ts int64) Segment
	GetOrCreateSegmentIfLive(ctx context.Context, segmentId int64,
		streamTime int64,
		getOrCreateSegment func(ctx context.Context, segmentId int64) (Segment, error),
		cleanupExpired func(ctx context.Context, expired []*KeySegment) error,
	) (Segment, error)
	GetOrCreateSegment(ctx context.Context, segmentId int64) (Segment, error)
	Segments(timeFrom int64, timeTo int64) []Segment
	CleanupExpiredMeta(ctx context.Context, expired []*KeySegment) error
	GetSegmentNamesFromRemote(ctx context.Context) ([]string, error)
	Destroy(ctx context.Context) error
	StartTransaction(ctx context.Context) error
	CommitTransaction(ctx context.Context, taskRepr string, transactionalID uint64) error
	AbortTransaction(ctx context.Context) error
	GetTransactionID(ctx context.Context, taskRepr string) (uint64, bool, error)
}
