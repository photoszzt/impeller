package store

import (
	"context"
	"math"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/hashfuncs"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/sharedlog_stream"
	"sharedlog-stream/pkg/txn_data"
	"sharedlog-stream/pkg/utils"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhangyunhao116/skipmap"
	"golang.org/x/sync/errgroup"
)

func updateSeqnumForDups(retainDuplicates bool, seqNum *uint32) {
	if retainDuplicates {
		atomic.CompareAndSwapUint32(seqNum, math.MaxUint32, 0)
		atomic.AddUint32(seqNum, 1)
	}
}

type WinSnapshotCallback[K, V any] func(
	ctx context.Context,
	tplogOff []commtypes.TpLogOff,
	chkptMeta []commtypes.ChkptMetaData,
	snapshot []*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]) error

type InMemorySkipMapWindowStoreG[K, V any] struct {
	mux                         sync.RWMutex
	storeNoDup                  *skipmap.Int64Map[*skipmap.FuncMap[K, V]]
	storeWithDup                *skipmap.Int64Map[*skipmap.FuncMap[VersionedKeyG[K], V]]
	compareFunc                 CompareFuncG[K]
	compareFuncWithVersionedKey CompareFuncG[VersionedKeyG[K]]
	kvPairSerdeG                commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]]
	keySerdeG                   commtypes.SerdeG[K]
	snapshotCallback            WinSnapshotCallback[K, V]
	bgErrG                      *errgroup.Group
	bgCtx                       context.Context
	name                        string
	windowSize                  int64
	retentionPeriod             int64
	observedStreamTime          int64 // protected by mux
	seqNum                      uint32
	retainDuplicates            bool
	insId                       uint8
}

var (
	_ = CoreWindowStoreG[int, int](&InMemorySkipMapWindowStoreG[int, int]{})
	_ = CachedWindowStateStore[int, int](&InMemorySkipMapWindowStoreG[int, int]{})
)

func NewInMemorySkipMapWindowStore[K, V any](name string, retentionPeriod int64,
	windowSize int64, retainDuplicates bool, compareFunc CompareFuncG[K],
) *InMemorySkipMapWindowStoreG[K, V] {
	if retainDuplicates {
		return &InMemorySkipMapWindowStoreG[K, V]{
			name:               name,
			windowSize:         windowSize,
			retentionPeriod:    retentionPeriod,
			retainDuplicates:   retainDuplicates,
			observedStreamTime: 0,
			storeWithDup:       skipmap.NewInt64[*skipmap.FuncMap[VersionedKeyG[K], V]](),
			storeNoDup:         nil,
			compareFuncWithVersionedKey: func(lhs, rhs VersionedKeyG[K]) int {
				return CompareWithVersionedKey(lhs, rhs, compareFunc)
			},
		}
	} else {
		return &InMemorySkipMapWindowStoreG[K, V]{
			name:               name,
			windowSize:         windowSize,
			retentionPeriod:    retentionPeriod,
			retainDuplicates:   retainDuplicates,
			observedStreamTime: 0,
			storeNoDup:         skipmap.NewInt64[*skipmap.FuncMap[K, V]](),
			storeWithDup:       nil,
			compareFunc:        compareFunc,
		}
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) SetWinSnapshotCallback(ctx context.Context, f WinSnapshotCallback[K, V]) {
	s.bgErrG, s.bgCtx = errgroup.WithContext(ctx)
	s.snapshotCallback = f
}

func (s *InMemorySkipMapWindowStoreG[K, V]) SetKVSerde(serdeFormat commtypes.SerdeFormat,
	keySerde commtypes.SerdeG[commtypes.KeyAndWindowStartTsG[K]],
	origKeySerde commtypes.SerdeG[K],
	valSerde commtypes.SerdeG[V],
) error {
	var err error
	s.keySerdeG = origKeySerde
	s.kvPairSerdeG, err = commtypes.GetKeyValuePairSerdeG(serdeFormat, keySerde, valSerde)
	return err
}

func (s *InMemorySkipMapWindowStoreG[K, V]) GetKVSerde() commtypes.SerdeG[*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]] {
	return s.kvPairSerdeG
}

