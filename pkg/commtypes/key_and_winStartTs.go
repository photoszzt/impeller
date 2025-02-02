//go:generate msgp
//msgp:ignore KeyAndWindowStartTs KeyAndWindowStartTsJSONSerde KeyAndWindowStartTsMsgpSerde
package commtypes

type KeyAndWindowStartTsSerialized struct {
	KeySerialized []byte `msg:"k" json:"k"`
	WindowStartTs int64  `msg:"ts" json:"ts"`
}
