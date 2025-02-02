package store

import (
	"context"
	"math"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils"
	"sharedlog-stream/pkg/utils/syncutils"
	"time"

	"golang.org/x/sync/errgroup"
)

type SegmentedWindowStoreG[K, V any] struct {
	seqNumMu         syncutils.Mutex
	bytesStore       SegmentedBytesStore
	keySerde         commtypes.SerdeG[K]
	valSerde         commtypes.SerdeG[V]
	windowKeySchema  *WindowKeySchema
	snapshotCallback WinSnapshotCallback[K, V]
	bgErrG           *errgroup.Group
	bgCtx            context.Context
	windowSize       int64
	seqNum           uint32
	retainDuplicates bool
	insId            uint8
}

var _ = CoreWindowStoreG[int, int](&SegmentedWindowStoreG[int, int]{})

func NewSegmentedWindowStore[K, V any](
	bytesStore SegmentedBytesStore,
	retainDuplicates bool,
	windowSize int64,
	keySerde commtypes.SerdeG[K],
	valSerde commtypes.SerdeG[V],
) *SegmentedWindowStoreG[K, V] {
	return &SegmentedWindowStoreG[K, V]{
		bytesStore:       bytesStore,
		retainDuplicates: retainDuplicates,
		windowSize:       windowSize,
		seqNum:           0,
		windowKeySchema:  &WindowKeySchema{},
	}
}

func (rws *SegmentedWindowStoreG[K, V]) updateSeqnumForDups() {
	if rws.retainDuplicates {
		rws.seqNumMu.Lock()
		defer rws.seqNumMu.Unlock()
		if rws.seqNum == math.MaxUint32 {
			rws.seqNum = 0
		}
		rws.seqNum += 1
	}
}

func (rws *SegmentedWindowStoreG[K, V]) Name() string { return rws.bytesStore.Name() }

func (rws *SegmentedWindowStoreG[K, V]) Put(ctx context.Context, key K, value optional.Option[V], windowStartTimestamp int64, tm TimeMeta) error {
	if !(value.IsNone() && rws.retainDuplicates) {
		rws.updateSeqnumForDups()
		var err error
		kBytes, _, err := rws.keySerde.Encode(key)
		if err != nil {
			return err
		}
		rws.seqNumMu.Lock()
		defer rws.seqNumMu.Unlock()
		k, err := rws.windowKeySchema.ToStoreKeyBinary(kBytes, windowStartTimestamp, rws.seqNum)
		if err != nil {
			return err
		}
		var vBytes []byte
		v, ok := value.Take()
		if ok {
			vBytes, _, err = rws.valSerde.Encode(v)
			if err != nil {
				return err
			}
		}
		return rws.bytesStore.Put(ctx, k, vBytes)
	}
	return nil
}

func (rws *SegmentedWindowStoreG[K, V]) PutWithoutPushToChangelogG(ctx context.Context, key K, value optional.Option[V], windowStartTs int64) error {
	kBytes, _, err := rws.keySerde.Encode(key)
	if err != nil {
		return err
	}
	k, err := rws.windowKeySchema.ToStoreKeyBinary(kBytes, windowStartTs, rws.seqNum)
	if err != nil {
		return err
	}
	var vBytes []byte
	v, ok := value.Take()
	if ok {
		vBytes, _, err = rws.valSerde.Encode(v)
		if err != nil {
			return err
		}
	}
	return rws.bytesStore.Put(ctx, k, vBytes)
}

func (rws *SegmentedWindowStoreG[K, V]) PutWithoutPushToChangelog(
	ctx context.Context, key commtypes.KeyT, value commtypes.ValueT,
) error {
	keyTs := key.(commtypes.KeyAndWindowStartTsG[K])
	if utils.IsNil(value) {
		return rws.Put(ctx, keyTs.Key, optional.None[V](), keyTs.WindowStartTs, TimeMeta{RecordTsMs: 0})
	} else {
		return rws.Put(ctx, key.(K), optional.Some(value.(V)), keyTs.WindowStartTs, TimeMeta{RecordTsMs: 0})
	}
}

