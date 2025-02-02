package processor

import (
	"context"
	"math"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/store"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type TimeTracker struct {
	mux            sync.Mutex
	emitIntervalMs uint64
	streamTime     int64
	minTime        int64
	nextTimeToEmit uint64
}

func NewTimeTracker() *TimeTracker {
	return &TimeTracker{
		emitIntervalMs: 1000,
		streamTime:     0,
		minTime:        math.MaxInt64,
	}
}

func (t *TimeTracker) SetEmitInterval(emitIntervalMs uint64) {
	t.emitIntervalMs = emitIntervalMs
}

func (t *TimeTracker) AdvanceStreamTime(recordTs int64) {
	t.mux.Lock()
	if recordTs > t.streamTime {
		t.streamTime = recordTs
	}
	t.mux.Unlock()
}

func (t *TimeTracker) UpdatedMinTime(recordTs int64) {
	t.mux.Lock()
	if recordTs < t.minTime {
		t.minTime = recordTs
	}
	t.mux.Unlock()
}

func (t *TimeTracker) AdvanceNextTimeToEmit() {
	t.mux.Lock()
	t.nextTimeToEmit += t.emitIntervalMs
	t.mux.Unlock()
}

/*
type StreamStreamJoinProcessor struct {
	otherWindowStore  store.CoreWindowStore
	joiner            ValueJoinerWithKeyTs
	sharedTimeTracker *TimeTracker
	name              string
	BaseProcessor
	joinAfterMs  int64
	joinGraceMs  int64
	joinBeforeMs int64
	outer        bool
	isLeftSide   bool
}

var _ = Processor(&StreamStreamJoinProcessor{})

func NewStreamStreamJoinProcessor(
	name string,
	otherWindowStore store.CoreWindowStore,
	jw *commtypes.JoinWindows,
	joiner ValueJoinerWithKeyTs,
	outer bool,
	isLeftSide bool,
	stk *TimeTracker,
) *StreamStreamJoinProcessor {
	ssjp := &StreamStreamJoinProcessor{
		isLeftSide:        isLeftSide,
		outer:             outer,
		otherWindowStore:  otherWindowStore,
		joiner:            joiner,
		joinGraceMs:       jw.GracePeriodMs(),
		sharedTimeTracker: stk,
		name:              name,
		BaseProcessor:     BaseProcessor{},
	}
	if isLeftSide {
		ssjp.joinBeforeMs = jw.BeforeMs()
		ssjp.joinAfterMs = jw.AfterMs()
	} else {
		ssjp.joinBeforeMs = jw.AfterMs()
		ssjp.joinAfterMs = jw.BeforeMs()
	}
	ssjp.BaseProcessor.ProcessingFunc = ssjp.ProcessAndReturn
	return ssjp
}

func (p *StreamStreamJoinProcessor) Name() string {
	return p.name
}

func (p *StreamStreamJoinProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Key == nil || msg.Value == nil {
		log.Warn().Msgf("skipping record due to null key or value. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}

	// needOuterJoin := p.outer
	inputTs := msg.Timestamp
	// debug.Fprintf(os.Stderr, "input ts: %v\n", inputTs)
	var timeFrom uint64
	var timeTo uint64
	timeFromTmp := int64(inputTs) - int64(p.joinBeforeMs)
	if timeFromTmp < 0 {
		timeFrom = 0
	} else {
		timeFrom = uint64(timeFromTmp)
	}
	timeToTmp := inputTs + int64(p.joinAfterMs)
	if timeToTmp > 0 {
		timeTo = uint64(timeToTmp)
	} else {
		timeTo = 0
	}
	// sharedTimeTracker is only used for outer join
	p.sharedTimeTracker.AdvanceStreamTime(inputTs)

	// TODO: emit all non-joined records which window has closed
	timeFromSec := timeFrom / 1000
	timeFromNs := (timeFrom - timeFromSec*1000) * 1000000

	timeToSec := timeTo / 1000
	timeToNs := (timeTo - timeToSec*1000) * 1000000
	msgs := make([]commtypes.Message, 0)
	err := p.otherWindowStore.Fetch(ctx, msg.Key,
		time.Unix(int64(timeFromSec), int64(timeFromNs)),
		time.Unix(int64(timeToSec), int64(timeToNs)), func(otherRecordTs int64, kt commtypes.KeyT, vt commtypes.ValueT) error {
			var newTs int64
			// needOuterJoin = false
			newVal := p.joiner.Apply(kt, msg.Value, vt, msg.Timestamp, otherRecordTs)
			if inputTs > otherRecordTs {
				newTs = inputTs
			} else {
				newTs = otherRecordTs
			}
			msgs = append(msgs, commtypes.Message{Key: msg.Key, Value: newVal, Timestamp: newTs})
			return nil
		})
	return msgs, err
		// if needOuterJoin {
		// 	// TODO
		// }
}
*/

