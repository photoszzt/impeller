package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type PayloadArrJSONSerde struct {
	DefaultJSONSerde
}

func (s PayloadArrJSONSerde) String() string {
	return "PayloadArrJSONSerde"
}

var _ = fmt.Stringer(PayloadArrJSONSerde{})

var _ = Serde(PayloadArrJSONSerde{})

func (s PayloadArrJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PayloadArr)
	if !ok {
		vTmp := value.(PayloadArr)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s PayloadArrJSONSerde) Decode(value []byte) (interface{}, error) {
	v := PayloadArr{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type PayloadArrMsgpSerde struct {
	DefaultMsgpSerde
}

var _ = Serde(PayloadArrMsgpSerde{})

func (s PayloadArrMsgpSerde) String() string {
	return "PayloadArrMsgpSerde"
}

var _ = fmt.Stringer(PayloadArrMsgpSerde{})

func (s PayloadArrMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*PayloadArr)
	if !ok {
		vTmp := value.(PayloadArr)
		v = &vTmp
	}
	b := PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s PayloadArrMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := PayloadArr{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetPayloadArrSerde(serdeFormat SerdeFormat) (Serde, error) {
	switch serdeFormat {
	case JSON:
		return PayloadArrJSONSerde{}, nil
	case MSGP:
		return PayloadArrMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
