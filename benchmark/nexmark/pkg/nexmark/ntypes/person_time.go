//go:generate msgp
//msgp:ignore PersonTimeJSONEncoder PersonTimeJSONDecoder PersonTimeJSONSerde
//msgp:ignore PersonTimeMsgpEncoder PersonTimeMsgpDecoder PersonTimeMsgpSerde
package ntypes

import (
	"fmt"
)

type PersonTime struct {
	Name      string `json:"name" msg:"name"`
	ID        uint64 `json:"id" msg:"id"`
	StartTime int64  `json:"startTime" msg:"startTime"`
}

func SizeOfPersonTime(k PersonTime) int64 {
	return 16 + int64(len(k.Name))
}

var _ = fmt.Stringer(PersonTime{})

func (pt PersonTime) String() string {
	return fmt.Sprintf("PersonTime: {Name: %s, ID: %d, StartTime: %d}",
		pt.Name, pt.ID, pt.StartTime)
}
