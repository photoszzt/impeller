package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionBidJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionBidJSONSerdeG) String() string {
	return "AuctionBidJSONSerdeG"
}

var _ = fmt.Stringer(AuctionBidJSONSerdeG{})

var _ = commtypes.SerdeG[*AuctionBid](AuctionBidJSONSerdeG{})

type AuctionBidMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s AuctionBidMsgpSerdeG) String() string {
	return "AuctionBidMsgpSerdeG"
}

var _ = fmt.Stringer(AuctionBidMsgpSerdeG{})

var _ = commtypes.SerdeG[*AuctionBid](AuctionBidMsgpSerdeG{})

func (s AuctionBidJSONSerdeG) Encode(value *AuctionBid) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s AuctionBidJSONSerdeG) Decode(value []byte) (*AuctionBid, error) {
	v := AuctionBid{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s AuctionBidMsgpSerdeG) Encode(value *AuctionBid) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s AuctionBidMsgpSerdeG) Decode(value []byte) (*AuctionBid, error) {
	v := AuctionBid{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetAuctionBidSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[*AuctionBid], error) {
	if serdeFormat == commtypes.JSON {
		return AuctionBidJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return AuctionBidMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
