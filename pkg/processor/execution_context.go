package processor

import (
	"sharedlog-stream/pkg/proc_interface"
	"sharedlog-stream/pkg/producer_consumer"
)

type ExecutionContext interface {
	proc_interface.ProcArgs
	proc_interface.ProducersConsumers
}

type BaseExecutionContext struct {
	proc_interface.BaseConsumersProducers
	proc_interface.BaseProcArgs
}

func NewExecutionContext(
	consumers []*producer_consumer.MeteredConsumer,
	producers []producer_consumer.MeteredProducerIntr,
	funcName string,
	curEpoch uint16,
	parNum uint8,
) BaseExecutionContext {
	return BaseExecutionContext{
		BaseProcArgs:           proc_interface.NewBaseProcArgs(funcName, curEpoch, parNum),
		BaseConsumersProducers: proc_interface.NewBaseSrcsSinks(consumers, producers),
	}
}

func NewExecutionContextFromComponents(
	consumersProducers proc_interface.BaseConsumersProducers,
	procArgs proc_interface.BaseProcArgs,
) BaseExecutionContext {
	return BaseExecutionContext{
		BaseProcArgs:           procArgs,
		BaseConsumersProducers: consumersProducers,
	}
}
