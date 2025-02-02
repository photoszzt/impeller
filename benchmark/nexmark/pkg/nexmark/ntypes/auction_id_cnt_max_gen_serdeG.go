package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdCntMaxJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdCntMaxJSONSerdeG) String() string {
	return "AuctionIdCntMaxJSONSerdeG"
}

var _ = fmt.Stringer(AuctionIdCntMaxJSONSerdeG{})

var _ = commtypes.SerdeG[AuctionIdCntMax](AuctionIdCntMaxJSONSerdeG{})

type AuctionIdCntMaxMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s AuctionIdCntMaxMsgpSerdeG) String() string {
	return "AuctionIdCntMaxMsgpSerdeG"
}

var _ = fmt.Stringer(AuctionIdCntMaxMsgpSerdeG{})

var _ = commtypes.SerdeG[AuctionIdCntMax](AuctionIdCntMaxMsgpSerdeG{})

func (s AuctionIdCntMaxJSONSerdeG) Encode(value AuctionIdCntMax) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s AuctionIdCntMaxJSONSerdeG) Decode(value []byte) (AuctionIdCntMax, error) {
	v := AuctionIdCntMax{}
	if err := json.Unmarshal(value, &v); err != nil {
		return AuctionIdCntMax{}, err
	}
	return v, nil
}

func (s AuctionIdCntMaxMsgpSerdeG) Encode(value AuctionIdCntMax) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s AuctionIdCntMaxMsgpSerdeG) Decode(value []byte) (AuctionIdCntMax, error) {
	v := AuctionIdCntMax{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return AuctionIdCntMax{}, err
	}
	return v, nil
}

func GetAuctionIdCntMaxSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[AuctionIdCntMax], error) {
	if serdeFormat == commtypes.JSON {
		return AuctionIdCntMaxJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return AuctionIdCntMaxMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
