//go:generate msgp
//msgp:ignore StartEndTimeJSONEncoder StartEndTimeJSONDecoder StartEndTimeJSONSerde
//msgp:ignore StartEndTimeMsgpEncoder StartEndTimeMsgpDecoder StartEndTimeMsgpSerde
package ntypes

import (
	"fmt"
	"sharedlog-stream/pkg/commtypes"
)

type StartEndTime struct {
	StartTimeMs int64 `json:"sTs" msg:"sTs"`
	EndTimeMs   int64 `json:"eTs" msg:"eTs"`
}

func SizeOfStartEndTime(k StartEndTime) int64 {
	return 16
}

var _ = fmt.Stringer(StartEndTime{})

func (se StartEndTime) Stringer() string {
	return fmt.Sprintf("StartEndTime: {StartTsMs: %d, EndTsMs: %d}", se.StartTimeMs, se.EndTimeMs)
}

func CompareStartEndTime(a, b StartEndTime) int {
	if a.StartTimeMs < b.StartTimeMs {
		return -1
	} else { // a.st >= b.st
		if a.StartTimeMs == b.StartTimeMs {
			if a.EndTimeMs < b.EndTimeMs {
				return -1
			} else if a.EndTimeMs == b.EndTimeMs {
				return 0
			} else {
				return 1
			}
		} else {
			return 1
		}
	}
}

var _ = commtypes.EventTimeExtractor(&StartEndTime{})

func (se *StartEndTime) ExtractEventTime() (int64, error) {
	return se.StartTimeMs, nil
}

func (se StartEndTime) String() string {
	return fmt.Sprintf("%d %d", se.StartTimeMs, se.EndTimeMs)
}
