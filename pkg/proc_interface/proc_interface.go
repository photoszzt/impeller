package proc_interface

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/exactly_once_intr"
	"sharedlog-stream/pkg/producer_consumer"
)

type ProducersConsumers interface {
	Producers() []producer_consumer.MeteredProducerIntr
	Consumers() []*producer_consumer.MeteredConsumer
	StartWarmup()
	AllConsumersIsInit() bool
}

type BaseConsumersProducers struct {
	consumers         []*producer_consumer.MeteredConsumer
	producers         []producer_consumer.MeteredProducerIntr
	allConsumerIsInit bool
}

func NewBaseSrcsSinks(srcs []*producer_consumer.MeteredConsumer,
	sinks []producer_consumer.MeteredProducerIntr,
) BaseConsumersProducers {
	allConsumerIsInit := true
	for _, c := range srcs {
		if !c.IsInitialSource() {
			allConsumerIsInit = false
			break
		}
	}
	return BaseConsumersProducers{
		consumers:         srcs,
		producers:         sinks,
		allConsumerIsInit: allConsumerIsInit,
	}
}

func (pa *BaseConsumersProducers) Consumers() []*producer_consumer.MeteredConsumer {
	return pa.consumers
}

func (pa *BaseConsumersProducers) Producers() []producer_consumer.MeteredProducerIntr {
	return pa.producers
}

func (pa *BaseConsumersProducers) StartWarmup() {
	for _, src := range pa.consumers {
		src.StartWarmup()
	}
	for _, sink := range pa.producers {
		sink.StartWarmup()
	}
}

func (pa *BaseConsumersProducers) AllConsumersIsInit() bool {
	return pa.allConsumerIsInit
}

type ProcArgs interface {
	SubstreamNum() uint8
	CurEpoch() uint16
	FuncName() string
	RecordFinishFunc() exactly_once_intr.RecordPrevInstanceFinishFunc
	SetRecordFinishFunc(recordFinishFunc exactly_once_intr.RecordPrevInstanceFinishFunc)
	TrackParFunc() exactly_once_intr.TrackProdSubStreamFunc
	SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc)
}

type BaseProcArgs struct {
	recordFinishFunc exactly_once_intr.RecordPrevInstanceFinishFunc
	trackParFunc     exactly_once_intr.TrackProdSubStreamFunc
	funcName         string
	curEpoch         uint16
	parNum           uint8
}

func NewBaseProcArgs(funcName string, curEpoch uint16, parNum uint8) BaseProcArgs {
	return BaseProcArgs{
		recordFinishFunc: exactly_once_intr.DefaultRecordPrevInstanceFinishFunc,
		trackParFunc:     exactly_once_intr.DefaultTrackProdSubstreamFunc,
		funcName:         funcName,
		curEpoch:         curEpoch,
		parNum:           parNum,
	}
}

func (pa *BaseProcArgs) SubstreamNum() uint8 {
	return pa.parNum
}

func (pa *BaseProcArgs) FuncName() string {
	return pa.funcName
}

func (pa *BaseProcArgs) CurEpoch() uint16 {
	return pa.curEpoch
}

func (pa *BaseProcArgs) RecordFinishFunc() exactly_once_intr.RecordPrevInstanceFinishFunc {
	return pa.recordFinishFunc
}

func (pa *BaseProcArgs) SetRecordFinishFunc(recordFinishFunc exactly_once_intr.RecordPrevInstanceFinishFunc) {
	pa.recordFinishFunc = recordFinishFunc
}

func (pa *BaseProcArgs) TrackParFunc() exactly_once_intr.TrackProdSubStreamFunc {
	return pa.trackParFunc
}

func (pa *BaseProcArgs) SetTrackParFunc(trackParFunc exactly_once_intr.TrackProdSubStreamFunc) {
	pa.trackParFunc = trackParFunc
}

type ProcessMsgFunc[K, V any] func(ctx context.Context, msg commtypes.MessageG[K, V], argsTmp interface{}) error

type ProcessAndReturnFunc[KI, VI, KO, VO any] func(ctx context.Context, msg commtypes.MessageG[KI, VI]) ([]commtypes.MessageG[KO, VO], error)
