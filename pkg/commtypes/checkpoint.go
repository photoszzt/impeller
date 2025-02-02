//go:generate msgp
package commtypes

type ChkptMetaData struct {
	Unprocessed []uint64 `json:"up,omitempty" msg:"up,omitempty"`
	// stores the last marker seqnumber for align checkpoint
	LastChkptMarker uint64 `json:"lcpm,omitempty" msg:"lcpm,omitempty"`
}

type Checkpoint struct {
	KvArr     [][]byte        `json:"kvarr,omitempty" msg:"kvarr,omitempty"`
	ChkptMeta []ChkptMetaData `json:"chkptm,omitempty" msg:"chkptm,omitempty"`
}
