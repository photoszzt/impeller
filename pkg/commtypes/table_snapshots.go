//go:generate msgp
//msgp:ignore TableSnapshotsJSONSerde TableSnapshotsMsgpSerde
package commtypes

type TableSnapshots struct {
	TabMaps map[string][][]byte `json:"tab_maps" msg:"tab_maps"`
}
