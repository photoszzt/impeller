package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdCategoryJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdCategoryJSONSerde) String() string {
	return "AuctionIdCategoryJSONSerde"
}

var _ = fmt.Stringer(AuctionIdCategoryJSONSerde{})

var _ = commtypes.Serde(AuctionIdCategoryJSONSerde{})

func (s AuctionIdCategoryJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionIdCategory)
	if !ok {
		vTmp := value.(AuctionIdCategory)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s AuctionIdCategoryJSONSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionIdCategory{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type AuctionIdCategoryMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(AuctionIdCategoryMsgpSerde{})

func (s AuctionIdCategoryMsgpSerde) String() string {
	return "AuctionIdCategoryMsgpSerde"
}

var _ = fmt.Stringer(AuctionIdCategoryMsgpSerde{})

func (s AuctionIdCategoryMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionIdCategory)
	if !ok {
		vTmp := value.(AuctionIdCategory)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s AuctionIdCategoryMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionIdCategory{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetAuctionIdCategorySerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return AuctionIdCategoryJSONSerde{}, nil
	case commtypes.MSGP:
		return AuctionIdCategoryMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