func (rws *SegmentedWindowStoreG[K, V]) Get(ctx context.Context, key K, windowStartTimestamp int64) (V, bool, error) {
	var v V
	var err error
	kBytes, b, err := rws.keySerde.Encode(key)
	defer func() {
		if rws.keySerde.UsedBufferPool() && b != nil {
			*b = kBytes
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return v, false, err
	}
	rws.seqNumMu.Lock()
	defer rws.seqNumMu.Unlock()
	k, err := rws.windowKeySchema.ToStoreKeyBinary(kBytes, windowStartTimestamp, rws.seqNum)
	if err != nil {
		return v, false, err
	}
	valBytes, ok, err := rws.bytesStore.Get(ctx, k)
	if err != nil {
		return v, false, err
	}
	if ok {
		val, err := rws.valSerde.Decode(valBytes)
		if err != nil {
			return v, false, err
		}
		return val, ok, err
	}
	return v, false, nil
}

func (rws *SegmentedWindowStoreG[K, V]) Fetch(ctx context.Context, key K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64 /* ts */, K, V) error,
) error {
	tsFrom := timeFrom.UnixMilli()
	tsTo := timeTo.UnixMilli()
	var err error
	kBytes, b, err := rws.keySerde.Encode(key)
	defer func() {
		if rws.keySerde.UsedBufferPool() && b != nil {
			*b = kBytes
			commtypes.PushBuffer(b)
		}
	}()
	if err != nil {
		return err
	}
	err = rws.bytesStore.Fetch(ctx, kBytes, tsFrom, tsTo, func(ts int64, kBytes, vBytes []byte) error {
		k, err := rws.keySerde.Decode(kBytes)
		if err != nil {
			return err
		}
		v, err := rws.valSerde.Decode(vBytes)
		if err != nil {
			return err
		}
		return iterFunc(ts, k, v)
	})
	return err
}

func (rws *SegmentedWindowStoreG[K, V]) FetchWithKeyRange(ctx context.Context, keyFrom K, keyTo K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	tsFrom := timeFrom.UnixMilli()
	tsTo := timeTo.UnixMilli()
	var err error
	useBuf := rws.keySerde.UsedBufferPool()
	kFromBytes, b1, err := rws.keySerde.Encode(keyFrom)
	if err != nil {
		if useBuf && b1 != nil {
			*b1 = kFromBytes
			commtypes.PushBuffer(b1)
		}
		return err
	}
	kToBytes, b2, err := rws.keySerde.Encode(keyTo)
	if err != nil {
		if useBuf {
			if b1 != nil {
				*b1 = kFromBytes
				commtypes.PushBuffer(b1)
			}
			if b2 != nil {
				*b2 = kFromBytes
				commtypes.PushBuffer(b2)
			}
		}
		return err
	}
	err = rws.bytesStore.FetchWithKeyRange(ctx, kFromBytes, kToBytes, tsFrom, tsTo,
		func(ts int64, kBytes, vBytes []byte) error {
			k, err := rws.keySerde.Decode(kBytes)
			if err != nil {
				return err
			}
			v, err := rws.valSerde.Decode(vBytes)
			if err != nil {
				return err
			}
			return iterFunc(ts, k, v)
		})
	if useBuf {
		if b1 != nil {
			*b1 = kFromBytes
			commtypes.PushBuffer(b1)
		}
		if b2 != nil {
			*b2 = kFromBytes
			commtypes.PushBuffer(b2)
		}
	}
	return err
}

func (rws *SegmentedWindowStoreG[K, V]) FetchAll(ctx context.Context, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	tsFrom := timeFrom.UnixMilli()
	tsTo := timeTo.UnixMilli()
	return rws.bytesStore.FetchAll(ctx, tsFrom, tsTo, func(ts int64, k, v []byte) error {
		key, err := rws.keySerde.Decode(k)
		if err != nil {
			return err
		}
		val, err := rws.valSerde.Decode(v)
		if err != nil {
			return err
		}
		return iterFunc(ts, key, val)
	})
}

