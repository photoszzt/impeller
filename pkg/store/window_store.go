package store

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/txn_data"
	"time"
)

type CoreWindowStore interface {
	StateStore
	Put(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT, windowStartTimestamp int64) error
	Get(ctx context.Context, key commtypes.KeyT, windowStartTimestamp int64) (commtypes.ValueT, bool, error)
	Fetch(ctx context.Context, key commtypes.KeyT, timeFrom time.Time, timeTo time.Time,
		iterFunc func(int64 /* ts */, commtypes.KeyT, commtypes.ValueT) error) error
	FetchWithKeyRange(ctx context.Context, keyFrom commtypes.KeyT,
		keyTo commtypes.KeyT, timeFrom time.Time, timeTo time.Time,
		iterFunc func(int64, commtypes.KeyT, commtypes.ValueT) error) error
	FetchAll(ctx context.Context, timeFrom time.Time, timeTo time.Time,
		iterFunc func(int64, commtypes.KeyT, commtypes.ValueT) error) error
	IterAll(iterFunc func(int64, commtypes.KeyT, commtypes.ValueT) error) error
	TableType() TABLE_TYPE
	UpdateTrackParFunc
	OnlyUpdateInMemWinStore
}

type CoreWindowStoreG[K, V any] interface {
	StateStore
	Put(ctx context.Context, key K, value optional.Option[V], windowStartTimestamp int64, tm TimeMeta) error
	Get(ctx context.Context, key K, windowStartTimestamp int64) (V, bool, error)
	Fetch(ctx context.Context, key K, timeFrom time.Time, timeTo time.Time,
		iterFunc func(int64 /* ts */, K, V) error) error
	FetchWithKeyRange(ctx context.Context, keyFrom K,
		keyTo K, timeFrom time.Time, timeTo time.Time,
		iterFunc func(int64, K, V) error) error
	FetchAll(ctx context.Context, timeFrom time.Time, timeTo time.Time,
		iterFunc func(int64, K, V) error) error
	IterAll(iterFunc func(int64, K, V) error) error
	TableType() TABLE_TYPE
	Flush(ctx context.Context) (uint32, error)
	Snapshot(ctx context.Context, tplogOff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool)
	SetWinSnapshotCallback(ctx context.Context, f WinSnapshotCallback[K, V])
	WaitForAllSnapshot() error
	SetKVSerde(serdeFormat commtypes.SerdeFormat,
		keySerde commtypes.SerdeG[commtypes.KeyAndWindowStartTsG[K]],
		origKeySerde commtypes.SerdeG[K],
		valSerde commtypes.SerdeG[V],
	) error
	GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]]
	RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error
	OnlyUpdateInMemWinStoreG[K, V]
	BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error
	OnlyUpdateInMemStore
	SetInstanceId(uint8)
	GetInstanceId() uint8
}

type WindowStoreBackedByChangelog interface {
	CoreWindowStore
	WindowStoreOpWithChangelog
}

type WindowStoreBackedByChangelogG[K, V any] interface {
	CoreWindowStoreG[K, V]
	WindowStoreOpWithChangelog
}

type CachedWindowStoreBackedByChangelogG[K, V any] interface {
	CachedWindowStateStore[K, V]
	WindowStoreOpWithChangelog
}

type OnlyUpdateInMemWinStore interface {
	PutWithoutPushToChangelog(ctx context.Context,
		key commtypes.KeyT, value commtypes.ValueT) error
}

type OnlyUpdateInMemWinStoreG[K, V any] interface {
	PutWithoutPushToChangelogG(ctx context.Context,
		key K, value optional.Option[V], windowStartTs int64) error
}

type WindowStoreOp interface {
	OnlyUpdateInMemWinStore
	GetInstanceId() uint8
	Flush(ctx context.Context) (uint32, error)
	Snapshot(ctx context.Context, tplogOff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool)
	WaitForAllSnapshot() error
	RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error
	BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error
}

type WindowStoreOpWithChangelog interface {
	StoreBackedByChangelog
	RestoreFromChangelog
	UpdateTrackParFunc
	ProduceRangeRecording
	SubstreamNum() uint8
	FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error)
	WindowStoreOp
}

type WinStoreOps struct {
	Wsos []WindowStoreOp
	Wsc  map[string]WindowStoreOpWithChangelog
}
