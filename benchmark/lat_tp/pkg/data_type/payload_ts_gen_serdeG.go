package datatype

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PayloadTsJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s PayloadTsJSONSerdeG) String() string {
	return "PayloadTsJSONSerdeG"
}

var _ = fmt.Stringer(PayloadTsJSONSerdeG{})

var _ = commtypes.SerdeG[PayloadTs](PayloadTsJSONSerdeG{})

type PayloadTsMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s PayloadTsMsgpSerdeG) String() string {
	return "PayloadTsMsgpSerdeG"
}

var _ = fmt.Stringer(PayloadTsMsgpSerdeG{})

var _ = commtypes.SerdeG[PayloadTs](PayloadTsMsgpSerdeG{})

func (s PayloadTsJSONSerdeG) Encode(value PayloadTs) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s PayloadTsJSONSerdeG) Decode(value []byte) (PayloadTs, error) {
	v := PayloadTs{}
	if err := json.Unmarshal(value, &v); err != nil {
		return PayloadTs{}, err
	}
	return v, nil
}

func (s PayloadTsMsgpSerdeG) Encode(value PayloadTs) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s PayloadTsMsgpSerdeG) Decode(value []byte) (PayloadTs, error) {
	v := PayloadTs{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return PayloadTs{}, err
	}
	return v, nil
}

func GetPayloadTsSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[PayloadTs], error) {
	if serdeFormat == commtypes.JSON {
		return PayloadTsJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return PayloadTsMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
