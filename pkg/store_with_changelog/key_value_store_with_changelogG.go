package store_with_changelog

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/processor"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/store"
	"sharedlog-stream/pkg/txn_data"
)

type KeyValueStoreWithChangelogG[K, V any] struct {
	kvstore          store.CoreKeyValueStoreG[K, V]
	msgSerde         commtypes.MessageGSerdeG[K, V]
	trackFunc        exactly_once_intr.TrackProdSubStreamFunc
	changelogManager *ChangelogManager[K, V]
	// changelogProduce *stats.ConcurrentStatsCollector[int64]
	kUseBuf bool
	vUseBuf bool
}

var (
	_ = store.KeyValueStoreBackedByChangelogG[int, int](&KeyValueStoreWithChangelogG[int, int]{})
	_ = store.CachedKeyValueStoreBackedByChangelogG[int, int](&KeyValueStoreWithChangelogG[int, int]{})
)

func NewKeyValueStoreWithChangelogG[K, V any](mp *MaterializeParam[K, V],
	store store.CoreKeyValueStoreG[K, V],
) (*KeyValueStoreWithChangelogG[K, V], error) {
	changelog, err := CreateChangelog(
		mp.storeName, mp.changelogParam.NumPartition, mp.serdeFormat, mp.bufMaxSize)
	if err != nil {
		return nil, err
	}
	changelogManager, err := NewChangelogManager(changelog, mp.msgSerde, mp.changelogParam.TimeOut,
		mp.changelogParam.FlushDuration, mp.serdeFormat, mp.parNum)
	if err != nil {
		return nil, err
	}
	err = store.SetKVSerde(mp.serdeFormat, mp.msgSerde.GetKeySerdeG(), mp.msgSerde.GetValSerdeG())
	if err != nil {
		return nil, err
	}
	store.SetInstanceId(mp.parNum)
	return &KeyValueStoreWithChangelogG[K, V]{
		kvstore:          store,
		trackFunc:        exactly_once_intr.DefaultTrackProdSubstreamFunc,
		msgSerde:         mp.msgSerde,
		changelogManager: changelogManager,
		// changelogProduce: stats.NewConcurrentStatsCollector[int64](mp.storeName+"-clProd",
		// 	stats.DEFAULT_COLLECT_DURATION),
		kUseBuf: mp.msgSerde.GetKeySerdeG().UsedBufferPool(),
		vUseBuf: mp.msgSerde.GetValSerdeG().UsedBufferPool(),
	}, nil
}

func (st *KeyValueStoreWithChangelogG[K, V]) OutputRemainingStats() {
	st.changelogManager.OutputRemainingStats()
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat, keySerde commtypes.SerdeG[K], valSerde commtypes.SerdeG[V]) error {
	return nil
}

func (st *KeyValueStoreWithChangelogG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[K, V]] {
	return st.kvstore.GetKVSerde()
}

func (st *KeyValueStoreWithChangelogG[K, V]) Name() string {
	return st.kvstore.Name()
}

