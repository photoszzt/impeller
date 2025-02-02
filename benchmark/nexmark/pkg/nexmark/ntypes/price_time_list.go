//go:generate msgp
//msgp:ignore PriceTimeListJSONSerde PriceTimeListMsgpSerde
package ntypes

import (
	"fmt"
)

type PriceTimeSlice []PriceTime

func (t PriceTimeSlice) Len() int {
	return len(t)
}

func (t PriceTimeSlice) Less(i, j int) bool {
	return ComparePriceTime(t[i], t[j]) < 0
}

func (t PriceTimeSlice) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func Delete(t PriceTimeSlice, i, j int) PriceTimeSlice {
	return append(t[:i], t[j:]...)
}

func RemoveMatching(t PriceTimeSlice, pt *PriceTime) PriceTimeSlice {
	for i, v := range t {
		if v.Price == pt.Price && v.DateTime == pt.DateTime {
			return Delete(t, i, i+1)
		}
	}
	return t
}

type PriceTimeList struct {
	PTList PriceTimeSlice `json:"ptList" msg:"ptList"`
}

func SizeOfPriceTimeList(ptl PriceTimeList) int64 {
	return int64(16 * len(ptl.PTList))
}

var _ = fmt.Stringer(PriceTimeList{})

func (ptl PriceTimeList) String() string {
	return fmt.Sprintf("PriceTimeList: {PTList: %v}", ptl.PTList)
}

func castToPriceTimeListPtr(value interface{}) *PriceTimeList {
	ptl, ok := value.(*PriceTimeList)
	if !ok {
		ptlTmp := value.(PriceTimeList)
		ptl = &ptlTmp
	}
	return ptl
}
