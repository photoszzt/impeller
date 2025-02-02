package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type BidAndMaxJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s BidAndMaxJSONSerde) String() string {
	return "BidAndMaxJSONSerde"
}

var _ = fmt.Stringer(BidAndMaxJSONSerde{})

var _ = commtypes.Serde(BidAndMaxJSONSerde{})

func (s BidAndMaxJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*BidAndMax)
	if !ok {
		vTmp := value.(BidAndMax)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s BidAndMaxJSONSerde) Decode(value []byte) (interface{}, error) {
	v := BidAndMax{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type BidAndMaxMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(BidAndMaxMsgpSerde{})

func (s BidAndMaxMsgpSerde) String() string {
	return "BidAndMaxMsgpSerde"
}

var _ = fmt.Stringer(BidAndMaxMsgpSerde{})

func (s BidAndMaxMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*BidAndMax)
	if !ok {
		vTmp := value.(BidAndMax)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s BidAndMaxMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := BidAndMax{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetBidAndMaxSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return BidAndMaxJSONSerde{}, nil
	case commtypes.MSGP:
		return BidAndMaxMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
