//go:generate msgp
//msgp:ignore AuctionIdSellerJSONSerde AuctionIdSellerMsgpSerde
package ntypes

import (
	"fmt"
)

type AuctionIdSeller struct {
	AucId  uint64 `json:"aucId,omitempty" msg:"aucId,omitempty"`
	Seller uint64 `json:"seller,omitempty" msg:"seller,omitempty"`
}

func SizeOfAuctionIdSeller(k AuctionIdSeller) int64 {
	return 16
}

func CompareAuctionIDSeller(a, b AuctionIdSeller) int {
	if a.AucId < b.AucId {
		return -1
	} else if a.AucId == b.AucId {
		if a.Seller < b.Seller {
			return -1
		} else if a.Seller == b.Seller {
			return 0
		} else {
			return 1
		}
	} else {
		return 1
	}
}

func AuctionIdSellerLess(a, b AuctionIdSeller) bool {
	return CompareAuctionIDSeller(a, b) < 0
}

var _ = fmt.Stringer(AuctionIdSeller{})

func (aic AuctionIdSeller) String() string {
	return fmt.Sprintf("AuctionIdSeller: {AucID: %d, Seller: %d}", aic.AucId, aic.Seller)
}

func CastToAuctionIdSeller(value interface{}) *AuctionIdSeller {
	val, ok := value.(*AuctionIdSeller)
	if !ok {
		valTmp := value.(AuctionIdSeller)
		val = &valTmp
	}
	return val
}
