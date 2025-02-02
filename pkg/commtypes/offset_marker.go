//go:generate msgp
package commtypes

type OffsetMarker struct {
	ConSeqNums map[string]uint64 `json:"offset" msg:"offset"`
}
