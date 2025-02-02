package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/store"

	"github.com/rs/zerolog/log"
)

/*
type TableSourceProcessor struct {
	store store.CoreKeyValueStore
	name  string
	BaseProcessor
}

var _ = Processor(&TableSourceProcessor{})

func NewTableSourceProcessor() *TableSourceProcessor {
	p := &TableSourceProcessor{
		name:          "toTable",
		BaseProcessor: BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func NewTableSourceProcessorWithTable(tab store.CoreKeyValueStore) *TableSourceProcessor {
	p := &TableSourceProcessor{
		name:          "toTable",
		store:         tab,
		BaseProcessor: BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *TableSourceProcessor) Name() string {
	return p.name
}

func (p *TableSourceProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Key == nil {
		log.Warn().Msgf("Skipping record due to null key")
		return nil, nil
	}
	if p.store != nil {
		var oldVal interface{}
		v, ok, err := p.store.Get(ctx, msg.Key)
		if err != nil {
			return nil, err
		}
		if ok {
			oldValTs := commtypes.CastToValTsPtr(v)
			if oldValTs != nil {
				oldVal = oldValTs.Value
				if msg.Timestamp < oldValTs.Timestamp {
					log.Warn().Msgf("Detected out-of-order table update for %s, old Ts=[%v] new Ts=[%v].",
						p.store.Name(), oldValTs.Timestamp, msg.Timestamp)
				}
			}
		} else {
			oldVal = nil
		}
		err = p.store.Put(ctx, msg.Key, commtypes.CreateValueTimestamp(msg.Value, msg.Timestamp))
		if err != nil {
			return nil, err
		}
		return []commtypes.Message{{Key: msg.Key, Value: commtypes.Change{NewVal: msg.Value, OldVal: oldVal}, Timestamp: msg.Timestamp}}, nil
	} else {
		return []commtypes.Message{{Key: msg.Key, Value: commtypes.Change{NewVal: msg.Value, OldVal: nil}, Timestamp: msg.Timestamp}}, nil
	}
}
*/

func MsgSerdeWithValueTs(serdeFormat commtypes.SerdeFormat, keySerde commtypes.Serde, valSerde commtypes.Serde) (commtypes.MessageSerde, error) {
	valueTsSerde, err := commtypes.GetValueTsSerde(serdeFormat, valSerde)
	if err != nil {
		return nil, err
	}
	return commtypes.GetMsgSerde(serdeFormat, keySerde, valueTsSerde)
}

func MsgSerdeWithValueTsG[K, V any](serdeFormat commtypes.SerdeFormat,
	keySerde commtypes.SerdeG[K], valSerde commtypes.SerdeG[V],
) (commtypes.MessageGSerdeG[K, commtypes.ValueTimestampG[V]], error) {
	valueTsSerde, err := commtypes.GetValueTsGSerdeG(serdeFormat, valSerde)
	if err != nil {
		return nil, err
	}
	return commtypes.GetMsgGSerdeG(serdeFormat, keySerde, valueTsSerde)
}

/*
func ToInMemKVTable(storeName string, compare store.KVStoreLessFunc) (
	*MeteredProcessor, store.CoreKeyValueStore,
) {
	s := store.NewInMemoryKeyValueStore(storeName, compare)
	toTableProc := NewMeteredProcessor(NewTableSourceProcessorWithTable(s))
	return toTableProc, s
}
*/

type TableSourceProcessorG[K, V any] struct {
	store store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[V]]
	name  string
	BaseProcessorG[K, V, K, commtypes.ChangeG[V]]
	observedStreamTime int64
}

var _ = ProcessorG[int, int, int, commtypes.ChangeG[int]](&TableSourceProcessorG[int, int]{})

func NewTableSourceProcessorG[K, V any]() *TableSourceProcessorG[K, V] {
	p := &TableSourceProcessorG[K, V]{
		name:           "toTable",
		BaseProcessorG: BaseProcessorG[K, V, K, commtypes.ChangeG[V]]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func NewTableSourceProcessorWithTableG[K, V any](tab store.CoreKeyValueStoreG[K, commtypes.ValueTimestampG[V]]) *TableSourceProcessorG[K, V] {
	p := &TableSourceProcessorG[K, V]{
		name:           "toTable",
		store:          tab,
		BaseProcessorG: BaseProcessorG[K, V, K, commtypes.ChangeG[V]]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *TableSourceProcessorG[K, V]) Name() string {
	return p.name
}

func (p *TableSourceProcessorG[K, V]) ProcessAndReturn(ctx context.Context, msg commtypes.MessageG[K, V]) (
	[]commtypes.MessageG[K, commtypes.ChangeG[V]], error,
) {
	if msg.Key.IsNone() {
		log.Warn().Msgf("Skipping record due to null key")
		return nil, nil
	}
	if msg.TimestampMs > p.observedStreamTime {
		p.observedStreamTime = msg.TimestampMs
	}
	if p.store != nil {
		oldVal := optional.None[V]()
		key := msg.Key.Unwrap()
		oldValTs, ok, err := p.store.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if ok {
			oldVal = optional.Some(oldValTs.Value)
			if msg.TimestampMs < oldValTs.Timestamp {
				log.Warn().Msgf("Detected out-of-order table update for %s, old Ts=[%v] new Ts=[%v].",
					p.store.Name(), oldValTs.Timestamp, msg.TimestampMs)
			}
		}
		err = p.store.Put(ctx, key, commtypes.CreateValueTimestampGOptional(msg.Value, msg.TimestampMs),
			store.TimeMeta{RecordTsMs: p.observedStreamTime, StartProcTs: msg.StartProcTime})
		if err != nil {
			return nil, err
		}
		return []commtypes.MessageG[K, commtypes.ChangeG[V]]{{
			Key:           msg.Key,
			Value:         optional.Some(commtypes.ChangeG[V]{NewVal: msg.Value, OldVal: oldVal}),
			TimestampMs:   msg.TimestampMs,
			StartProcTime: msg.StartProcTime}}, nil
	} else {
		return []commtypes.MessageG[K, commtypes.ChangeG[V]]{{
			Key:           msg.Key,
			Value:         optional.Some(commtypes.ChangeG[V]{NewVal: msg.Value, OldVal: optional.None[V]()}),
			TimestampMs:   msg.TimestampMs,
			StartProcTime: msg.StartProcTime}}, nil
	}
}
