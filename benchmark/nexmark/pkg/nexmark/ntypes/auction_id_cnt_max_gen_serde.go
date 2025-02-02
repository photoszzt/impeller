package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdCntMaxJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdCntMaxJSONSerde) String() string {
	return "AuctionIdCntMaxJSONSerde"
}

var _ = fmt.Stringer(AuctionIdCntMaxJSONSerde{})

var _ = commtypes.Serde(AuctionIdCntMaxJSONSerde{})

func (s AuctionIdCntMaxJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionIdCntMax)
	if !ok {
		vTmp := value.(AuctionIdCntMax)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s AuctionIdCntMaxJSONSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionIdCntMax{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type AuctionIdCntMaxMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(AuctionIdCntMaxMsgpSerde{})

func (s AuctionIdCntMaxMsgpSerde) String() string {
	return "AuctionIdCntMaxMsgpSerde"
}

var _ = fmt.Stringer(AuctionIdCntMaxMsgpSerde{})

func (s AuctionIdCntMaxMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionIdCntMax)
	if !ok {
		vTmp := value.(AuctionIdCntMax)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s AuctionIdCntMaxMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionIdCntMax{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetAuctionIdCntMaxSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return AuctionIdCntMaxJSONSerde{}, nil
	case commtypes.MSGP:
		return AuctionIdCntMaxMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
