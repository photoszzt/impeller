//go:generate msgp
//msgp:ignore AuctionIdCntMaxJSONSerde AuctionIdCntMaxMsgpSerde
package ntypes

import (
	"fmt"
)

type AuctionIdCntMax struct {
	AucId  uint64 `json:"aucId" msg:"aucId"`
	Count  uint64 `json:"cnt" msg:"cnt"`
	MaxCnt uint64 `json:"maxCnt" msg:"maxCnt"`
}

func SizeOfAuctionIdCntMax(k AuctionIdCntMax) int64 {
	return 24
}

var _ = fmt.Stringer(AuctionIdCntMax{})

func (aicm AuctionIdCntMax) String() string {
	return fmt.Sprintf("AuctionIdCntMax: {AucID: %d, Count: %d, MaxCnt: %d}",
		aicm.AucId, aicm.Count, aicm.MaxCnt)
}
