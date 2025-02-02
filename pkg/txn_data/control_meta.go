//go:generate msgp
//msgp:ignore ControlMetadataJSONSerde ControlMetadataMsgpSerde
package txn_data

import (
	"slices"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type ControlMetadata struct {
	Config           map[string]uint8       `json:"sg,omitempty" msgp:"sg,omitempty"`
	KeyMaps          map[string][]KeyMaping `json:"km,omitempty" msgp:"km,omitempty"`
	FinishedPrevTask string                 `json:"fpt,omitempty" msgp:"fpt,omitempty"`
	Epoch            uint16                 `json:"ep,omitempty" msgp:"ep,omitempty"`
	InstanceId       uint8                  `json:"iid,omitempty" msgp:"iid,omitempty"`
}

type KeyMaping struct {
	Key         []byte `json:"k,omitempty" msgp:"k,omitempty"`
	Hash        uint64 `json:"h,omitempty" msgp:"h,omitempty"`
	SubstreamId uint8  `json:"sid,omitempty" msgp:"sid,omitempty"`
}

func (a KeyMaping) Equal(b KeyMaping) bool {
	return a.Hash == b.Hash && a.SubstreamId == b.SubstreamId && slices.Equal(a.Key, b.Key)
}

func StringLess(a, b string) bool {
	return a < b
}

func (a ControlMetadata) Equal(b ControlMetadata) bool {
	opts := cmp.Options{cmpopts.SortMaps(StringLess), cmpopts.EquateEmpty()}
	return a.FinishedPrevTask == b.FinishedPrevTask &&
		a.Epoch == b.Epoch && a.InstanceId == b.InstanceId &&
		cmp.Equal(a.Config, b.Config, opts...) &&
		cmp.Equal(a.KeyMaps, b.KeyMaps, opts...)
}
