package commtypes

import (
	"fmt"

	"cs.utexas.edu/zjia/faas/types"
)

var EmptyProducerId = ProducerId{TaskEpoch: 0, TaskId: 0}

type ProducerId struct {
	TaskId    uint64
	TaskEpoch uint32
}

var (
	_ = fmt.Stringer(&ProducerId{})
	_ = fmt.GoStringer(&ProducerId{})
)

func (p *ProducerId) String() string {
	return fmt.Sprintf("ProducerId: {TaskId: %#x, TaskEpoch: %#x}",
		p.TaskId, p.TaskEpoch)
}

func (p *ProducerId) GoString() string {
	return fmt.Sprintf("ProducerId: {TaskId: %#x, TaskEpoch: %#x}",
		p.TaskId, p.TaskEpoch)
}

func NewProducerId() ProducerId {
	return ProducerId{
		TaskId:    0,
		TaskEpoch: 0,
	}
}

func (tg *ProducerId) GetCurrentEpoch() uint32 {
	return tg.TaskEpoch
}

func (tg *ProducerId) GetCurrentTaskId() uint64 {
	return tg.TaskId
}

func (tg *ProducerId) GetProducerId() ProducerId {
	return ProducerId{
		TaskId:    tg.TaskId,
		TaskEpoch: tg.TaskEpoch,
	}
}

func (tg *ProducerId) InitTaskId(env types.Environment) {
	for tg.TaskId == 0 {
		tg.TaskId = env.GenerateUniqueID()
	}
}

func EqualProdId(a, b *ProdId) bool {
	if a != nil && b != nil {
		return a.TaskEpoch == b.TaskEpoch && a.TaskId == b.TaskId
	} else {
		return a == nil && b == nil
	}
}
