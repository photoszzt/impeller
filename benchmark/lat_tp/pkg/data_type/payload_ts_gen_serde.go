package datatype

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PayloadTsJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s PayloadTsJSONSerde) String() string {
	return "PayloadTsJSONSerde"
}

var _ = fmt.Stringer(PayloadTsJSONSerde{})

var _ = commtypes.Serde(PayloadTsJSONSerde{})

func (s PayloadTsJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PayloadTs)
	if !ok {
		vTmp := value.(PayloadTs)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s PayloadTsJSONSerde) Decode(value []byte) (interface{}, error) {
	v := PayloadTs{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type PayloadTsMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(PayloadTsMsgpSerde{})

func (s PayloadTsMsgpSerde) String() string {
	return "PayloadTsMsgpSerde"
}

var _ = fmt.Stringer(PayloadTsMsgpSerde{})

func (s PayloadTsMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PayloadTs)
	if !ok {
		vTmp := value.(PayloadTs)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s PayloadTsMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := PayloadTs{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetPayloadTsSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return PayloadTsJSONSerde{}, nil
	case commtypes.MSGP:
		return PayloadTsMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
