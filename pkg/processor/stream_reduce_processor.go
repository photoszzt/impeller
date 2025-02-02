package processor

/*
type StreamReduceProcessor struct {
	store   store.CoreKeyValueStore
	reducer Reducer
	name    string
	BaseProcessor
}

var _ = Processor(&StreamReduceProcessor{})

func NewStreamReduceProcessor(name string, reducer Reducer, store store.CoreKeyValueStore) *StreamReduceProcessor {
	p := &StreamReduceProcessor{
		reducer: reducer,
		store:   store,
		name:    name,
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StreamReduceProcessor) Name() string {
	return p.name
}

func (p *StreamReduceProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Key == nil || msg.Value == nil {
		log.Warn().Msgf("skipping record due to null key or value. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}
	val, ok, err := p.store.Get(ctx, msg.Key)
	if err != nil {
		return nil, err
	}
	var newAgg interface{}
	var newTs int64
	if ok {
		oldAggTs := val.(*commtypes.ValueTimestamp)
		newAgg = p.reducer.Apply(oldAggTs.Value, msg.Value)
		if msg.Timestamp > oldAggTs.Timestamp {
			newTs = msg.Timestamp
		} else {
			newTs = oldAggTs.Timestamp
		}
	} else {
		newAgg = msg.Value
		newTs = msg.Timestamp
	}
	err = p.store.Put(ctx, msg.Key, commtypes.CreateValueTimestamp(newAgg, newTs))
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{{Key: msg.Key, Value: newAgg, Timestamp: newTs}}, nil
}

type StreamReduceProcessorG[K, V any] struct {
	store   store.CoreKeyValueStoreG[K, commtypes.ValueTimestamp]
	reducer ReducerG[V]
	name    string
}

var _ = Processor(&StreamReduceProcessor{})

func NewStreamReduceProcessorG[K, V any](name string, reducer ReducerG[V],
	store store.CoreKeyValueStoreG[K, commtypes.ValueTimestamp]) *StreamReduceProcessorG[K, V] {
	return &StreamReduceProcessorG[K, V]{
		reducer: reducer,
		store:   store,
		name:    name,
	}
}

func (p *StreamReduceProcessorG[K, V]) Name() string {
	return p.name
}

func (p *StreamReduceProcessorG[K, V]) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Key == nil || msg.Value == nil {
		log.Warn().Msgf("skipping record due to null key or value. key=%v, val=%v", msg.Key, msg.Value)
		return nil, nil
	}
	key := msg.Key.(K)
	oldAggTs, ok, err := p.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var newAgg interface{}
	var newTs int64
	if ok {
		newAgg = p.reducer.Apply(oldAggTs.Value.(V), msg.Value.(V))
		if msg.Timestamp > oldAggTs.Timestamp {
			newTs = msg.Timestamp
		} else {
			newTs = oldAggTs.Timestamp
		}
	} else {
		newAgg = msg.Value
		newTs = msg.Timestamp
	}
	err = p.store.Put(ctx, key, commtypes.CreateValueTimestampOptionalWithIntrVal(newAgg, newTs), newTs)
	if err != nil {
		return nil, err
	}
	return []commtypes.Message{{Key: msg.Key, Value: newAgg, Timestamp: newTs}}, nil
}

*/
