package store

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/txn_data"
	"strings"
	"time"
)

type TimeMeta struct {
	StartProcTs time.Time
	RecordTsMs  int64
}

type CoreKeyValueStoreG[K, V any] interface {
	StateStore
	Get(ctx context.Context, key K) (V, bool, error)
	Range(ctx context.Context, from optional.Option[K], to optional.Option[K],
		iterFunc func(K, V) error) error
	ApproximateNumEntries() (uint64, error)
	Put(ctx context.Context, key K, value optional.Option[V], tm TimeMeta) error
	PutIfAbsent(ctx context.Context, key K, value V, tm TimeMeta) (optional.Option[V], error)
	// PutAll(context.Context, []*commtypes.Message) error
	Delete(ctx context.Context, key K) error
	TableType() TABLE_TYPE
	Flush(ctx context.Context) (uint32, error)
	Snapshot(context.Context, []commtypes.TpLogOff, []commtypes.ChkptMetaData, bool)
	SetSnapshotCallback(ctx context.Context, f KVSnapshotCallback[K, V])
	WaitForAllSnapshot() error
	RestoreFromSnapshot(snapshot [][]byte) error
	SetKVSerde(serdeFormat commtypes.SerdeFormat, keySerde commtypes.SerdeG[K], valSerde commtypes.SerdeG[V]) error
	GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[K, V]]
	SetInstanceId(uint8)
	GetInstanceId() uint8
	BuildKeyMeta(ctx context.Context, kms map[string][]txn_data.KeyMaping) error
	OnlyUpdateInMemStore
}

type OnlyUpdateInMemStore interface {
	PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error
}

type OnlyUpdateInMemStoreG[K, V any] interface {
	PutWithoutPushToChangelogG(ctx context.Context, key K, value V) error
}

type StoreWithFlush interface {
	Flush(ctx context.Context) (uint32, error)
}

type KeyValueStoreOp interface {
	OnlyUpdateInMemStore
	GetInstanceId() uint8
	Flush(ctx context.Context) (uint32, error)
	Snapshot(context.Context, []commtypes.TpLogOff, []commtypes.ChkptMetaData, bool)
	WaitForAllSnapshot() error
	RestoreFromSnapshot(snapshot [][]byte) error
	BuildKeyMeta(ctx context.Context, kms map[string][]txn_data.KeyMaping) error
}

type KeyValueStoreOpWithChangelog interface {
	StoreBackedByChangelog
	RestoreFromChangelog
	ChangelogIsSrc() bool
	UpdateTrackParFunc
	ProduceRangeRecording
	FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error)
	SubstreamNum() uint8
	KeyValueStoreOp
}

type KVStoreOps struct {
	Kvo []KeyValueStoreOp
	Kvc map[string]KeyValueStoreOpWithChangelog
}

type KeyValueStoreBackedByChangelogG[K, V any] interface {
	CoreKeyValueStoreG[K, V]
	KeyValueStoreOpWithChangelog
}

type CachedKeyValueStoreBackedByChangelogG[K, V any] interface {
	CachedKeyValueStore[K, V]
	KeyValueStoreOpWithChangelog
}

type LessFunc[K any] func(k1, k2 K) bool

func IntLessFunc(k1, k2 int) bool {
	return k1 < k2
}

func Uint64LessFunc(k1, k2 uint64) bool {
	return k1 < k2
}

func StringLessFunc(k1, k2 string) bool {
	return strings.Compare(k1, k2) < 0
}

type Segment interface {
	StateStore
	Get(ctx context.Context, key []byte) ([]byte, bool, error)
	Range(ctx context.Context, from []byte, to []byte,
		iterFunc func([]byte, []byte) error) error
	ReverseRange(from []byte, to []byte, iterFunc func([]byte, []byte) error) error
	PrefixScan(prefix interface{}, prefixKeyEncoder commtypes.Encoder, iterFunc func([]byte, []byte) error) error
	ApproximateNumEntries(ctx context.Context) (uint64, error)
	Put(ctx context.Context, key []byte, value []byte) error
	PutIfAbsent(ctx context.Context, key []byte, value []byte) ([]byte, error)
	PutAll(context.Context, []*commtypes.Message) error
	Delete(ctx context.Context, key []byte) error

	Destroy(ctx context.Context) error
	DeleteRange(ctx context.Context, keyFrom interface{}, keyTo interface{}) error
}
