package processor

/*
type PrintProcessor struct {
	BaseProcessor
}

func NewPrintProcessor() Processor {
	p := &PrintProcessor{
		BaseProcessor: BaseProcessor{},
	}
	p.BaseProcessor.ProcessingFunc = p.ProcessAndReturn
	return p
}

var _ Processor = &PrintProcessor{}

func (p *PrintProcessor) Name() string {
	return "PrintProcessor"
}

func (p *PrintProcessor) ProcessAndReturn(ctx context.Context, msg commtypes.Message) ([]commtypes.Message, error) {
	fmt.Fprintf(os.Stderr, "msg: %v\n", msg)
	return []commtypes.Message{msg}, nil
}
*/
