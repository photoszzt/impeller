package store

import (
	"context"
)

type SegmentedBytesStore interface {
	StateStore

	// Fetch all records from the segmented store with the provided key and time range
	// from all existing segments
	Fetch(ctx context.Context, key []byte, from int64, to int64,
		iterFunc func(int64 /* ts */, []byte, []byte) error) error

	// Fetch all records from the segmented store with the provided key and time range
	// from all existing segments in backward order (from latest to earliest)
	BackwardFetch(key []byte, from int64, to int64, iterFunc func(int64 /* ts */, []byte /* k */, []byte /* v */) error) error
	FetchWithKeyRange(ctx context.Context, keyFrom []byte, keyTo []byte, from int64, to int64,
		iterFunc func(ts int64 /* ts */, k []byte, v []byte) error) error
	BackwardFetchWithKeyRange(keyFrom []byte, keyTo []byte, from int64, to int64,
		iterFunc func(ts int64 /* ts */, k []byte, v []byte) error) error
	FetchAll(ctx context.Context, from int64, to int64, iterFunc func(ts int64 /* ts */, k []byte, v []byte) error) error
	BackwardFetchAll(iterFunc func(ts int64 /* ts */, k []byte, v []byte) error) error
	Remove(ctx context.Context, key []byte) error
	RemoveWithTs(key []byte, timestamp uint64)
	Put(ctx context.Context, key []byte, value []byte) error
	Get(ctx context.Context, key []byte) ([]byte, bool, error)
	DropDatabase(ctx context.Context) error
	TableType() TABLE_TYPE
	StartTransaction(ctx context.Context) error
	CommitTransaction(ctx context.Context, taskRepr string, transactionalID uint64) error
	AbortTransaction(ctx context.Context) error
	GetTransactionID(ctx context.Context, taskRepr string) (uint64, bool, error)
}

type KeySchema interface {
	// Given a range of record keys and a time, construct a Segmented key that represents
	// the upper range of keys to search when performing range queries.
	UpperRange(key []byte, to int64) ([]byte, error)
	LowerRange(key []byte, from int64) []byte
	ToStoreBinaryKeyPrefix(key []byte, ts int64) ([]byte, error)
	UpperRangeFixedSize(key []byte, to int64) ([]byte, error)
	LowerRangeFixedSize(key []byte, from int64) ([]byte, error)
	SegmentTimestamp(key []byte) int64
	HasNextCondition(curKey []byte, binaryKeyFrom []byte, binaryKeyTo []byte,
		from int64, to int64) (bool, int64)
	ExtractStoreKeyBytes(key []byte) []byte
	// SegmentsToSearch()
}
