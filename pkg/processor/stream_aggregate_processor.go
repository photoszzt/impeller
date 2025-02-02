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

type StreamAggregateProcessorG[K, V, VA any] struct {
	store       store.CachedKeyValueStore[K, commtypes.ValueTimestampG[VA]]
	initializer InitializerG[VA]
	aggregator  AggregatorG[K, V, VA]
	name        string
	BaseProcessorG[K, V, K, commtypes.ChangeG[VA]]
	useCache bool
}

var _ = ProcessorG[int, int, int, commtypes.ChangeG[int]](&StreamAggregateProcessorG[int, int, int]{})

func NewStreamAggregateProcessorG[K, V, VA any](
	name string, store store.CachedKeyValueStore[K, commtypes.ValueTimestampG[VA]],
	initializer InitializerG[VA], aggregator AggregatorG[K, V, VA],
	useCache bool,
) ProcessorG[K, V, K, commtypes.ChangeG[VA]] {
	p := &StreamAggregateProcessorG[K, V, VA]{
		initializer: initializer,
		aggregator:  aggregator,
		store:       store,
		name:        name,
		useCache:    useCache,
	}
	fmt.Fprintf(os.Stderr, "StreamAggregateProc useCache: %v\n", useCache)
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	if useCache {
		store.SetCacheFlushCallback(func(ctx context.Context, msg commtypes.MessageG[K, commtypes.ChangeG[commtypes.ValueTimestampG[VA]]]) error {
			// fmt.Fprintf(os.Stderr, "StreamAgg cache flush cb\n")
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
			msgForNext := commtypes.MessageG[K, commtypes.ChangeG[VA]]{
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

func (p *StreamAggregateProcessorG[K, V, VA]) Name() string {
	return p.name
}

func (p *StreamAggregateProcessorG[K, V, VA]) ProcessAndReturn(ctx context.Context,
	msg commtypes.MessageG[K, V],
) ([]commtypes.MessageG[K, commtypes.ChangeG[VA]], error) {
	if msg.Key.IsNone() || msg.Value.IsNone() {
		log.Warn().Msgf("skipping record due to null key or value. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}
	key := msg.Key.Unwrap()
	oldAggTs, ok, err := p.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var oldAgg optional.Option[VA]
	var newTs int64
	if ok {
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
	msgVal := msg.Value.Unwrap()
	newAgg := p.aggregator.Apply(key, msgVal, oldAgg)
	err = p.store.Put(ctx, key, commtypes.CreateValueTimestampGOptional(newAgg, newTs),
		store.TimeMeta{RecordTsMs: newTs, StartProcTs: msg.StartProcTime})
	if err != nil {
		return nil, err
	}
	change := commtypes.ChangeG[VA]{
		NewVal: newAgg,
		OldVal: oldAgg,
	}
	// debug.Fprintf(os.Stderr, "StreamAgg key %v, oldVal %v, newVal %v\n", msg.Key, oldAgg, newAgg)
	if p.useCache {
		return nil, nil
	} else {
		return []commtypes.MessageG[K, commtypes.ChangeG[VA]]{{
			Key: msg.Key, Value: optional.Some(change), TimestampMs: newTs,
			StartProcTime: msg.StartProcTime,
		}}, nil
	}
}
