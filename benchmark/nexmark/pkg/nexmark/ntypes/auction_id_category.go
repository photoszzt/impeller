//go:generate msgp
//msgp:ignore AuctionIdCategoryJSONSerde AuctionIdCategoryMsgpSerde
package ntypes

import (
	"fmt"
)

type AuctionIdCategory struct {
	AucId    uint64 `json:"aucId,omitempty" msg:"aucId,omitempty"`
	Category uint64 `json:"cat,omitempty" msg:"cat,omitempty"`
}

func SizeOfAuctionIdCategory(k AuctionIdCategory) int64 {
	return 16
}

var _ = fmt.Stringer(AuctionIdCategory{})

func (aic AuctionIdCategory) String() string {
	return fmt.Sprintf("AuctionIdCat: {AucID: %d, Cat: %d}", aic.AucId, aic.Category)
}

func CompareAuctionIdCategory(a, b *AuctionIdCategory) int {
	if a.AucId < b.AucId {
		return -1
	} else if a.AucId == b.AucId {
		if a.Category < b.Category {
			return -1
		} else if a.Category == b.Category {
			return 0
		} else {
			return 1
		}
	} else {
		return 1
	}
}

func AuctionIdCategoryLess(a, b AuctionIdCategory) bool {
	return CompareAuctionIdCategory(&a, &b) < 0
}
