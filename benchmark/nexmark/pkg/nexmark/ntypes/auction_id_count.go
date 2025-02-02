//go:generate msgp
//msgp:ignore AuctionIdCountJSONEncoder AuctionIdCountJSONDecoder AuctionIdCountJSONSerde
//msgp:ignore AuctionIdCountMsgpEncoder AuctionIdCountMsgpDecoder AuctionIdCountMsgpSerde
package ntypes

import (
	"fmt"
)

type AuctionIdCount struct {
	AucId uint64 `json:"aucId,omitempty" msg:"aucId,omitempty"`
	Count uint64 `json:"cnt,omitempty" msg:"cnt,omitempty"`
}

func SizeOfAuctionIdCount(k AuctionIdCount) int64 {
	return 16
}

var _ = fmt.Stringer(AuctionIdCount{})

func (aic AuctionIdCount) String() string {
	return fmt.Sprintf("AuctionIdCount: {AucId: %d, Count: %d}", aic.AucId, aic.Count)
}
