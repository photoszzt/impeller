//go:generate msgp
//msgp:ignore EpochMetaJSONSerde EpochMetaMsgpSerde EpochMetaMsgpSerdeG EpochJSONMsgpSerdeG
//go:generate stringer -type=EpochMark
package commtypes

import (
	"fmt"
)

type EpochMark uint8

const (
	EMPTY     = EpochMark(iota)
	EPOCH_END // epoch end, for 2pc protocol, it's commit
	ABORT
	SCALE_FENCE
	FENCE
	STREAM_END
	CHKPT_MARK
)

type ProduceRange struct {
	Start        uint64 `json:"s" msg:"s"`
	SubStreamNum uint8  `json:"sNum" msg:"sNum"`
}

type SeqRange struct {
	End   uint64
	Start uint64
}

func (s SeqRange) String() string {
	return fmt.Sprintf("SeqRange: {Start: %#x, End: %#x}", s.Start, s.End)
}

type SeqRangeSet map[SeqRange]struct{}

func NewSeqRangeSet() SeqRangeSet {
	return make(map[SeqRange]struct{})
}

func (s SeqRangeSet) Add(val SeqRange) {
	s[val] = struct{}{}
}

func (s SeqRangeSet) Remove(val SeqRange) {
	delete(s, val)
}

func (s SeqRangeSet) Has(val SeqRange) bool {
	_, ok := s[val]
	return ok
}

var _ = fmt.Stringer(ProduceRange{})

func (p ProduceRange) String() string {
	return fmt.Sprintf("ProduceRange: {Start: 0x%x, SubStreamNum: %d}", p.Start, p.SubStreamNum)
}

type EpochMarker struct {
	ConSeqNums   map[string]uint64         `json:"ConSeqNum,omitempty" msg:"ConSeqNum,omitempty"`
	OutputRanges map[string][]ProduceRange `json:"outRng,omitempty" msg:"outRng,omitempty"`
	// it holds epoch generation number for align checkpoint
	StartTime  int64     `json:"startTime,omitempty" msg:"startTime,omitempty"`
	ScaleEpoch uint16    `json:"sepoch,omitempty" msg:"sepoch,omitempty"`
	ProdIndex  uint8     `json:"prodIndex,omitempty" msg:"prodIndex,omitempty"`
	Mark       EpochMark `json:"mark,omitempty" msg:"mark,omitempty"`
}
