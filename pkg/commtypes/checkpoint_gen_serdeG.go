package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type CheckpointJSONSerdeG struct {
	DefaultJSONSerde
}

func (s CheckpointJSONSerdeG) String() string {
	return "CheckpointJSONSerdeG"
}

var _ = fmt.Stringer(CheckpointJSONSerdeG{})

var _ = SerdeG[Checkpoint](CheckpointJSONSerdeG{})

type CheckpointMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s CheckpointMsgpSerdeG) String() string {
	return "CheckpointMsgpSerdeG"
}

var _ = fmt.Stringer(CheckpointMsgpSerdeG{})

var _ = SerdeG[Checkpoint](CheckpointMsgpSerdeG{})

func (s CheckpointJSONSerdeG) Encode(value Checkpoint) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s CheckpointJSONSerdeG) Decode(value []byte) (Checkpoint, error) {
	v := Checkpoint{}
	if err := json.Unmarshal(value, &v); err != nil {
		return Checkpoint{}, err
	}
	return v, nil
}

func (s CheckpointMsgpSerdeG) Encode(value Checkpoint) ([]byte, *[]byte, error) {
	// b := PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s CheckpointMsgpSerdeG) Decode(value []byte) (Checkpoint, error) {
	v := Checkpoint{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return Checkpoint{}, err
	}
	return v, nil
}

func GetCheckpointSerdeG(serdeFormat SerdeFormat) (SerdeG[Checkpoint], error) {
	if serdeFormat == JSON {
		return CheckpointJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return CheckpointMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
