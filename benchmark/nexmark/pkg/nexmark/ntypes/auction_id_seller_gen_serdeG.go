package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdSellerJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdSellerJSONSerdeG) String() string {
	return "AuctionIdSellerJSONSerdeG"
}

var _ = fmt.Stringer(AuctionIdSellerJSONSerdeG{})

var _ = commtypes.SerdeG[AuctionIdSeller](AuctionIdSellerJSONSerdeG{})

type AuctionIdSellerMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s AuctionIdSellerMsgpSerdeG) String() string {
	return "AuctionIdSellerMsgpSerdeG"
}

var _ = fmt.Stringer(AuctionIdSellerMsgpSerdeG{})

var _ = commtypes.SerdeG[AuctionIdSeller](AuctionIdSellerMsgpSerdeG{})

func (s AuctionIdSellerJSONSerdeG) Encode(value AuctionIdSeller) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s AuctionIdSellerJSONSerdeG) Decode(value []byte) (AuctionIdSeller, error) {
	v := AuctionIdSeller{}
	if err := json.Unmarshal(value, &v); err != nil {
		return AuctionIdSeller{}, err
	}
	return v, nil
}

func (s AuctionIdSellerMsgpSerdeG) Encode(value AuctionIdSeller) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s AuctionIdSellerMsgpSerdeG) Decode(value []byte) (AuctionIdSeller, error) {
	v := AuctionIdSeller{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return AuctionIdSeller{}, err
	}
	return v, nil
}

func GetAuctionIdSellerSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[AuctionIdSeller], error) {
	if serdeFormat == commtypes.JSON {
		return AuctionIdSellerJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return AuctionIdSellerMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
