package main

import (
	"encoding/json"
	"math/rand"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
)

type q7BidsSpec struct {
	BidsSpec []bidSpec `json:"bids_spec"`
}

type bidSpec struct {
	Start int64  `json:"start"`
	End   int64  `json:"end"`
	Max   uint64 `json:"max"`
	Num   uint32 `json:"num"`
}

func q7_gen_data(spec_byte []byte, outputFile string) error {
	spec := q7BidsSpec{}
	err := json.Unmarshal(spec_byte, &spec)
	if err != nil {
		return err
	}
	r := rand.New(rand.NewSource(3))
	var events []*ntypes.Event
	for _, bidSpec := range spec.BidsSpec {
		startMs := bidSpec.Start * 1000
		endMs := bidSpec.End * 1000
		diffMs := endMs - startMs
		t := startMs + r.Int63n(diffMs)
		events = append(events, ntypes.NewBidEvent(&ntypes.Bid{
			DateTime: t,
			Price:    bidSpec.Max,
		}))
		for i := 0; i < int(bidSpec.Num)-1; i++ {
			v := uint64(r.Int63n(int64(bidSpec.Max)))
			t := startMs + r.Int63n(diffMs)
			events = append(events, ntypes.NewBidEvent(&ntypes.Bid{
				DateTime: t,
				Price:    v,
			}))
		}
	}
	return outputEvents(events, outputFile)
}
