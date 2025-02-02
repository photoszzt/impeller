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
	"time"

	"github.com/rs/zerolog/log"
)

type InMemoryWindowStoreWithChangelogG[K, V any] struct {
	msgSerde         commtypes.MessageGSerdeG[commtypes.KeyAndWindowStartTsG[K], V]
	originKeySerde   commtypes.SerdeG[K]
	windowStore      store.CoreWindowStoreG[K, V]
	trackFunc        exactly_once_intr.TrackProdSubStreamFunc
	changelogManager *ChangelogManager[commtypes.KeyAndWindowStartTsG[K], V]
	parNum           uint8
	kUseBuf          bool
	vUseBuf          bool
}

func createChangelogManagerAndUpdateMsgSerde[K, V any](mp *MaterializeParam[K, V]) (
	*ChangelogManager[commtypes.KeyAndWindowStartTsG[K], V],
	commtypes.MessageGSerdeG[commtypes.KeyAndWindowStartTsG[K], V], error,
) {
	changelog, err := CreateChangelog(
		mp.storeName, mp.changelogParam.NumPartition,
		mp.serdeFormat, mp.bufMaxSize)
	if err != nil {
		return nil, nil, err
	}
	keyAndWindowStartTsSerde, err := commtypes.GetKeyAndWindowStartTsSerdeG(mp.serdeFormat, mp.msgSerde.GetKeySerdeG())
	if err != nil {
		return nil, nil, err
	}
	msgSerde, err := commtypes.GetMsgGSerdeG(mp.serdeFormat, keyAndWindowStartTsSerde, mp.msgSerde.GetValSerdeG())
	if err != nil {
		return nil, nil, err
	}
	changelogManager, err := NewChangelogManager(changelog, msgSerde, mp.changelogParam.TimeOut,
		mp.changelogParam.FlushDuration, mp.serdeFormat, mp.parNum)
	if err != nil {
		return nil, nil, err
	}
	return changelogManager, msgSerde, nil
}

var (
	_ = store.WindowStoreBackedByChangelogG[int, string](&InMemoryWindowStoreWithChangelogG[int, string]{})
	_ = store.CachedWindowStoreBackedByChangelogG[int, string](&InMemoryWindowStoreWithChangelogG[int, string]{})
)

