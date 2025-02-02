package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type BidPriceJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s BidPriceJSONSerdeG) String() string {
	return "BidPriceJSONSerdeG"
}

var _ = fmt.Stringer(BidPriceJSONSerdeG{})

var _ = commtypes.SerdeG[BidPrice](BidPriceJSONSerdeG{})

type BidPriceMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s BidPriceMsgpSerdeG) String() string {
	return "BidPriceMsgpSerdeG"
}

var _ = fmt.Stringer(BidPriceMsgpSerdeG{})

var _ = commtypes.SerdeG[BidPrice](BidPriceMsgpSerdeG{})

func (s BidPriceJSONSerdeG) Encode(value BidPrice) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s BidPriceJSONSerdeG) Decode(value []byte) (BidPrice, error) {
	v := BidPrice{}
	if err := json.Unmarshal(value, &v); err != nil {
		return BidPrice{}, err
	}
	return v, nil
}

func (s BidPriceMsgpSerdeG) Encode(value BidPrice) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s BidPriceMsgpSerdeG) Decode(value []byte) (BidPrice, error) {
	v := BidPrice{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return BidPrice{}, err
	}
	return v, nil
}

func GetBidPriceSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[BidPrice], error) {
	if serdeFormat == commtypes.JSON {
		return BidPriceJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return BidPriceMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
