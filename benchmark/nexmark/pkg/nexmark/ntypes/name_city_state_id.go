//go:generate msgp
//msgp:ignore NameCityStateIdJSONSerde NameCityStateIdMsgpSerde

package ntypes

import (
	"fmt"
)

type NameCityStateId struct {
	Name  string `msg:"name" json:"name"`
	City  string `msg:"city" json:"city"`
	State string `msg:"state" json:"state"`
	ID    uint64 `msg:"id" json:"id"`
}

func SizeOfNameCityStateId(k NameCityStateId) int64 {
	return 8 + int64(len(k.Name)) + int64(len(k.City)) + int64(len(k.State))
}

var _ = fmt.Stringer(NameCityStateId{})

func (ncsi NameCityStateId) String() string {
	return fmt.Sprintf("NameCityStateId: {Name: %s, City: %s, State: %s, ID: %d}",
		ncsi.Name, ncsi.City, ncsi.State, ncsi.ID)
}