func (s *InMemorySkipMapWindowStoreG[K, V]) Name() string { return s.name }
func (s *InMemorySkipMapWindowStoreG[K, V]) Put(ctx context.Context, key K, value optional.Option[V], windowStartTimestamp int64, tm TimeMeta) error {
	s.removeExpiredSegments()
	s.mux.Lock()
	if windowStartTimestamp > s.observedStreamTime {
		s.observedStreamTime = windowStartTimestamp
	}
	expired := windowStartTimestamp <= s.observedStreamTime-s.retentionPeriod
	s.mux.Unlock()
	// debug.Fprintf(os.Stderr, "start ts: %d, observed: %d\n", windowStartTimestamp, s.observedStreamTime-s.retentionPeriod)
	if expired {
		log.Warn().Msgf("Skipping record for expired segment.")
	} else {
		val, exists := value.Take()
		if exists {
			updateSeqnumForDups(s.retainDuplicates, &s.seqNum)
			if s.retainDuplicates {
				k := VersionedKeyG[K]{
					Version: atomic.LoadUint32(&s.seqNum),
					Key:     key,
				}
				actual, _ := s.storeWithDup.LoadOrStore(windowStartTimestamp, skipmap.NewFunc[VersionedKeyG[K], V](
					func(a, b VersionedKeyG[K]) bool {
						return s.compareFuncWithVersionedKey(a, b) < 0
					}))
				actual.Store(k, val)
			} else {
				actual, _ := s.storeNoDup.LoadOrStore(windowStartTimestamp, skipmap.NewFunc[K, V](func(a, b K) bool {
					return s.compareFunc(a, b) < 0
				}))
				actual.Store(key, val)
			}
		} else if !s.retainDuplicates {
			// Skip if value is null and duplicates are allowed since this delete is a no-op
			v, ok := s.storeNoDup.Load(windowStartTimestamp)
			if ok {
				v.Delete(key)
			}
		}
	}
	return nil
}

