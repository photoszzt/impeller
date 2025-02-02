package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type BidAndMaxJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s BidAndMaxJSONSerdeG) String() string {
	return "BidAndMaxJSONSerdeG"
}

var _ = fmt.Stringer(BidAndMaxJSONSerdeG{})

var _ = commtypes.SerdeG[BidAndMax](BidAndMaxJSONSerdeG{})

type BidAndMaxMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s BidAndMaxMsgpSerdeG) String() string {
	return "BidAndMaxMsgpSerdeG"
}

var _ = fmt.Stringer(BidAndMaxMsgpSerdeG{})

var _ = commtypes.SerdeG[BidAndMax](BidAndMaxMsgpSerdeG{})

func (s BidAndMaxJSONSerdeG) Encode(value BidAndMax) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s BidAndMaxJSONSerdeG) Decode(value []byte) (BidAndMax, error) {
	v := BidAndMax{}
	if err := json.Unmarshal(value, &v); err != nil {
		return BidAndMax{}, err
	}
	return v, nil
}

func (s BidAndMaxMsgpSerdeG) Encode(value BidAndMax) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s BidAndMaxMsgpSerdeG) Decode(value []byte) (BidAndMax, error) {
	v := BidAndMax{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return BidAndMax{}, err
	}
	return v, nil
}

func GetBidAndMaxSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[BidAndMax], error) {
	if serdeFormat == commtypes.JSON {
		return BidAndMaxJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return BidAndMaxMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