func (st *KeyValueStoreWithChangelogG[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	return st.kvstore.Get(ctx, key)
}

func (st *KeyValueStoreWithChangelogG[K, V]) Flush(ctx context.Context) (uint32, error) {
	return st.changelogManager.Flush(ctx)
}

func (st *KeyValueStoreWithChangelogG[K, V]) ConfigureExactlyOnce(
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	st.changelogManager.ConfigExactlyOnce(rem, guarantee)
}

func (st *KeyValueStoreWithChangelogG[K, V]) Put(ctx context.Context, key K, value optional.Option[V], tm store.TimeMeta) error {
	msg := commtypes.MessageG[K, V]{
		Key:   optional.Some(key),
		Value: value,
	}
	// pStart := stats.TimerBegin()
	msgSerOp, kbuf, vbuf, err := commtypes.MsgGToMsgSer(msg, st.msgSerde)
	if err != nil {
		return err
	}
	msgSer, ok := msgSerOp.Take()
	defer func() {
		if st.kUseBuf && kbuf != nil {
			*kbuf = msgSer.KeyEnc
			commtypes.PushBuffer(kbuf)
		}
		if st.vUseBuf && vbuf != nil {
			*vbuf = msgSer.ValueEnc
			commtypes.PushBuffer(vbuf)
		}
	}()
	if !ok {
		return nil
	}
	err = st.changelogManager.produce(ctx, msgSer, st.kvstore.GetInstanceId())
	// elapsed := stats.Elapsed(pStart).Microseconds()
	// st.changelogProduce.AddSample(elapsed)
	if err != nil {
		return err
	}
	err = st.kvstore.Put(ctx, key, value, tm)
	return err
}

func (st *KeyValueStoreWithChangelogG[K, V]) BuildKeyMeta(ctx context.Context, kms map[string][]txn_data.KeyMaping) error {
	kms[st.changelogManager.TopicName()] = make([]txn_data.KeyMaping, 0)
	hasher := hashfuncs.ByteSliceHasher{}
	return st.kvstore.Range(ctx, optional.None[K](), optional.None[K](), func(k K, v V) error {
		kBytes, _, err := st.msgSerde.GetKeySerdeG().Encode(k)
		if err != nil {
			return err
		}
		hash := hasher.HashSum64(kBytes)
		kms[st.changelogManager.TopicName()] = append(kms[st.changelogManager.TopicName()], txn_data.KeyMaping{
			Key:         kBytes,
			SubstreamId: st.kvstore.GetInstanceId(),
			Hash:        hash,
		})
		return nil
	})
}

func (st *KeyValueStoreWithChangelogG[K, V]) PutWithoutPushToChangelog(ctx context.Context, key commtypes.KeyT, value commtypes.ValueT) error {
	return st.kvstore.PutWithoutPushToChangelog(ctx, key, value)
}

func (st *KeyValueStoreWithChangelogG[K, V]) PutIfAbsent(ctx context.Context, key K, value V, tm store.TimeMeta) (optional.Option[V], error) {
	origVal, exists, err := st.kvstore.Get(ctx, key)
	if err != nil {
		return optional.None[V](), err
	}
	if !exists {
		err := st.Put(ctx, key, optional.Some(value), tm)
		if err != nil {
			return optional.None[V](), err
		}
		return optional.None[V](), nil
	}
	return optional.Some(origVal), nil
}

func (st *KeyValueStoreWithChangelogG[K, V]) Delete(ctx context.Context, key K) error {
	msg := commtypes.MessageG[K, V]{
		Key:   optional.Some(key),
		Value: optional.None[V](),
	}
	msgSerOp, kbuf, vbuf, err := commtypes.MsgGToMsgSer(msg, st.msgSerde)
	if err != nil {
		return err
	}
	msgSer, ok := msgSerOp.Take()
	defer func() {
		if st.kUseBuf && kbuf != nil {
			*kbuf = msgSer.KeyEnc
			commtypes.PushBuffer(kbuf)
		}
		if st.vUseBuf && vbuf != nil {
			*vbuf = msgSer.ValueEnc
			commtypes.PushBuffer(vbuf)
		}
	}()
	if ok {
		err := st.changelogManager.produce(ctx, msgSer, st.kvstore.GetInstanceId())
		if err != nil {
			return err
		}
		return st.kvstore.Delete(ctx, key)
	} else {
		return nil
	}
}

func (st *KeyValueStoreWithChangelogG[K, V]) ApproximateNumEntries() (uint64, error) {
	return st.kvstore.ApproximateNumEntries()
}

func (st *KeyValueStoreWithChangelogG[K, V]) Range(ctx context.Context, from optional.Option[K], to optional.Option[K], iterFunc func(K, V) error) error {
	return st.kvstore.Range(ctx, from, to, iterFunc)
}

func (st *KeyValueStoreWithChangelogG[K, V]) TableType() store.TABLE_TYPE {
	return st.kvstore.TableType()
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc) {
	st.trackFunc = trackParFunc
	st.trackFunc(st.changelogManager.TopicName(), st.kvstore.GetInstanceId())
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetStreamFlushCallbackFunc(cb exactly_once_intr.FlushCallbackFunc) {
	st.changelogManager.producer.SetFlushCallback(cb)
}

func (st *KeyValueStoreWithChangelogG[K, V]) ConsumeOneLogEntry(ctx context.Context, parNum uint8) (int, error) {
	msgSeq, err := st.changelogManager.Consume(ctx, parNum)
	if err != nil {
		return 0, err
	}
	count := 0
	if msgSeq.MsgArr != nil {
		for _, msg := range msgSeq.MsgArr {
			if msg.Key.IsNone() && msg.Value.IsNone() {
				continue
			}
			count += 1
			k := msg.Key.Unwrap()
			v := msg.Value.Unwrap()
			err = st.PutWithoutPushToChangelog(ctx, k, v)
			if err != nil {
				return 0, err
			}
		}
	} else {
		msg := msgSeq.Msg
		if msg.Key.IsNone() && msg.Value.IsNone() {
			return 0, nil
		}
		count += 1
		k := msg.Key.Unwrap()
		v := msg.Value.Unwrap()
		err = st.PutWithoutPushToChangelog(ctx, k, v)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (st *KeyValueStoreWithChangelogG[K, V]) ChangelogTopicName() string {
	return st.changelogManager.TopicName()
}

func (st *KeyValueStoreWithChangelogG[K, V]) ChangelogIsSrc() bool {
	return false
}

func (st *KeyValueStoreWithChangelogG[K, V]) Stream() sharedlog_stream.Stream {
	return st.changelogManager.Stream()
}

func (st *KeyValueStoreWithChangelogG[K, V]) GetInitialProdSeqNum() uint64 {
	return st.changelogManager.producer.GetInitialProdSeqNum(st.kvstore.GetInstanceId())
}

func (st *KeyValueStoreWithChangelogG[K, V]) ResetInitialProd() {
	st.changelogManager.producer.ResetInitialProd()
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetLastMarkerSeq(lastMarkerSeq uint64) {
	st.changelogManager.producer.SetLastMarkerSeq(lastMarkerSeq)
}

func (st *KeyValueStoreWithChangelogG[K, V]) SubstreamNum() uint8 {
	return st.kvstore.GetInstanceId()
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetCacheFlushCallback(
	store.KVStoreCacheFlushCallbackFunc[K, V]) {
}

func (st *KeyValueStoreWithChangelogG[K, V]) Snapshot(
	ctx context.Context, tpLogoff []commtypes.TpLogOff, unprocessed []commtypes.ChkptMetaData, resetBg bool,
) {
	st.kvstore.Snapshot(ctx, tpLogoff, unprocessed, resetBg)
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetSnapshotCallback(ctx context.Context, f store.KVSnapshotCallback[K, V]) {
	st.kvstore.SetSnapshotCallback(ctx, f)
}

func (st *KeyValueStoreWithChangelogG[K, V]) WaitForAllSnapshot() error {
	return st.kvstore.WaitForAllSnapshot()
}

func (st *KeyValueStoreWithChangelogG[K, V]) RestoreFromSnapshot(snapshot [][]byte) error {
	return st.kvstore.RestoreFromSnapshot(snapshot)
}

func (st *KeyValueStoreWithChangelogG[K, V]) FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error) {
	return st.changelogManager.findLastEpochMetaWithAuxData(ctx, parNum)
}

func (st *KeyValueStoreWithChangelogG[K, V]) SetInstanceId(id uint8) {
	st.kvstore.SetInstanceId(id)
}

func (st *KeyValueStoreWithChangelogG[K, V]) GetInstanceId() uint8 {
	return st.kvstore.GetInstanceId()
}

func CreateInMemorySkipmapKVTableWithChangelogG[K, V any](mp *MaterializeParam[K, V], less store.LessFunc[K],
) (*KeyValueStoreWithChangelogG[K, V], error) {
	s := store.NewInMemorySkipmapKeyValueStoreG[K, V](mp.storeName, less)
	s.SetInstanceId(mp.parNum)
	return NewKeyValueStoreWithChangelogG[K, V](mp, s)
}

func ToInMemSkipmapKVTableWithChangelog[K, V any](mp *MaterializeParam[K, commtypes.ValueTimestampG[V]],
	less store.LessFunc[K],
) (*processor.MeteredProcessorG[K, V, K, commtypes.ChangeG[V]],
	*KeyValueStoreWithChangelogG[K, commtypes.ValueTimestampG[V]], error,
) {
	s := store.NewInMemorySkipmapKeyValueStoreG[K, commtypes.ValueTimestampG[V]](mp.storeName, less)
	s.SetInstanceId(mp.parNum)
	storeWithlog, err := NewKeyValueStoreWithChangelogG[K, commtypes.ValueTimestampG[V]](mp, s)
	if err != nil {
		return nil, nil, err
	}
	toTableProc := processor.NewMeteredProcessorG[K, V, K, commtypes.ChangeG[V]](
		processor.NewTableSourceProcessorWithTableG[K, V](storeWithlog))
	return toTableProc, storeWithlog, nil
}
