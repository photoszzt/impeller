package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PriceTimeListJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s PriceTimeListJSONSerde) String() string {
	return "PriceTimeListJSONSerde"
}

var _ = fmt.Stringer(PriceTimeListJSONSerde{})

var _ = commtypes.Serde(PriceTimeListJSONSerde{})

func (s PriceTimeListJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PriceTimeList)
	if !ok {
		vTmp := value.(PriceTimeList)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s PriceTimeListJSONSerde) Decode(value []byte) (interface{}, error) {
	v := PriceTimeList{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type PriceTimeListMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(PriceTimeListMsgpSerde{})

func (s PriceTimeListMsgpSerde) String() string {
	return "PriceTimeListMsgpSerde"
}

var _ = fmt.Stringer(PriceTimeListMsgpSerde{})

func (s PriceTimeListMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PriceTimeList)
	if !ok {
		vTmp := value.(PriceTimeList)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s PriceTimeListMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := PriceTimeList{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetPriceTimeListSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return PriceTimeListJSONSerde{}, nil
	case commtypes.MSGP:
		return PriceTimeListMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
