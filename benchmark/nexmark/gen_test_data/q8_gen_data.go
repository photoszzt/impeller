package main

import (
	"encoding/json"
	"fmt"
	"os"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
)

type q8AucSpec struct {
	PerTsAfterStart []int64 `json:"per_ts_after_start"`
	AucByPerson     []uint8 `json:"auc_by_person"`
	AucsAfterPerson []int64 `json:"aucs_after_person"`
	Start           int64   `json:"start"`
	End             int64   `json:"end"`
}

type q8AucSpecs struct {
	Q8AucSpecs []q8AucSpec `json:"person_spec"`
}

func q8_gen_data(spec_byte []byte, outputFile string) error {
	spec := q8AucSpecs{}
	err := json.Unmarshal(spec_byte, &spec)
	if err != nil {
		return err
	}
	var events []*ntypes.Event
	personID := 0
	aucID := 0

	for _, aucSpec := range spec.Q8AucSpecs {
		startMs := aucSpec.Start * 1000
		createdAucs := 0
		for pidx, afterStart := range aucSpec.PerTsAfterStart {
			personTs := startMs + afterStart
			personEvent := ntypes.NewPersonEvent(&ntypes.Person{
				DateTime: personTs,
				ID:       uint64(personID),
				Name:     fmt.Sprintf("per_%d", personID),
			})
			events = append(events, personEvent)

			aucByPerson := aucSpec.AucByPerson[pidx]
			fmt.Fprintf(os.Stderr, "aucByPerson: %d, createdAucs: %d\n", aucByPerson, createdAucs)
			for i := 0; i < int(aucByPerson); i++ {
				fmt.Fprintf(os.Stderr, "createdAucs: %d, i: %d\n", createdAucs, i)
				aucTs := personTs + aucSpec.AucsAfterPerson[createdAucs]
				events = append(events, ntypes.NewAuctionEvnet(&ntypes.Auction{
					ID:       uint64(aucID),
					DateTime: aucTs,
					Expires:  aucTs + 5000,
					Seller:   uint64(personID),
					ItemName: fmt.Sprintf("auc_%d", aucID),
				}))
				aucID += 1
				createdAucs += 1
			}
			personID += 1
		}
	}
	return outputEvents(events, outputFile)
}
