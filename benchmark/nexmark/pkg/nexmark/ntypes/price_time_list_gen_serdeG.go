package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type PriceTimeListJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s PriceTimeListJSONSerdeG) String() string {
	return "PriceTimeListJSONSerdeG"
}

var _ = fmt.Stringer(PriceTimeListJSONSerdeG{})

var _ = commtypes.SerdeG[PriceTimeList](PriceTimeListJSONSerdeG{})

type PriceTimeListMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s PriceTimeListMsgpSerdeG) String() string {
	return "PriceTimeListMsgpSerdeG"
}

var _ = fmt.Stringer(PriceTimeListMsgpSerdeG{})

var _ = commtypes.SerdeG[PriceTimeList](PriceTimeListMsgpSerdeG{})

func (s PriceTimeListJSONSerdeG) Encode(value PriceTimeList) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s PriceTimeListJSONSerdeG) Decode(value []byte) (PriceTimeList, error) {
	v := PriceTimeList{}
	if err := json.Unmarshal(value, &v); err != nil {
		return PriceTimeList{}, err
	}
	return v, nil
}

func (s PriceTimeListMsgpSerdeG) Encode(value PriceTimeList) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s PriceTimeListMsgpSerdeG) Decode(value []byte) (PriceTimeList, error) {
	v := PriceTimeList{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return PriceTimeList{}, err
	}
	return v, nil
}

func GetPriceTimeListSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[PriceTimeList], error) {
	if serdeFormat == commtypes.JSON {
		return PriceTimeListJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return PriceTimeListMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
