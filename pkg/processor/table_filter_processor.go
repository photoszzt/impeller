package processor

/*
type TableFilterProcessor struct {
	pred  Predicate
	store store.CoreKeyValueStore
	name  string
	BaseProcessor
	filterNot bool
}

var _ = Processor(&TableFilterProcessor{})

func NewTableFilterProcessor(name string, pred Predicate, filterNot bool) *TableFilterProcessor {
	p := &TableFilterProcessor{
		pred:      pred,
		filterNot: filterNot,
		store:     nil,
		name:      name,
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

func NewTableFilterProcessorQuriable(name string, pred Predicate, filterNot bool, store store.CoreKeyValueStore) *TableFilterProcessor {
	return &TableFilterProcessor{
		pred:      pred,
		filterNot: filterNot,
		store:     store,
		name:      name,
	}
}

func (p *TableFilterProcessor) Name() string {
	return p.name
}

func (p *TableFilterProcessor) computeVal(key interface{}, value interface{}) (interface{}, error) {
	var newVal interface{}
	if value != nil {
		ok, err := p.pred.Assert(key, value)
		if err != nil {
			return nil, err
		}
		if p.filterNot != ok {
			newVal = value
		}
	}
	return newVal, nil
}

func (p *TableFilterProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	change, ok := msg.Value.(*commtypes.Change)
	if !ok {
		changeTmp := msg.Value.(commtypes.Change)
		change = &changeTmp
	}
	oldVal, err := p.computeVal(msg.Key, change.OldVal)
	if err != nil {
		return nil, err
	}
	newVal, err := p.computeVal(msg.Key, change.NewVal)
	if err != nil {
		return nil, err
	}
	if oldVal == nil && newVal == nil {
		return nil, nil
	}
	if p.store != nil {
		err = p.store.Put(ctx, msg.Key, commtypes.CreateValueTimestamp(newVal, msg.Timestamp))
		if err != nil {
			return nil, err
		}
	}
	return []commtypes.Message{{
		Key:       msg.Key,
		Value:     commtypes.Change{NewVal: newVal, OldVal: oldVal},
		Timestamp: msg.Timestamp,
	}}, nil
}
*/
