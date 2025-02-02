package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdCategoryJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdCategoryJSONSerdeG) String() string {
	return "AuctionIdCategoryJSONSerdeG"
}

var _ = fmt.Stringer(AuctionIdCategoryJSONSerdeG{})

var _ = commtypes.SerdeG[AuctionIdCategory](AuctionIdCategoryJSONSerdeG{})

type AuctionIdCategoryMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s AuctionIdCategoryMsgpSerdeG) String() string {
	return "AuctionIdCategoryMsgpSerdeG"
}

var _ = fmt.Stringer(AuctionIdCategoryMsgpSerdeG{})

var _ = commtypes.SerdeG[AuctionIdCategory](AuctionIdCategoryMsgpSerdeG{})

func (s AuctionIdCategoryJSONSerdeG) Encode(value AuctionIdCategory) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s AuctionIdCategoryJSONSerdeG) Decode(value []byte) (AuctionIdCategory, error) {
	v := AuctionIdCategory{}
	if err := json.Unmarshal(value, &v); err != nil {
		return AuctionIdCategory{}, err
	}
	return v, nil
}

func (s AuctionIdCategoryMsgpSerdeG) Encode(value AuctionIdCategory) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s AuctionIdCategoryMsgpSerdeG) Decode(value []byte) (AuctionIdCategory, error) {
	v := AuctionIdCategory{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return AuctionIdCategory{}, err
	}
	return v, nil
}

func GetAuctionIdCategorySerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[AuctionIdCategory], error) {
	if serdeFormat == commtypes.JSON {
		return AuctionIdCategoryJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return AuctionIdCategoryMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
