package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PersonTimeJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s PersonTimeJSONSerdeG) String() string {
	return "PersonTimeJSONSerdeG"
}

var _ = fmt.Stringer(PersonTimeJSONSerdeG{})

var _ = commtypes.SerdeG[PersonTime](PersonTimeJSONSerdeG{})

type PersonTimeMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s PersonTimeMsgpSerdeG) String() string {
	return "PersonTimeMsgpSerdeG"
}

var _ = fmt.Stringer(PersonTimeMsgpSerdeG{})

var _ = commtypes.SerdeG[PersonTime](PersonTimeMsgpSerdeG{})

func (s PersonTimeJSONSerdeG) Encode(value PersonTime) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s PersonTimeJSONSerdeG) Decode(value []byte) (PersonTime, error) {
	v := PersonTime{}
	if err := json.Unmarshal(value, &v); err != nil {
		return PersonTime{}, err
	}
	return v, nil
}

func (s PersonTimeMsgpSerdeG) Encode(value PersonTime) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s PersonTimeMsgpSerdeG) Decode(value []byte) (PersonTime, error) {
	v := PersonTime{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return PersonTime{}, err
	}
	return v, nil
}

func GetPersonTimeSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[PersonTime], error) {
	if serdeFormat == commtypes.JSON {
		return PersonTimeJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return PersonTimeMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
