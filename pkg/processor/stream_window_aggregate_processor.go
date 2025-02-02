package processor

import (
	"context"
	"fmt"
	"os"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/store"

	"github.com/rs/zerolog/log"
)

/*
type StreamWindowAggregateProcessor struct {
	store       store.CoreWindowStore
	initializer Initializer
	aggregator  Aggregator
	windows     commtypes.EnumerableWindowDefinition
	name        string
	BaseProcessor
	observedStreamTime int64
}

var _ = Processor(&StreamWindowAggregateProcessor{})

func NewStreamWindowAggregateProcessor(name string, store store.CoreWindowStore, initializer Initializer,
	aggregator Aggregator, windows commtypes.EnumerableWindowDefinition,
) *StreamWindowAggregateProcessor {
	p := &StreamWindowAggregateProcessor{
		initializer:        initializer,
		aggregator:         aggregator,
		observedStreamTime: 0,
		store:              store,
		windows:            windows,
		BaseProcessor:      BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamWindowAggregateProcessor) Name() string {
	return p.name
}

func (p *StreamWindowAggregateProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Key == nil {
		log.Warn().Msgf("skipping record due to null key. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}
	// debug.Fprintf(os.Stderr, "stream window agg msg k %v, v %v, ts %d\n", msg.Key, msg.Value, msg.Timestamp)
	ts := msg.Timestamp
	if p.observedStreamTime < ts {
		p.observedStreamTime = ts
	}
	closeTime := p.observedStreamTime - p.windows.GracePeriodMs()
	matchedWindows, keys, err := p.windows.WindowsFor(ts)
	if err != nil {
		return nil, fmt.Errorf("windows for err %v", err)
	}
	newMsgs := make([]commtypes.Message, 0)
	for _, windowStart := range keys {
		window := matchedWindows[windowStart]
		windowEnd := window.End()
		if windowEnd > closeTime {
			var oldAgg interface{}
			var newTs int64
			val, exists, err := p.store.Get(ctx, msg.Key, windowStart)
			if err != nil {
				return nil, fmt.Errorf("win agg get err %v", err)
			}
			if exists {
				oldAggTs, ok := val.(*commtypes.ValueTimestamp)
				if !ok {
					oldAggTsTmp := val.(commtypes.ValueTimestamp)
					oldAggTs = &oldAggTsTmp
				}
				oldAgg = oldAggTs.Value
				if msg.Timestamp > oldAggTs.Timestamp {
					newTs = msg.Timestamp
				} else {
					newTs = oldAggTs.Timestamp
				}
			} else {
				oldAgg = p.initializer.Apply()
				newTs = msg.Timestamp
			}
			newAgg := p.aggregator.Apply(msg.Key, msg.Value, oldAgg)
			err = p.store.Put(ctx, msg.Key, commtypes.CreateValueTimestamp(newAgg, newTs), windowStart)
			if err != nil {
				return nil, fmt.Errorf("win agg put err %v", err)
			}
			newMsgs = append(newMsgs, commtypes.Message{
				Key:       &commtypes.WindowedKey{Key: msg.Key, Window: window},
				Value:     commtypes.Change{NewVal: newAgg, OldVal: oldAgg},
				Timestamp: newTs,
			})
		} else {
			log.Warn().Interface("key", msg.Key).
				Interface("value", msg.Value).
				Int64("timestamp", msg.Timestamp).Msg("Skipping record for expired window. ")
		}
	}
	return newMsgs, nil
}
*/

type StreamWindowAggregateProcessorG[K, V, VA any] struct {
	store       store.CachedWindowStateStore[K, commtypes.ValueTimestampG[VA]]
	initializer InitializerG[VA]
	aggregator  AggregatorG[K, V, VA]
	windows     commtypes.EnumerableWindowDefinition
	name        string
	BaseProcessorG[K, V, commtypes.WindowedKeyG[K], commtypes.ChangeG[VA]]
	observedStreamTime int64
	useCache           bool
}

