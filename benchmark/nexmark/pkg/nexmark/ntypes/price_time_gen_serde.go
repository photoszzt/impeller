package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PriceTimeJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s PriceTimeJSONSerde) String() string {
	return "PriceTimeJSONSerde"
}

var _ = fmt.Stringer(PriceTimeJSONSerde{})

var _ = commtypes.Serde(PriceTimeJSONSerde{})

func (s PriceTimeJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PriceTime)
	if !ok {
		vTmp := value.(PriceTime)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s PriceTimeJSONSerde) Decode(value []byte) (interface{}, error) {
	v := PriceTime{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type PriceTimeMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(PriceTimeMsgpSerde{})

func (s PriceTimeMsgpSerde) String() string {
	return "PriceTimeMsgpSerde"
}

var _ = fmt.Stringer(PriceTimeMsgpSerde{})

func (s PriceTimeMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PriceTime)
	if !ok {
		vTmp := value.(PriceTime)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s PriceTimeMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := PriceTime{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetPriceTimeSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return PriceTimeJSONSerde{}, nil
	case commtypes.MSGP:
		return PriceTimeMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
