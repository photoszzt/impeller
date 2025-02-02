package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type BidPriceJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s BidPriceJSONSerde) String() string {
	return "BidPriceJSONSerde"
}

var _ = fmt.Stringer(BidPriceJSONSerde{})

var _ = commtypes.Serde(BidPriceJSONSerde{})

func (s BidPriceJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*BidPrice)
	if !ok {
		vTmp := value.(BidPrice)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s BidPriceJSONSerde) Decode(value []byte) (interface{}, error) {
	v := BidPrice{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type BidPriceMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(BidPriceMsgpSerde{})

func (s BidPriceMsgpSerde) String() string {
	return "BidPriceMsgpSerde"
}

var _ = fmt.Stringer(BidPriceMsgpSerde{})

func (s BidPriceMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*BidPrice)
	if !ok {
		vTmp := value.(BidPrice)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s BidPriceMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := BidPrice{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetBidPriceSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return BidPriceJSONSerde{}, nil
	case commtypes.MSGP:
		return BidPriceMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
