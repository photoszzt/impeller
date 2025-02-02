package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type AuctionIdCountJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s AuctionIdCountJSONSerde) String() string {
	return "AuctionIdCountJSONSerde"
}

var _ = fmt.Stringer(AuctionIdCountJSONSerde{})

var _ = commtypes.Serde(AuctionIdCountJSONSerde{})

func (s AuctionIdCountJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionIdCount)
	if !ok {
		vTmp := value.(AuctionIdCount)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s AuctionIdCountJSONSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionIdCount{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type AuctionIdCountMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(AuctionIdCountMsgpSerde{})

func (s AuctionIdCountMsgpSerde) String() string {
	return "AuctionIdCountMsgpSerde"
}

var _ = fmt.Stringer(AuctionIdCountMsgpSerde{})

func (s AuctionIdCountMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*AuctionIdCount)
	if !ok {
		vTmp := value.(AuctionIdCount)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s AuctionIdCountMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := AuctionIdCount{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetAuctionIdCountSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return AuctionIdCountJSONSerde{}, nil
	case commtypes.MSGP:
		return AuctionIdCountMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
