package store

import (
	"bytes"
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

type BaseSegmentedBytesStore struct {
	keySchema          KeySchema
	segments           Segments
	name               string
	observedStreamTime int64
	tableType          TABLE_TYPE
}

var _ = SegmentedBytesStore(&BaseSegmentedBytesStore{})

/*
func NewRedisSegmentedBytesStore(name string,
	retention int64, // ms
	keySchema KeySchema,
	rkvs *RedisKeyValueStore,
) *BaseSegmentedBytesStore {
	segmentInterval := retention / 2
	if segmentInterval < 60_000 {
		segmentInterval = 60_000
	}
	return &BaseSegmentedBytesStore{
		name:               name,
		keySchema:          keySchema,
		segments:           NewRedisKeyValueSegments(name, retention, segmentInterval, rkvs),
		observedStreamTime: -1,
	}
}

func NewMongoDBSegmentedBytesStore(ctx context.Context, name string,
	retention int64, // ms
	keySchema KeySchema,
	mkvs *MongoDBKeyValueStore,
) (*BaseSegmentedBytesStore, error) {
	segmentInterval := retention / 2
	if segmentInterval < 60_000 {
		segmentInterval = 60_000
	}
	segs, err := NewMongoDBKeyValueSegments(ctx, name, retention, segmentInterval, mkvs)
	if err != nil {
		return nil, err
	}
	return &BaseSegmentedBytesStore{
		name:               name,
		keySchema:          keySchema,
		observedStreamTime: -1,
		segments:           segs,
		tableType:          MONGODB,
	}, nil
}
*/

func (s *BaseSegmentedBytesStore) IsOpen() bool { return true }
func (s *BaseSegmentedBytesStore) Name() string { return s.name }

// Fetch all records from the segmented store with the provided key and time range
// from all existing segments
func (s *BaseSegmentedBytesStore) Fetch(ctx context.Context, key []byte, from int64, to int64,
	iterFunc func(int64 /* ts */, []byte, []byte) error,
) error {
	binaryFrom, err := s.keySchema.LowerRangeFixedSize(key, from)
	if err != nil {
		return err
	}
	binaryTo, err := s.keySchema.UpperRangeFixedSize(key, to)
	if err != nil {
		return err
	}
	// debug.Fprintf(os.Stderr, "fetch from: %d, to: %d, binaryFrom: %v, binaryTo: %v\n", from, to, binaryFrom, binaryTo)
	segment_slice := s.segments.Segments(from, to)
	// debug.Fprintf(os.Stderr, "segment slice: %v\n", segment_slice)
	for _, seg := range segment_slice {
		// debug.Fprintf(os.Stderr, "seg: %s\n", seg.Name())
		err := seg.Range(ctx, binaryFrom, binaryTo,
			func(kt []byte, vt []byte) error {
				// debug.Fprintf(os.Stderr, "got k: %v, v: %v\n", kt, vt)
				has, ts := s.keySchema.HasNextCondition(kt, key, key, from, to)
				if has {
					k := s.keySchema.ExtractStoreKeyBytes(kt)
					err := iterFunc(ts, k, vt)
					if err != nil {
						return err
					}
				}
				return nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// Fetch all records from the segmented store with the provided key and time range
// from all existing segments in backward order (from latest to earliest)
func (s *BaseSegmentedBytesStore) BackwardFetch(key []byte, from int64, to int64,
	iterFunc func(int64 /* ts */, []byte, []byte) error,
) error {
	panic("not implemented")
}

func (s *BaseSegmentedBytesStore) FetchWithKeyRange(ctx context.Context, keyFrom []byte, keyTo []byte,
	from int64, to int64,
	iterFunc func(int64 /* ts */, []byte, []byte) error,
) error {
	if bytes.Compare(keyFrom, keyTo) > 0 {
		return fmt.Errorf("key from should be smaller than key to")
	}
	binaryFrom, err := s.keySchema.LowerRangeFixedSize(keyFrom, from)
	if err != nil {
		return err
	}
	binaryTo, err := s.keySchema.UpperRangeFixedSize(keyTo, to)
	if err != nil {
		return err
	}
	segment_slice := s.segments.Segments(from, to)
	for _, seg := range segment_slice {
		err := seg.Range(ctx, binaryFrom, binaryTo, func(kt []byte, vt []byte) error {
			bytes := kt
			has, ts := s.keySchema.HasNextCondition(bytes, keyFrom, keyTo, from, to)
			if has {
				k := s.keySchema.ExtractStoreKeyBytes(bytes)
				err := iterFunc(ts, k, vt)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *BaseSegmentedBytesStore) BackwardFetchWithKeyRange(
	keyFrom []byte,
	keyTo []byte,
	from int64,
	to int64,
	iterFunc func(int64 /* ts */, []byte, []byte) error,
) error {
	panic("not implemented")
}

func (s *BaseSegmentedBytesStore) FetchAll(
	ctx context.Context,
	from int64, to int64,
	iterFunc func(int64 /* ts */, []byte, []byte) error,
) error {
	segment_slice := s.segments.Segments(from, to)
	for _, seg := range segment_slice {
		err := seg.Range(ctx, nil, nil, func(kt []byte, vt []byte) error {
			bytes := kt
			has, ts := s.keySchema.HasNextCondition(bytes, nil, nil, from, to)
			if has {
				k := s.keySchema.ExtractStoreKeyBytes(bytes)
				err := iterFunc(ts, k, vt)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *BaseSegmentedBytesStore) BackwardFetchAll(
	iterFunc func(int64 /* ts */, []byte, []byte) error,
) error {
	panic("not implemented")
}

func (s *BaseSegmentedBytesStore) Remove(ctx context.Context, key []byte) error {
	ts := s.keySchema.SegmentTimestamp(key)
	if ts > s.observedStreamTime {
		s.observedStreamTime = ts
	}
	segment := s.segments.GetSegmentForTimestamp(ts)
	if segment == nil {
		return nil
	}
	err := segment.Delete(ctx, key)
	return err
}

func (s *BaseSegmentedBytesStore) RemoveWithTs(key []byte, timestamp uint64) {
	panic("not implemented")
}

func (s *BaseSegmentedBytesStore) Put(ctx context.Context, key []byte, value []byte) error {
	ts := s.keySchema.SegmentTimestamp(key)
	if ts > s.observedStreamTime {
		s.observedStreamTime = ts
	}
	// debug.Fprintf(os.Stderr, "current observed time is %d\n", s.observedStreamTime)
	segmentId := s.segments.SegmentId(ts)
	segment, err := s.segments.GetOrCreateSegmentIfLive(ctx, segmentId, s.observedStreamTime,
		s.segments.GetOrCreateSegment, s.segments.CleanupExpiredMeta)
	if err != nil {
		return err
	}
	if segment == nil {
		log.Warn().Msg("Skipping record for expired segment")
	} else {
		return segment.Put(ctx, key, value)
	}
	return nil
}

func (s *BaseSegmentedBytesStore) Get(ctx context.Context, key []byte) ([]byte, bool, error) {
	segment := s.segments.GetSegmentForTimestamp(s.keySchema.SegmentTimestamp(key))
	if segment == nil {
		return nil, false, nil
	}
	return segment.Get(ctx, key)
}

func (s *BaseSegmentedBytesStore) DropDatabase(ctx context.Context) error {
	return s.segments.Destroy(ctx)
}

func (s *BaseSegmentedBytesStore) TableType() TABLE_TYPE {
	return s.tableType
}

func (s *BaseSegmentedBytesStore) StartTransaction(ctx context.Context) error {
	return s.segments.StartTransaction(ctx)
}
func (s *BaseSegmentedBytesStore) CommitTransaction(ctx context.Context, taskRepr string, transactionID uint64) error {
	return s.segments.CommitTransaction(ctx, taskRepr, transactionID)
}
func (s *BaseSegmentedBytesStore) AbortTransaction(ctx context.Context) error {
	return s.segments.AbortTransaction(ctx)
}

func (s *BaseSegmentedBytesStore) GetTransactionID(ctx context.Context, taskRepr string) (uint64, bool, error) {
	return s.segments.GetTransactionID(ctx, taskRepr)
}
