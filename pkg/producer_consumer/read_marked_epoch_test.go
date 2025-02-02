package producer_consumer

import (
	"sharedlog-stream/pkg/commtypes"
	"testing"
)

func TestRead(t *testing.T) {
	ProdId1 :=
		commtypes.ProducerId{
			TaskId:    1,
			TaskEpoch: 1,
		}
	ProdId2 :=
		commtypes.ProducerId{
			TaskId:    2,
			TaskEpoch: 1,
		}
	_ = []*commtypes.RawMsg{
		{
			Payload:      []byte{0, 1},
			IsControl:    false,
			LogSeqNum:    1,
			MsgSeqNum:    1,
			IsPayloadArr: false,
			ProdId:       ProdId1,
		},
		{
			Payload:      []byte{0, 1},
			IsControl:    false,
			LogSeqNum:    2,
			MsgSeqNum:    1,
			IsPayloadArr: false,
			ProdId:       ProdId2,
		},
		{
			Payload:      []byte{0, 1},
			IsControl:    false,
			LogSeqNum:    3,
			MsgSeqNum:    2,
			IsPayloadArr: false,
			ProdId:       ProdId1,
		},
		{
			Payload:      []byte{0, 1},
			IsControl:    true,
			LogSeqNum:    3,
			MsgSeqNum:    3,
			IsPayloadArr: false,
			ProdId:       ProdId1,
		},
	}
}
