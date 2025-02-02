//go:generate msgp
package commtypes

type PayloadArr struct {
	Payloads [][]byte `json:"parr,omitempty" msg:"parr,omitempty"`
}
