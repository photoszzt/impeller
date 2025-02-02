package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type NameCityStateIdJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s NameCityStateIdJSONSerdeG) String() string {
	return "NameCityStateIdJSONSerdeG"
}

var _ = fmt.Stringer(NameCityStateIdJSONSerdeG{})

var _ = commtypes.SerdeG[NameCityStateId](NameCityStateIdJSONSerdeG{})

type NameCityStateIdMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s NameCityStateIdMsgpSerdeG) String() string {
	return "NameCityStateIdMsgpSerdeG"
}

var _ = fmt.Stringer(NameCityStateIdMsgpSerdeG{})

var _ = commtypes.SerdeG[NameCityStateId](NameCityStateIdMsgpSerdeG{})

func (s NameCityStateIdJSONSerdeG) Encode(value NameCityStateId) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s NameCityStateIdJSONSerdeG) Decode(value []byte) (NameCityStateId, error) {
	v := NameCityStateId{}
	if err := json.Unmarshal(value, &v); err != nil {
		return NameCityStateId{}, err
	}
	return v, nil
}

func (s NameCityStateIdMsgpSerdeG) Encode(value NameCityStateId) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s NameCityStateIdMsgpSerdeG) Decode(value []byte) (NameCityStateId, error) {
	v := NameCityStateId{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return NameCityStateId{}, err
	}
	return v, nil
}

func GetNameCityStateIdSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[NameCityStateId], error) {
	if serdeFormat == commtypes.JSON {
		return NameCityStateIdJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return NameCityStateIdMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
