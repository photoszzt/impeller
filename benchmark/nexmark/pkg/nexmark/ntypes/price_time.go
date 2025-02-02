//go:generate msgp
//msgp:ignore PriceTimeJSONSerde PriceTimeMsgpSerde
package ntypes

import (
	"fmt"
)

type PriceTime struct {
	Price    uint64 `json:"price" msg:"price"`
	DateTime int64  `json:"dateTime" msg:"dateTime"` // unix timestamp in ms
}

func SizeOfPriceTime(k PriceTime) int64 {
	return 16
}

func SizeOfPriceTimePtrIn(k *PriceTime) int64 {
	return 16
}

func ComparePriceTime(a, b PriceTime) int {
	if a.Price < b.Price {
		return -1
	} else if a.Price == b.Price {
		if a.DateTime < b.DateTime {
			return -1
		} else if a.DateTime == b.DateTime {
			return 0
		} else {
			return 1
		}
	} else {
		return 1
	}
}

var _ = fmt.Stringer(PriceTime{})

func (pt PriceTime) String() string {
	return fmt.Sprintf("PriceTime: {Price: %d, Ts: %d}",
		pt.Price, pt.DateTime)
}

func CastToPriceTimePtr(value interface{}) *PriceTime {
	pt, ok := value.(*PriceTime)
	if !ok {
		ptTmp := value.(PriceTime)
		pt = &ptTmp
	}
	return pt
}
