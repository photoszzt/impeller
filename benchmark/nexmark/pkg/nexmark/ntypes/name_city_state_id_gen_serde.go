package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type NameCityStateIdJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s NameCityStateIdJSONSerde) String() string {
	return "NameCityStateIdJSONSerde"
}

var _ = fmt.Stringer(NameCityStateIdJSONSerde{})

var _ = commtypes.Serde(NameCityStateIdJSONSerde{})

func (s NameCityStateIdJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*NameCityStateId)
	if !ok {
		vTmp := value.(NameCityStateId)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s NameCityStateIdJSONSerde) Decode(value []byte) (interface{}, error) {
	v := NameCityStateId{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type NameCityStateIdMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(NameCityStateIdMsgpSerde{})

func (s NameCityStateIdMsgpSerde) String() string {
	return "NameCityStateIdMsgpSerde"
}

var _ = fmt.Stringer(NameCityStateIdMsgpSerde{})

func (s NameCityStateIdMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*NameCityStateId)
	if !ok {
		vTmp := value.(NameCityStateId)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s NameCityStateIdMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := NameCityStateId{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetNameCityStateIdSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return NameCityStateIdJSONSerde{}, nil
	case commtypes.MSGP:
		return NameCityStateIdMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
