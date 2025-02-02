package processor

/*
type FlatMapper interface {
	FlatMap(commtypes.Message) ([]commtypes.Message, error)
}

var _ = (FlatMapper)(FlatMapperFunc(nil))

type FlatMapperFunc func(commtypes.Message) ([]commtypes.Message, error)

func (fn FlatMapperFunc) FlatMap(msg commtypes.Message) ([]commtypes.Message, error) {
	return fn(msg)
}

type FlatMapProcessor struct {
	mapper FlatMapper
	name   string
}

func NewFlatMapProcessor(name string, mapper FlatMapper) *FlatMapProcessor {
	return &FlatMapProcessor{
		mapper: mapper,
	}
}

func (p *FlatMapProcessor) Name() string {
	return p.name
}

func (p *FlatMapProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	msgs, err := p.mapper.FlatMap(msg)
	if err != nil {
		return nil, err
	}
	for _, msg_out := range msgs {
		msg_out.Timestamp = msg.Timestamp
	}
	return msgs, nil
}
*/
