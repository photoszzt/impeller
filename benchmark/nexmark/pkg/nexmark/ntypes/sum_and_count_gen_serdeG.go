package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type SumAndCountJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s SumAndCountJSONSerdeG) String() string {
	return "SumAndCountJSONSerdeG"
}

var _ = fmt.Stringer(SumAndCountJSONSerdeG{})

var _ = commtypes.SerdeG[SumAndCount](SumAndCountJSONSerdeG{})

type SumAndCountMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s SumAndCountMsgpSerdeG) String() string {
	return "SumAndCountMsgpSerdeG"
}

var _ = fmt.Stringer(SumAndCountMsgpSerdeG{})

var _ = commtypes.SerdeG[SumAndCount](SumAndCountMsgpSerdeG{})

func (s SumAndCountJSONSerdeG) Encode(value SumAndCount) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s SumAndCountJSONSerdeG) Decode(value []byte) (SumAndCount, error) {
	v := SumAndCount{}
	if err := json.Unmarshal(value, &v); err != nil {
		return SumAndCount{}, err
	}
	return v, nil
}

func (s SumAndCountMsgpSerdeG) Encode(value SumAndCount) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s SumAndCountMsgpSerdeG) Decode(value []byte) (SumAndCount, error) {
	v := SumAndCount{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return SumAndCount{}, err
	}
	return v, nil
}

func GetSumAndCountSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[SumAndCount], error) {
	if serdeFormat == commtypes.JSON {
		return SumAndCountJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return SumAndCountMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
