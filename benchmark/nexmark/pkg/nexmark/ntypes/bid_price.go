//go:generate msgp
//msgp:ignore BidPriceJSONSerde BidPriceMsgpSerde
package ntypes

type BidPrice struct {
	Price uint64 `json:"price" msg:"price"`
}

func SizeOfBidPrice(k BidPrice) int64 {
	return 8
}

func SizeOfBidPricePtrIn(k *BidPrice) int64 {
	return 8
}