func NewInMemoryWindowStoreWithChangelogG[K, V any](
	windowStore store.CoreWindowStoreG[K, V],
	mp *MaterializeParam[K, V],
) (*InMemoryWindowStoreWithChangelogG[K, V], error) {
	changelogManager, msgSerde, err := createChangelogManagerAndUpdateMsgSerde(mp)
	if err != nil {
		return nil, err
	}
	err = windowStore.SetKVSerde(mp.serdeFormat,
		msgSerde.GetKeySerdeG(), mp.msgSerde.GetKeySerdeG(), msgSerde.GetValSerdeG())
	if err != nil {
		return nil, err
	}
	return &InMemoryWindowStoreWithChangelogG[K, V]{
		windowStore:      windowStore,
		msgSerde:         msgSerde,
		originKeySerde:   mp.msgSerde.GetKeySerdeG(),
		parNum:           mp.ParNum(),
		trackFunc:        exactly_once_intr.DefaultTrackProdSubstreamFunc,
		changelogManager: changelogManager,
		// changeLogProduce: stats.NewConcurrentInt64Collector(mp.storeName+"-clProd", stats.DEFAULT_COLLECT_DURATION),
		// storePutLatency: stats.NewConcurrentInt64Collector(mp.storeName+"-storePutLatency", stats.DEFAULT_COLLECT_DURATION),
		// trackFuncLat: stats.NewConcurrentStatsCollector[int64](mp.storeName+"-trackFuncLat", stats.DEFAULT_COLLECT_DURATION),
		kUseBuf: msgSerde.GetKeySerdeG().UsedBufferPool(),
		vUseBuf: msgSerde.GetValSerdeG().UsedBufferPool(),
	}, nil
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) Name() string {
	return st.windowStore.Name()
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) ChangelogManager() *ChangelogManager[commtypes.KeyAndWindowStartTsG[K], V] {
	return st.changelogManager
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) OutputRemainingStats() {
	st.changelogManager.OutputRemainingStats()
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) Flush(ctx context.Context) (uint32, error) {
	return st.changelogManager.Flush(ctx)
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) Put(ctx context.Context,
	key K, value optional.Option[V], windowStartTimestamp int64, tm store.TimeMeta,
) error {
	keyTs := commtypes.KeyAndWindowStartTsG[K]{
		Key:           key,
		WindowStartTs: windowStartTimestamp,
	}
	msg := commtypes.MessageG[commtypes.KeyAndWindowStartTsG[K], V]{
		Key:   optional.Some(keyTs),
		Value: value,
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
		err := st.changelogManager.produce(ctx, msgSer, st.parNum)
		if err != nil {
			return err
		}
		// putStart := stats.TimerBegin()
		err = st.windowStore.Put(ctx, key, value, windowStartTimestamp, tm)
		// elapsed := stats.Elapsed(putStart).Microseconds()
		// st.storePutLatency.AddSample(elapsed)
		return err
	} else {
		log.Warn().Msgf("get empty key and value")
		return nil
	}
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error {
	kms[st.changelogManager.TopicName()] = make([]txn_data.KeyMaping, 0)
	hasher := hashfuncs.ByteSliceHasher{}
	return st.windowStore.IterAll(func(ts int64, key K, value V) error {
		kBytes, _, err := st.originKeySerde.Encode(key)
		if err != nil {
			return err
		}
		hash := hasher.HashSum64(kBytes)
		kms[st.changelogManager.TopicName()] = append(kms[st.changelogManager.TopicName()], txn_data.KeyMaping{
			Key:         kBytes,
			SubstreamId: st.parNum,
			Hash:        hash,
		})
		return nil
	})
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) PutWithoutPushToChangelog(ctx context.Context,
	key commtypes.KeyT, value commtypes.ValueT,
) error {
	keyTs := key.(commtypes.KeyAndWindowStartTsG[K])
	return st.windowStore.Put(ctx, keyTs.Key, optional.Some(value.(V)), keyTs.WindowStartTs, store.TimeMeta{RecordTsMs: 0})
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) PutWithoutPushToChangelogG(ctx context.Context,
	key K, value optional.Option[V], windowStartTs int64,
) error {
	return st.windowStore.Put(ctx, key, value, windowStartTs, store.TimeMeta{RecordTsMs: 0})
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) Get(ctx context.Context, key K, windowStartTimestamp int64) (V, bool, error) {
	return st.windowStore.Get(ctx, key, windowStartTimestamp)
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) Fetch(
	ctx context.Context,
	key K,
	timeFrom time.Time,
	timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	return st.windowStore.Fetch(ctx, key, timeFrom, timeTo, iterFunc)
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) IterAll(iterFunc func(int64, K, V) error) error {
	return st.windowStore.IterAll(iterFunc)
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) FetchWithKeyRange(
	ctx context.Context,
	keyFrom K,
	keyTo K,
	timeFrom time.Time,
	timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	return st.windowStore.FetchWithKeyRange(ctx, keyFrom, keyTo, timeFrom, timeTo, iterFunc)
}

func (st *InMemoryWindowStoreWithChangelogG[K, V]) FetchAll(
	ctx context.Context,
	timeFrom time.Time,
	timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	return st.windowStore.FetchAll(ctx, timeFrom, timeTo, iterFunc)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) TableType() store.TABLE_TYPE {
	return store.IN_MEM
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc) {
	s.trackFunc = trackParFunc
	s.trackFunc(s.changelogManager.TopicName(), s.parNum)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetStreamFlushCallbackFunc(cb exactly_once_intr.FlushCallbackFunc) {
	s.changelogManager.producer.SetFlushCallback(cb)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) ConsumeOneLogEntry(ctx context.Context, parNum uint8) (int, error) {
	msgSeq, err := s.changelogManager.Consume(ctx, parNum)
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
			err := s.PutWithoutPushToChangelogG(ctx, k.Key, msg.Value, k.WindowStartTs)
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
		err := s.PutWithoutPushToChangelogG(ctx, k.Key, msg.Value, k.WindowStartTs)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) ConfigureExactlyOnce(
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	s.changelogManager.ConfigExactlyOnce(rem, guarantee)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) ChangelogTopicName() string {
	return s.changelogManager.TopicName()
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) Stream() sharedlog_stream.Stream {
	return s.changelogManager.Stream()
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) GetInitialProdSeqNum() uint64 {
	return s.changelogManager.producer.GetInitialProdSeqNum(s.parNum)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) ResetInitialProd() {
	s.changelogManager.producer.ResetInitialProd()
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetLastMarkerSeq(lastMarkerSeq uint64) {
	s.changelogManager.producer.SetLastMarkerSeq(lastMarkerSeq)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SubstreamNum() uint8 {
	return s.parNum
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetCacheFlushCallback(store.WindowStoreCacheFlushCallbackFunc[K, V]) {
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) Snapshot(ctx context.Context, tplogOff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool) {
	s.windowStore.Snapshot(ctx, tplogOff, chkptMeta, resetBg)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) WaitForAllSnapshot() error {
	return s.windowStore.WaitForAllSnapshot()
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetWinSnapshotCallback(ctx context.Context, f store.WinSnapshotCallback[K, V]) {
	s.windowStore.SetWinSnapshotCallback(ctx, f)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error {
	return s.windowStore.RestoreFromSnapshot(ctx, snapshot)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat,
	keySerde commtypes.SerdeG[commtypes.KeyAndWindowStartTsG[K]],
	origSerde commtypes.SerdeG[K],
	valSerde commtypes.SerdeG[V],
) error {
	return nil
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]] {
	return s.windowStore.GetKVSerde()
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error) {
	return s.changelogManager.findLastEpochMetaWithAuxData(ctx, parNum)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) SetInstanceId(id uint8) {
	s.windowStore.SetInstanceId(id)
}

func (s *InMemoryWindowStoreWithChangelogG[K, V]) GetInstanceId() uint8 {
	return s.windowStore.GetInstanceId()
}

func ToInMemSkipMapWindowTableWithChangelogG[K, V any](
	joinWindow *commtypes.JoinWindows,
	retainDuplicates bool,
	comparable store.CompareFuncG[K],
	mp *MaterializeParam[K, V],
) (*processor.MeteredProcessorG[K, V, K, V], *InMemoryWindowStoreWithChangelogG[K, V], error) {
	winTab := store.NewInMemorySkipMapWindowStore[K, V](mp.storeName,
		joinWindow.MaxSize()+joinWindow.GracePeriodMs(), joinWindow.MaxSize(), retainDuplicates, comparable)
	winTab.SetInstanceId(mp.parNum)
	tabWithLog, err := NewInMemoryWindowStoreWithChangelogG[K, V](winTab, mp)
	if err != nil {
		return nil, nil, err
	}
	toTableProc := processor.NewMeteredProcessorG[K, V, K, V](processor.NewStoreToWindowTableProcessorG[K, V](tabWithLog))
	return toTableProc, tabWithLog, nil
}

func CreateInMemSkipMapWindowTableWithChangelogG[K, V any](
	joinWindow commtypes.EnumerableWindowDefinition,
	retainDuplicates bool,
	cmpFunc store.CompareFuncG[K],
	mp *MaterializeParam[K, V],
) (*InMemoryWindowStoreWithChangelogG[K, V], error) {
	winTab := store.NewInMemorySkipMapWindowStore[K, V](mp.storeName,
		joinWindow.MaxSize()+joinWindow.GracePeriodMs(), joinWindow.MaxSize(), retainDuplicates, cmpFunc)
	winTab.SetInstanceId(mp.parNum)
	return NewInMemoryWindowStoreWithChangelogG[K, V](winTab, mp)
}
