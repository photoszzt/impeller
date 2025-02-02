package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionBidJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionBidJSONSerde) String() string {
	return "AuctionBidJSONSerde"
}

var _ = fmt.Stringer(AuctionBidJSONSerde{})

var _ = commtypes.Serde(AuctionBidJSONSerde{})

func (s AuctionBidJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionBid)
	if !ok {
		vTmp := value.(AuctionBid)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s AuctionBidJSONSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionBid{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

type AuctionBidMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

func (s AuctionBidMsgpSerde) String() string {
	return "AuctionBidMsgpSerde"
}

var _ = fmt.Stringer(AuctionBidMsgpSerde{})

var _ = commtypes.Serde(AuctionBidMsgpSerde{})

func (s AuctionBidMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionBid)
	if !ok {
		vTmp := value.(AuctionBid)
		v = &vTmp
	}
	// b := commtypes.PopBuffer(v.Msgsize())
	// buf := *b
	// r, err := v.MarshalMsg(buf[:0])
	r, err := v.MarshalMsg(nil)
	return r, nil, err
}

func (s AuctionBidMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionBid{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetAuctionBidSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return AuctionBidJSONSerde{}, nil
	case commtypes.MSGP:
		return AuctionBidMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
