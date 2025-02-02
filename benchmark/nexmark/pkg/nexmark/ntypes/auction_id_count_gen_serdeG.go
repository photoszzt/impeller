package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdCountJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdCountJSONSerdeG) String() string {
	return "AuctionIdCountJSONSerdeG"
}

var _ = fmt.Stringer(AuctionIdCountJSONSerdeG{})

var _ = commtypes.SerdeG[AuctionIdCount](AuctionIdCountJSONSerdeG{})

type AuctionIdCountMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s AuctionIdCountMsgpSerdeG) String() string {
	return "AuctionIdCountMsgpSerdeG"
}

var _ = fmt.Stringer(AuctionIdCountMsgpSerdeG{})

var _ = commtypes.SerdeG[AuctionIdCount](AuctionIdCountMsgpSerdeG{})

func (s AuctionIdCountJSONSerdeG) Encode(value AuctionIdCount) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s AuctionIdCountJSONSerdeG) Decode(value []byte) (AuctionIdCount, error) {
	v := AuctionIdCount{}
	if err := json.Unmarshal(value, &v); err != nil {
		return AuctionIdCount{}, err
	}
	return v, nil
}

func (s AuctionIdCountMsgpSerdeG) Encode(value AuctionIdCount) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s AuctionIdCountMsgpSerdeG) Decode(value []byte) (AuctionIdCount, error) {
	v := AuctionIdCount{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return AuctionIdCount{}, err
	}
	return v, nil
}

func GetAuctionIdCountSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[AuctionIdCount], error) {
	if serdeFormat == commtypes.JSON {
		return AuctionIdCountJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return AuctionIdCountMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
