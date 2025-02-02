//go:generate msgp
//msgp:ignore BidAndMaxJSONSerde BidAndMaxMsgpSerde
package ntypes

import (
	"fmt"
)

type BidAndMax struct {
	Price    uint64 `json:"price" msg:"price"`
	Auction  uint64 `json:"auction" msg:"auction"`
	Bidder   uint64 `json:"bidder" msg:"bidder"`
	BidTs    int64  `json:"bTs" msg:"bTs"`
	WStartMs int64  `json:"wStartMs" msg:"wStartMs"`
	WEndMs   int64  `json:"wEndMs" msg:"wEndMs"`
}

func SizeOfBidAndMax(k BidAndMax) int64 {
	return 48
}

var _ = fmt.Stringer(BidAndMax{})

func (bm BidAndMax) String() string {
	return fmt.Sprintf("BidAndMax: {Price: %d, Auction: %d, Bidder: %d, BidTs: %d, WinStartMs: %d, WinEndMs: %d}",
		bm.Price, bm.Auction, bm.Bidder, bm.BidTs, bm.WStartMs, bm.WEndMs)
}