func NewStreamWindowAggregateProcessorG[K, V, VA any](name string,
	store store.CachedWindowStateStore[K, commtypes.ValueTimestampG[VA]],
	initializer InitializerG[VA], aggregator AggregatorG[K, V, VA],
	windows commtypes.EnumerableWindowDefinition,
	useCache bool,
) ProcessorG[K, V, commtypes.WindowedKeyG[K], commtypes.ChangeG[VA]] {
	p := &StreamWindowAggregateProcessorG[K, V, VA]{
		initializer:        initializer,
		aggregator:         aggregator,
		observedStreamTime: 0,
		store:              store,
		windows:            windows,
		useCache:           useCache,
	}
	fmt.Fprintf(os.Stderr, "StreamWindowAggregateProc useCache: %v\n", useCache)
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	if useCache {
		store.SetCacheFlushCallback(func(ctx context.Context, msg commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[commtypes.ValueTimestampG[VA]]]) error {
			change := msg.Value.Unwrap()
			oldVal := optional.Map(change.OldVal, func(oldVal commtypes.ValueTimestampG[VA]) VA {
				return oldVal.Value
			})
			newVal := optional.Map(change.NewVal, func(newVal commtypes.ValueTimestampG[VA]) VA {
				return newVal.Value
			})
			ts := int64(0)
			if change.NewVal.IsSome() {
				newValTs := change.NewVal.Unwrap()
				ts = newValTs.Timestamp
			} else {
				ts = msg.TimestampMs
			}
			v := commtypes.ChangeG[VA]{
				NewVal: newVal,
				OldVal: oldVal,
			}
			msgForNext := commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[VA]]{
				Key:           msg.Key,
				Value:         optional.Some(v),
				TimestampMs:   ts,
				StartProcTime: msg.StartProcTime,
			}
			for _, nextProcessor := range p.nextProcessors {
				err := nextProcessor.Process(ctx, msgForNext)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	return p
}

func (p *StreamWindowAggregateProcessorG[K, V, VA]) Name() string {
	return p.name
}

func (p *StreamWindowAggregateProcessorG[K, V, VA]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) (
	[]commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[VA]], error,
) {
	if msg.Key.IsNone() {
		log.Warn().Msgf("skipping record due to null key. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}
	// debug.Fprintf(os.Stderr, "stream window agg msg k %v, v %v, ts %d\n", msg.Key, msg.Value, msg.Timestamp)
	ts := msg.TimestampMs
	if p.observedStreamTime < ts {
		p.observedStreamTime = ts
	}
	msgKey := msg.Key.Unwrap()
	msgVal := msg.Value.Unwrap()
	closeTime := p.observedStreamTime - p.windows.GracePeriodMs()
	matchedWindows, keys, err := p.windows.WindowsFor(ts)
	if err != nil {
		return nil, fmt.Errorf("windows for err %v", err)
	}
	newMsgs := make([]commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[VA]], 0)
	for _, windowStart := range keys {
		window := matchedWindows[windowStart]
		windowEnd := window.End()
		if windowEnd > closeTime {
			var oldAgg optional.Option[VA]
			var newTs int64
			oldAggTs, exists, err := p.store.Get(ctx, msgKey, windowStart)
			if err != nil {
				return nil, fmt.Errorf("win agg get err %v", err)
			}
			if exists {
				oldAgg = optional.Some(oldAggTs.Value)
				if msg.TimestampMs > oldAggTs.Timestamp {
					newTs = msg.TimestampMs
				} else {
					newTs = oldAggTs.Timestamp
				}
			} else {
				oldAgg = p.initializer.Apply()
				newTs = msg.TimestampMs
			}
			newAgg := p.aggregator.Apply(msgKey, msgVal, oldAgg)
			err = p.store.Put(ctx, msgKey, commtypes.CreateValueTimestampGOptional(newAgg, newTs), windowStart,
				store.TimeMeta{RecordTsMs: newTs, StartProcTs: msg.StartProcTime})
			if err != nil {
				return nil, fmt.Errorf("win agg put err %v", err)
			}
			newMsgs = append(newMsgs, commtypes.MessageG[commtypes.WindowedKeyG[K], commtypes.ChangeG[VA]]{
				Key:           optional.Some(commtypes.WindowedKeyG[K]{Key: msgKey, Window: window}),
				Value:         optional.Some(commtypes.ChangeG[VA]{NewVal: newAgg, OldVal: oldAgg}),
				TimestampMs:   newTs,
				StartProcTime: msg.StartProcTime,
			})
		} else {
			log.Warn().Interface("key", msg.Key).
				Interface("value", msg.Value).
				Int64("timestamp", msg.TimestampMs).Msg("Skipping record for expired window. ")
		}
	}
	if p.useCache {
		return nil, nil
	} else {
		return newMsgs, nil
	}
}
