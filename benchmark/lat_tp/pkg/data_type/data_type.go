//go:generate msgp
//msgp:ignore PayloadTsJSONSerde PayloadTsMsgpSerde
package datatype

import "sharedlog-stream/pkg/commtypes"

type PayloadTs struct {
	Payload []byte `json:"pl" msg:"pl"`
	Ts      int64  `json:"ts" msg:"ts"`
}

var _ commtypes.EventTimeExtractor = PayloadTs{}

func (pt PayloadTs) ExtractEventTime() (int64, error) {
	return int64(pt.Ts / 1000.0), nil
}