func (rws *SegmentedWindowStoreG[K, V]) IterAll(iterFunc func(int64, K, V) error) error {
	panic("not implemented")
}

func (rws *SegmentedWindowStoreG[K, V]) DropDatabase(ctx context.Context) error {
	return rws.bytesStore.DropDatabase(ctx)
}

func (rws *SegmentedWindowStoreG[K, V]) TableType() TABLE_TYPE {
	return rws.bytesStore.TableType()
}

func (rws *SegmentedWindowStoreG[K, V]) StartTransaction(ctx context.Context) error {
	return rws.bytesStore.StartTransaction(ctx)
}

func (rws *SegmentedWindowStoreG[K, V]) CommitTransaction(ctx context.Context, taskRepr string,
	transactionID uint64,
) error {
	return rws.bytesStore.CommitTransaction(ctx, taskRepr, transactionID)
}

func (rws *SegmentedWindowStoreG[K, V]) AbortTransaction(ctx context.Context) error {
	return rws.bytesStore.AbortTransaction(ctx)
}

func (rws *SegmentedWindowStoreG[K, V]) GetTransactionID(ctx context.Context, taskRepr string) (uint64, bool, error) {
	return rws.bytesStore.GetTransactionID(ctx, taskRepr)
}

func (rws *SegmentedWindowStoreG[K, V]) SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc) {
}

func (rws *SegmentedWindowStoreG[K, V]) FlushChangelog(ctx context.Context) error {
	return nil
}

func (rws *SegmentedWindowStoreG[K, V]) ConsumeChangelog(ctx context.Context, parNum uint8) (*commtypes.MsgAndSeqs, error) {
	return nil, nil
}

func (rws *SegmentedWindowStoreG[K, V]) ConfigureExactlyOnce(rem exactly_once_intr.ReadOnlyExactlyOnceManager, guarantee exactly_once_intr.GuaranteeMth, serdeFormat commtypes.SerdeFormat) error {
	panic("not supported")
}

func (rws *SegmentedWindowStoreG[K, V]) ChangelogTopicName() string {
	panic("not supported")
}

func (s *SegmentedWindowStoreG[K, V]) Flush(ctx context.Context) (uint32, error) {
	return 0, nil
}

func (s *SegmentedWindowStoreG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat,
	keySerde commtypes.SerdeG[commtypes.KeyAndWindowStartTsG[K]],
	origKeySerde commtypes.SerdeG[K],
	valSerde commtypes.SerdeG[V],
) error {
	panic("not supported")
}

func (s *SegmentedWindowStoreG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]] {
	panic("not supported")
}

func (s *SegmentedWindowStoreG[K, V]) RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error {
	panic("not implemented")
}

func (s *SegmentedWindowStoreG[K, V]) SetFlushCallback(func(ctx context.Context, msg commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[V]]) error) {
}

func (s *SegmentedWindowStoreG[K, V]) SetFlushCallbackFunc(exactly_once_intr.FlushCallbackFunc) {
}

func (s *SegmentedWindowStoreG[K, V]) BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error {
	panic("not supported")
}

func (s *SegmentedWindowStoreG[K, V]) SetWinSnapshotCallback(ctx context.Context, f WinSnapshotCallback[K, V]) {
	s.bgErrG, s.bgCtx = errgroup.WithContext(ctx)
	s.snapshotCallback = f
}

func (s *SegmentedWindowStoreG[K, V]) Snapshot(ctx context.Context, tplogoff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool) {
	panic("not implemented")
}

func (s *SegmentedWindowStoreG[K, V]) WaitForAllSnapshot() error {
	return s.bgErrG.Wait()
}

func (s *SegmentedWindowStoreG[K, V]) SetInstanceId(id uint8) {
	s.insId = id
}

func (s *SegmentedWindowStoreG[K, V]) GetInstanceId() uint8 {
	return s.insId
}
