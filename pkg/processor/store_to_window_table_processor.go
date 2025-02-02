package processor

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/store"
)

/*
type StoreToWindowTableProcessor struct {
	store store.CoreWindowStore
	name  string
	BaseProcessor
	observedTs int64
}

var _ = Processor(&StoreToWindowTableProcessor{})

func NewStoreToWindowTableProcessor(store store.CoreWindowStore) *StoreToWindowTableProcessor {
	p := &StoreToWindowTableProcessor{
		store:         store,
		name:          "StoreTo" + store.Name(),
		BaseProcessor: BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func (p *StoreToWindowTableProcessor) Name() string {
	return p.name
}

func (p *StoreToWindowTableProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	if msg.Timestamp > p.observedTs {
		p.observedTs = msg.Timestamp
	}
	if !utils.IsNil(msg.Key) {
		err := p.store.Put(ctx, msg.Key, msg.Value, p.observedTs)
		if err != nil {
			return nil, err
		}
	}
	return []commtypes.Message{msg}, nil
}
*/

type StoreToWindowTableProcessorG[K, V any] struct {
	store store.CoreWindowStoreG[K, V]
	name  string
	BaseProcessorG[K, V, K, V]
	observedTs int64
}

var _ = ProcessorG[int, int, int, int](&StoreToWindowTableProcessorG[int, int]{})

func NewStoreToWindowTableProcessorG[K, V any](store store.CoreWindowStoreG[K, V]) *StoreToWindowTableProcessorG[K, V] {
	p := &StoreToWindowTableProcessorG[K, V]{
		store:          store,
		name:           "StoreTo" + store.Name(),
		BaseProcessorG: BaseProcessorG[K, V, K, V]{},
	}
	p.BaseProcessorG.ProcessingFuncG = p.ProcessAndReturn
	return p
}

func (p *StoreToWindowTableProcessorG[K, V]) Name() string { return p.name }
func (p *StoreToWindowTableProcessorG[K, V]) ProcessAndReturn(ctx context.Context,
	msg commtypes.MessageG[K, V],
) ([]commtypes.MessageG[K, V], error) {
	if msg.TimestampMs > p.observedTs {
		p.observedTs = msg.TimestampMs
	}
	key, ok := msg.Key.Take()
	if ok {
		err := p.store.Put(ctx, key, msg.Value, p.observedTs,
			store.TimeMeta{RecordTsMs: p.observedTs, StartProcTs: msg.StartProcTime})
		if err != nil {
			return nil, err
		}
	}
	return []commtypes.MessageG[K, V]{msg}, nil
}

// // for test
// func ToInMemWindowTable(
// 	storeName string,
// 	joinWindow *commtypes.JoinWindows,
// 	compare store.CompareFunc,
// ) (*MeteredProcessor, store.CoreWindowStore, error) {
// 	store := store.NewInMemoryWindowStore(
// 		storeName,
// 		joinWindow.MaxSize()+joinWindow.GracePeriodMs(),
// 		joinWindow.MaxSize(), true, compare)
// 	toTableProc := NewMeteredProcessor(NewStoreToWindowTableProcessor(store))
// 	return toTableProc, store, nil
// }

func ToInMemSkipMapWindowTable[K, V any](
	storeName string,
	joinWindow *commtypes.JoinWindows,
	compare store.CompareFuncG[K],
) (*MeteredProcessorG[K, V, K, V], store.CoreWindowStoreG[K, V], error) {
	store := store.NewInMemorySkipMapWindowStore[K, V](
		storeName,
		joinWindow.MaxSize()+joinWindow.GracePeriodMs(),
		joinWindow.MaxSize(), true, compare)
	toTableProc := NewMeteredProcessorG[K, V, K, V](NewStoreToWindowTableProcessorG[K, V](store))
	return toTableProc, store, nil
}
