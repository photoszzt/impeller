package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type CheckpointJSONSerde struct {
	DefaultJSONSerde
}

func (s CheckpointJSONSerde) String() string {
	return "CheckpointJSONSerde"
}

var _ = fmt.Stringer(CheckpointJSONSerde{})

var _ = Serde(CheckpointJSONSerde{})

func (s CheckpointJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*Checkpoint)
	if !ok {
		vTmp := value.(Checkpoint)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s CheckpointJSONSerde) Decode(value []byte) (interface{}, error) {
	v := Checkpoint{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type CheckpointMsgpSerde struct {
	DefaultMsgpSerde
}

var _ = Serde(CheckpointMsgpSerde{})

func (s CheckpointMsgpSerde) String() string {
	return "CheckpointMsgpSerde"
}

var _ = fmt.Stringer(CheckpointMsgpSerde{})

func (s CheckpointMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*Checkpoint)
	if !ok {
		vTmp := value.(Checkpoint)
		v = &vTmp
	}
	b := PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s CheckpointMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := Checkpoint{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetCheckpointSerde(serdeFormat SerdeFormat) (Serde, error) {
	switch serdeFormat {
	case JSON:
		return CheckpointJSONSerde{}, nil
	case MSGP:
		return CheckpointMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
