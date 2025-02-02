//go:generate msgp
package ntypes

import (
	"fmt"
)

type AuctionBid struct {
	BidDateTime int64  `json:"bidDateTime" msg:"bidDateTime"`
	AucDateTime int64  `json:"aucDateTime" msg:"aucDateTime"`
	AucExpires  int64  `json:"aucExpires" msg:"aucExpires"`
	BidPrice    uint64 `json:"bidPrice" msg:"bidPrice"`
	AucCategory uint64 `json:"aucCategory" msg:"aucCategory"`
	AucSeller   uint64 `json:"aucSeller,omitempty" msg:"aucSeller,omitempty"`
}

func SizeOfAuctionBid(k *AuctionBid) int64 {
	return 48
}

var _ = fmt.Stringer(AuctionBid{})

func (ab AuctionBid) String() string {
	return fmt.Sprintf("AuctionBid: {BidTs: %d, AucTs: %d, AucExpiresTs: %d, BidPrice: %d, AucCat: %d}",
		ab.BidDateTime, ab.AucDateTime, ab.AucExpires, ab.BidPrice, ab.AucCategory)
}
