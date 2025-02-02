//go:generate msgp
//msgp:ignore OffsetRecordJSONSerde OffsetRecordMsgpSerde
package txn_data

type OffsetRecord struct {
	Offset    uint64 `json:"offset" msg:"os"`
	TaskId    uint64 `json:"aid" msg:"aid"`
	TaskEpoch uint16 `json:"ae" msg:"ae"`
}
