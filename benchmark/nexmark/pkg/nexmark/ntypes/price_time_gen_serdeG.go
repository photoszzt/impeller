package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PriceTimeJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s PriceTimeJSONSerdeG) String() string {
	return "PriceTimeJSONSerdeG"
}

var _ = fmt.Stringer(PriceTimeJSONSerdeG{})

var _ = commtypes.SerdeG[PriceTime](PriceTimeJSONSerdeG{})

type PriceTimeMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s PriceTimeMsgpSerdeG) String() string {
	return "PriceTimeMsgpSerdeG"
}

var _ = fmt.Stringer(PriceTimeMsgpSerdeG{})

var _ = commtypes.SerdeG[PriceTime](PriceTimeMsgpSerdeG{})

func (s PriceTimeJSONSerdeG) Encode(value PriceTime) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s PriceTimeJSONSerdeG) Decode(value []byte) (PriceTime, error) {
	v := PriceTime{}
	if err := json.Unmarshal(value, &v); err != nil {
		return PriceTime{}, err
	}
	return v, nil
}

func (s PriceTimeMsgpSerdeG) Encode(value PriceTime) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s PriceTimeMsgpSerdeG) Decode(value []byte) (PriceTime, error) {
	v := PriceTime{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return PriceTime{}, err
	}
	return v, nil
}

func GetPriceTimeSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[PriceTime], error) {
	if serdeFormat == commtypes.JSON {
		return PriceTimeJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return PriceTimeMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
