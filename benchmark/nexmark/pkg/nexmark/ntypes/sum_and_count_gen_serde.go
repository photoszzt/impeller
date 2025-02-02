package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type SumAndCountJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s SumAndCountJSONSerde) String() string {
	return "SumAndCountJSONSerde"
}

var _ = fmt.Stringer(SumAndCountJSONSerde{})

var _ = commtypes.Serde(SumAndCountJSONSerde{})

func (s SumAndCountJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*SumAndCount)
	if !ok {
		vTmp := value.(SumAndCount)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s SumAndCountJSONSerde) Decode(value []byte) (interface{}, error) {
	v := SumAndCount{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type SumAndCountMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(SumAndCountMsgpSerde{})

func (s SumAndCountMsgpSerde) String() string {
	return "SumAndCountMsgpSerde"
}

var _ = fmt.Stringer(SumAndCountMsgpSerde{})

func (s SumAndCountMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*SumAndCount)
	if !ok {
		vTmp := value.(SumAndCount)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s SumAndCountMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := SumAndCount{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetSumAndCountSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return SumAndCountJSONSerde{}, nil
	case commtypes.MSGP:
		return SumAndCountMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
