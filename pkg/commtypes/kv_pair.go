//go:generate msgp
package commtypes

type KeyValueSerrialized struct {
	KeyEnc   []byte `json:"key,omitempty" msg:"key,omitempty"`
	ValueEnc []byte `json:"val,omitempty" msg:"val,omitempty"`
}

type KeyValueSers []KeyValueSerrialized