type StreamStreamJoinProcessorG[K, V, VO, VR any] struct {
	otherWindowStore  store.CoreWindowStoreG[K, VO]
	joiner            ValueJoinerWithKeyTsG[K, V, VO, VR]
	sharedTimeTracker *TimeTracker
	name              string
	BaseProcessorG[K, V, K, VR]
	joinAfterMs  int64
	joinGraceMs  int64
	joinBeforeMs int64
	outer        bool
	isLeftSide   bool
}

var _ = ProcessorG[int, int, int, int](&StreamStreamJoinProcessorG[int, int, int, int]{})

func NewStreamStreamJoinProcessorG[K, V1, V2, VR any](
	name string,
	otherWindowStore store.CoreWindowStoreG[K, V2],
	jw *commtypes.JoinWindows,
	joiner ValueJoinerWithKeyTsG[K, V1, V2, VR],
	outer bool,
	isLeftSide bool,
	stk *TimeTracker,
) *StreamStreamJoinProcessorG[K, V1, V2, VR] {
	ssjp := &StreamStreamJoinProcessorG[K, V1, V2, VR]{
		isLeftSide:        isLeftSide,
		outer:             outer,
		otherWindowStore:  otherWindowStore,
		joiner:            joiner,
		joinGraceMs:       jw.GracePeriodMs(),
		sharedTimeTracker: stk,
		name:              name,
	}
	if isLeftSide {
		ssjp.joinBeforeMs = jw.BeforeMs()
		ssjp.joinAfterMs = jw.AfterMs()
	} else {
		ssjp.joinBeforeMs = jw.AfterMs()
		ssjp.joinAfterMs = jw.BeforeMs()
	}
	ssjp.BaseProcessorG.ProcessingFuncG = ssjp.ProcessAndReturn
	return ssjp
}

func (p *StreamStreamJoinProcessorG[K, V1, V2, VR]) Name() string {
	return p.name
}

func (p *StreamStreamJoinProcessorG[K, V1, V2, VR]) ProcessAndReturn(
	ctx context.Context, msg commtypes.MessageG[K, V1],
) ([]commtypes.MessageG[K, VR], error) {
	if msg.Key.IsNone() || msg.Value.IsNone() {
		log.Warn().Msgf("skipping record due to null key or value. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}

	// needOuterJoin := p.outer
	inputTs := msg.TimestampMs
	// debug.Fprintf(os.Stderr, "input ts: %v\n", inputTs)
	var timeFrom uint64
	var timeTo uint64
	timeFromTmp := int64(inputTs) - int64(p.joinBeforeMs)
	if timeFromTmp < 0 {
		timeFrom = 0
	} else {
		timeFrom = uint64(timeFromTmp)
	}
	timeToTmp := inputTs + int64(p.joinAfterMs)
	if timeToTmp > 0 {
		timeTo = uint64(timeToTmp)
	} else {
		timeTo = 0
	}
	// sharedTimeTracker is only used for outer join
	p.sharedTimeTracker.AdvanceStreamTime(inputTs)

	// TODO: emit all non-joined records which window has closed
	timeFromSec := timeFrom / 1000
	timeFromNs := (timeFrom - timeFromSec*1000) * 1000000

	timeToSec := timeTo / 1000
	timeToNs := (timeTo - timeToSec*1000) * 1000000
	msgs := make([]commtypes.MessageG[K, VR], 0, 2)
	key := msg.Key.Unwrap()
	msgVal := msg.Value.Unwrap()
	err := p.otherWindowStore.Fetch(ctx, key,
		time.Unix(int64(timeFromSec), int64(timeFromNs)),
		time.Unix(int64(timeToSec), int64(timeToNs)), func(otherRecordTs int64, kt K, vt V2) error {
			var newTs int64
			// needOuterJoin = false
			newVal := p.joiner.Apply(kt, msgVal, vt, msg.TimestampMs, otherRecordTs)
			if inputTs > otherRecordTs {
				newTs = inputTs
			} else {
				newTs = otherRecordTs
			}
			msgs = append(msgs, commtypes.MessageG[K, VR]{Key: msg.Key, Value: newVal,
				TimestampMs: newTs, StartProcTime: msg.StartProcTime})
			return nil
		})
	return msgs, err
	// if needOuterJoin {
	// 	// TODO
	// }
}