// Get is used in aggregates; it doesn't expects duplicates
func (s *InMemorySkipMapWindowStoreG[K, V]) Get(ctx context.Context, key K, windowStartTimestamp int64) (V, bool, error) {
	var v V
	s.removeExpiredSegments()
	s.mux.RLock()
	expired := windowStartTimestamp <= s.observedStreamTime-s.retentionPeriod
	s.mux.RUnlock()
	if expired {
		return v, false, nil
	}
	if s.retainDuplicates {
		panic("Get is not supported for window stores with duplicates")
	} else {
		kvmap, ok := s.storeNoDup.Load(windowStartTimestamp)
		if !ok {
			return v, false, nil
		}
		v, exists := kvmap.Load(key)
		return v, exists, nil
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) PutWithoutPushToChangelog(
	ctx context.Context, key commtypes.KeyT, value commtypes.ValueT,
) error {
	keyTs := key.(commtypes.KeyAndWindowStartTsG[K])
	if utils.IsNil(value) {
		return s.Put(ctx, keyTs.Key, optional.None[V](), keyTs.WindowStartTs, TimeMeta{RecordTsMs: 0})
	} else {
		return s.Put(ctx, key.(K), optional.Some(value.(V)), keyTs.WindowStartTs, TimeMeta{RecordTsMs: 0})
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) Fetch(ctx context.Context, key K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64 /* ts */, K, V) error,
) error {
	s.removeExpiredSegments()
	tsFrom := timeFrom.UnixMilli()
	tsTo := timeTo.UnixMilli()

	s.mux.RLock()
	minTime := s.observedStreamTime - s.retentionPeriod + 1
	s.mux.RUnlock()
	if minTime < tsFrom {
		minTime = tsFrom
	}

	if tsTo < minTime {
		return nil
	}

	if s.retainDuplicates {
		keyFrom := VersionedKeyG[K]{
			Version: 0,
			Key:     key,
		}
		keyTo := VersionedKeyG[K]{
			Version: math.MaxUint32,
			Key:     key,
		}
		return s.fetchWithKeyRangeWithDuplicates(keyFrom, keyTo, tsFrom, tsTo, iterFunc)
	} else {
		s.storeNoDup.RangeFrom(tsFrom, func(ts int64, kvmap *skipmap.FuncMap[K, V]) bool {
			if ts > tsTo {
				return false
			} else if ts < tsFrom {
				return true
			}
			v, exists := kvmap.Load(key)
			if exists {
				err := iterFunc(ts, key, v)
				if err != nil {
					return false
				}
			}
			return true
		})
		return nil
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) FetchWithKeyRange(ctx context.Context, keyFrom K,
	keyTo K, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	s.removeExpiredSegments()

	tsFrom := timeFrom.UnixMilli()
	tsTo := timeTo.UnixMilli()

	if tsFrom > tsTo {
		return nil
	}

	s.mux.RLock()
	minTime := s.observedStreamTime - s.retentionPeriod + 1
	s.mux.RUnlock()
	if minTime < tsFrom {
		minTime = tsFrom
	}

	if tsTo < minTime {
		return nil
	}

	if s.retainDuplicates {
		kFrom := VersionedKeyG[K]{
			Version: 0,
			Key:     keyFrom,
		}
		kTo := VersionedKeyG[K]{
			Version: math.MaxUint32,
			Key:     keyTo,
		}
		return s.fetchWithKeyRangeWithDuplicates(kFrom, kTo, tsFrom, tsTo, iterFunc)
	} else {
		s.storeNoDup.RangeFrom(tsFrom, func(ts int64, kvmap *skipmap.FuncMap[K, V]) bool {
			if ts > tsTo {
				return false
			} else if ts < tsFrom {
				return true
			}
			kvmap.RangeFrom(keyFrom, func(k K, v V) bool {
				if s.compareFunc(k, keyTo) <= 0 {
					err := iterFunc(ts, k, v)
					return err == nil
				} else {
					return false
				}
			})
			return true
		})
		return nil
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) fetchWithKeyRangeWithDuplicates(
	keyFrom VersionedKeyG[K], keyTo VersionedKeyG[K], timeFrom int64, timeTo int64,
	iterFunc func(int64, K, V) error,
) error {
	// fmt.Fprintf(os.Stderr, "keyFrom: %+v, keyTo: %+v, timeFrom: %v, timeTo: %v\n",
	// 	keyFrom, keyTo, timeFrom, timeTo)
	s.storeWithDup.RangeFrom(timeFrom, func(ts int64, kvmap *skipmap.FuncMap[VersionedKeyG[K], V]) bool {
		if ts > timeTo {
			return false
		} else if ts < timeFrom {
			return true
		}
		// fmt.Fprintf(os.Stderr, "current ts: %v\n", ts)
		kvmap.RangeFrom(keyFrom, func(k VersionedKeyG[K], v V) bool {
			// fmt.Fprintf(os.Stderr, "current key: %+v\n", k)
			if s.compareFuncWithVersionedKey(k, keyTo) <= 0 {
				err := iterFunc(ts, k.Key, v)
				return err == nil
			} else {
				return false
			}
		})
		return true
	})
	return nil
}

func (s *InMemorySkipMapWindowStoreG[K, V]) FetchAll(ctx context.Context, timeFrom time.Time, timeTo time.Time,
	iterFunc func(int64, K, V) error,
) error {
	s.removeExpiredSegments()

	tsFrom := timeFrom.UnixMilli()
	tsTo := timeTo.UnixMilli()

	if tsFrom > tsTo {
		return nil
	}

	s.mux.RLock()
	minTime := s.observedStreamTime - s.retentionPeriod + 1
	s.mux.RUnlock()
	if minTime < tsFrom {
		minTime = tsFrom
	}

	if tsTo < minTime {
		return nil
	}
	if s.retainDuplicates {
		s.storeWithDup.RangeFrom(tsFrom, func(ts int64, kvmap *skipmap.FuncMap[VersionedKeyG[K], V]) bool {
			if ts < tsFrom {
				return true
			} else if ts > tsTo {
				return false
			} else {
				kvmap.Range(func(k VersionedKeyG[K], v V) bool {
					err := iterFunc(ts, k.Key, v)
					return err == nil
				})
				return true
			}
		})
	} else {
		s.storeNoDup.RangeFrom(tsFrom, func(ts int64, kvmap *skipmap.FuncMap[K, V]) bool {
			if ts < tsFrom {
				return true
			} else if ts > tsTo {
				return false
			} else {
				kvmap.Range(func(k K, v V) bool {
					err := iterFunc(ts, k, v)
					return err == nil
				})
				return true
			}
		})
	}
	return nil
}

func (s *InMemorySkipMapWindowStoreG[K, V]) IterAll(iterFunc func(int64, K, V) error) error {
	s.removeExpiredSegments()
	s.mux.RLock()
	minTime := s.observedStreamTime - s.retentionPeriod
	s.mux.RUnlock()
	if s.retainDuplicates {
		s.storeWithDup.RangeFrom(minTime, func(ts int64, kvmap *skipmap.FuncMap[VersionedKeyG[K], V]) bool {
			if ts < minTime {
				return true
			}
			kvmap.Range(func(k VersionedKeyG[K], v V) bool {
				err := iterFunc(ts, k.Key, v)
				return err == nil
			})
			return true
		})
	} else {
		s.storeNoDup.RangeFrom(minTime, func(ts int64, kvmap *skipmap.FuncMap[K, V]) bool {
			if ts < minTime {
				return true
			}
			kvmap.Range(func(k K, v V) bool {
				err := iterFunc(ts, k, v)
				return err == nil
			})
			return true
		})
	}
	return nil
}

func (s *InMemorySkipMapWindowStoreG[K, V]) WaitForAllSnapshot() error {
	debug.Assert(s.bgErrG != nil, "bgErr should not be nil")
	return s.bgErrG.Wait()
}

// not thread-safe
func (s *InMemorySkipMapWindowStoreG[K, V]) Snapshot(ctx context.Context,
	tpLogoff []commtypes.TpLogOff, chkptMeta []commtypes.ChkptMetaData, resetBg bool,
) {
	debug.Assert(s.bgErrG != nil, "bgErr should not be nil")
	debug.Assert(s.snapshotCallback != nil, "snapshotCallback should not be nil")
	l := 0
	if s.retainDuplicates {
		l = s.storeWithDup.Len()
	} else {
		l = s.storeNoDup.Len()
	}
	// outBin := make([][]byte, 0, l)
	out := make([]*commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V], 0, l)
	if chkptMeta != nil || resetBg {
		s.bgErrG, s.bgCtx = errgroup.WithContext(ctx)
	}
	if s.retainDuplicates {
		// cpyBeg := time.Now()
		s.storeWithDup.Range(func(ts int64, kvmap *skipmap.FuncMap[VersionedKeyG[K], V]) bool {
			kvmap.Range(func(k VersionedKeyG[K], v V) bool {
				p := &commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]{
					Key:   commtypes.KeyAndWindowStartTsG[K]{Key: k.Key, WindowStartTs: ts},
					Value: v,
				}
				out = append(out, p)
				return true
			})
			return true
		})
		s.bgErrG.Go(func() error {
			return s.snapshotCallback(s.bgCtx, tpLogoff, chkptMeta, out)
		})
	} else {
		// cpyBeg := time.Now()
		s.storeNoDup.Range(func(ts int64, kvmap *skipmap.FuncMap[K, V]) bool {
			kvmap.Range(func(k K, v V) bool {
				p := &commtypes.KeyValuePair[commtypes.KeyAndWindowStartTsG[K], V]{
					Key:   commtypes.KeyAndWindowStartTsG[K]{Key: k, WindowStartTs: ts},
					Value: v,
				}
				out = append(out, p)
				return true
			})
			return true
		})
		// cpyElapsed := time.Since(cpyBeg)
		s.bgErrG.Go(func() error {
			return s.snapshotCallback(s.bgCtx, tpLogoff, chkptMeta, out)
		})
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) RestoreFromSnapshot(ctx context.Context, snapshot [][]byte) error {
	for _, penc := range snapshot {
		p, err := s.kvPairSerdeG.Decode(penc)
		if err != nil {
			log.Error().Err(err).Msg("failed to decode key-value pair")
			return err
		}
		err = s.Put(ctx, p.Key.Key, optional.Some(p.Value), p.Key.WindowStartTs, TimeMeta{RecordTsMs: 0})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *InMemorySkipMapWindowStoreG[K, V]) TableType() TABLE_TYPE { return IN_MEM }
func (s *InMemorySkipMapWindowStoreG[K, V]) SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc) {
}

func (s *InMemorySkipMapWindowStoreG[K, V]) PutWithoutPushToChangelogG(ctx context.Context, key K, value optional.Option[V], windowStartTs int64) error {
	return s.Put(ctx, key, value, windowStartTs, TimeMeta{RecordTsMs: 0})
}
func (s *InMemorySkipMapWindowStoreG[K, V]) Flush(ctx context.Context) (uint32, error) { return 0, nil }
func (s *InMemorySkipMapWindowStoreG[K, V]) ConsumeOneLogEntry(ctx context.Context, parNum uint8) (int, error) {
	panic("not supported")
}

func (s *InMemorySkipMapWindowStoreG[K, V]) ConfigureExactlyOnce(
	rem exactly_once_intr.ReadOnlyExactlyOnceManager,
	guarantee exactly_once_intr.GuaranteeMth,
) {
	panic("not supported")
}

func (s *InMemorySkipMapWindowStoreG[K, V]) ChangelogTopicName() string {
	panic("not supported")
}
func (s *InMemorySkipMapWindowStoreG[K, V]) GetInitialProdSeqNum() uint64    { panic("not supported") }
func (s *InMemorySkipMapWindowStoreG[K, V]) SetLastMarkerSeq(seq uint64)     { panic("not supported") }
func (s *InMemorySkipMapWindowStoreG[K, V]) ResetInitialProd()               { panic("not supported") }
func (s *InMemorySkipMapWindowStoreG[K, V]) Stream() sharedlog_stream.Stream { panic("not supported") }
func (s *InMemorySkipMapWindowStoreG[K, V]) SubstreamNum() uint8             { panic("not supported") }
func (s *InMemorySkipMapWindowStoreG[K, V]) FindLastEpochMetaWithAuxData(ctx context.Context, parNum uint8) (auxData []byte, metaSeqNum uint64, err error) {
	panic("not supported")
}

func (s *InMemorySkipMapWindowStoreG[K, V]) removeExpiredSegments() {
	s.mux.RLock()
	minLiveTimeTmp := int64(s.observedStreamTime) - int64(s.retentionPeriod) + 1
	s.mux.RUnlock()
	minLiveTime := int64(0)

	if minLiveTimeTmp > 0 {
		minLiveTime = int64(minLiveTimeTmp)
	}

	if s.retainDuplicates {
		s.storeWithDup.Range(func(ts int64, _ *skipmap.FuncMap[VersionedKeyG[K], V]) bool {
			if ts < minLiveTime {
				s.storeWithDup.Delete(ts)
				return true
			} else {
				return false
			}
		})
	} else {
		s.storeNoDup.Range(func(ts int64, _ *skipmap.FuncMap[K, V]) bool {
			if ts < minLiveTime {
				s.storeNoDup.Delete(ts)
				return true
			} else {
				return false
			}
		})
	}
}

func (s *InMemorySkipMapWindowStoreG[K, V]) SetCacheFlushCallback(
	WindowStoreCacheFlushCallbackFunc[K, V]) {
}

func (s *InMemorySkipMapWindowStoreG[K, V]) SetStreamFlushCallbackFunc(
	exactly_once_intr.FlushCallbackFunc) {
}

func (s *InMemorySkipMapWindowStoreG[K, V]) BuildKeyMeta(kms map[string][]txn_data.KeyMaping) error {
	kms[s.Name()] = make([]txn_data.KeyMaping, 0)
	hasher := hashfuncs.ByteSliceHasher{}
	return s.IterAll(func(ts int64, key K, value V) error {
		kBytes, _, err := s.keySerdeG.Encode(key)
		if err != nil {
			return err
		}
		hash := hasher.HashSum64(kBytes)
		kms[s.Name()] = append(kms[s.Name()], txn_data.KeyMaping{
			Key:         kBytes,
			SubstreamId: s.insId,
			Hash:        hash,
		})
		return nil
	})
}

func (s *InMemorySkipMapWindowStoreG[K, V]) SetInstanceId(id uint8) {
	s.insId = id
}

func (s *InMemorySkipMapWindowStoreG[K, V]) GetInstanceId() uint8 {
	return s.insId
}

func (s *InMemorySkipMapWindowStoreG[K, V]) OutputRemainingStats() {}
