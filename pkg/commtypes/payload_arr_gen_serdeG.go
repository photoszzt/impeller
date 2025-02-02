package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type PayloadArrJSONSerdeG struct {
	DefaultJSONSerde
}

func (s PayloadArrJSONSerdeG) String() string {
	return "PayloadArrJSONSerdeG"
}

var _ = fmt.Stringer(PayloadArrJSONSerdeG{})

var _ = SerdeG[PayloadArr](PayloadArrJSONSerdeG{})

type PayloadArrMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s PayloadArrMsgpSerdeG) String() string {
	return "PayloadArrMsgpSerdeG"
}

var _ = fmt.Stringer(PayloadArrMsgpSerdeG{})

var _ = SerdeG[PayloadArr](PayloadArrMsgpSerdeG{})

func (s PayloadArrJSONSerdeG) Encode(value PayloadArr) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s PayloadArrJSONSerdeG) Decode(value []byte) (PayloadArr, error) {
	v := PayloadArr{}
	if err := json.Unmarshal(value, &v); err != nil {
		return PayloadArr{}, err
	}
	return v, nil
}

func (s PayloadArrMsgpSerdeG) Encode(value PayloadArr) ([]byte, *[]byte, error) {
	b := PopBuffer(value.Msgsize())
	buf := *b
	r, err := value.MarshalMsg(buf[:0])
	// r, err := value.MarshalMsg(nil)
	return r, b, err
}

func (s PayloadArrMsgpSerdeG) Decode(value []byte) (PayloadArr, error) {
	v := PayloadArr{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return PayloadArr{}, err
	}
	return v, nil
}

func GetPayloadArrSerdeG(serdeFormat SerdeFormat) (SerdeG[PayloadArr], error) {
	if serdeFormat == JSON {
		return PayloadArrJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return PayloadArrMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
