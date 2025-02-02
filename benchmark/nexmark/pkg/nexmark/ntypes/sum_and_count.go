//go:generate msgp
//msgp:ignore SumAndCountJSONSerde SumAndCountMsgpSerde
package ntypes

import (
	"fmt"
)

type SumAndCount struct {
	Sum   uint64 `json:"sum" msg:"sum"`
	Count uint64 `json:"count" msg:"count"`
}

var _ = fmt.Stringer(SumAndCount{})

func (sc SumAndCount) String() string {
	return fmt.Sprintf("SumAndCount: {Sum: %d, Count: %d}", sc.Sum, sc.Count)
}

func SizeOfSumAndCount(k SumAndCount) int64 {
	return 16
}

func SizeOfSumAndCountPtr(k *SumAndCount) int64 {
	return 16
}
