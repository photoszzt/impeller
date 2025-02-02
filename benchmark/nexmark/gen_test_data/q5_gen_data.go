package main

import (
	"encoding/json"
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"strconv"
	"strings"
)

type q5Spec struct {
	Bids     map[string]int `json:"bids"`
	ID       uint64         `json:"id"`
	DateTime int64          `json:"dateTime"`
	Expires  int64          `json:"expires"`
}
type q5Specs struct {
	Auc []q5Spec
}

type events_arr struct {
	Earr []*ntypes.Event `json:"events"`
}

func q5_gen_data(spec_byte []byte, outputFile string) error {
	spec := q5Specs{}
	err := json.Unmarshal(spec_byte, &spec)
	if err != nil {
		return err
	}
	// fmt.Fprintf(os.Stderr, "%v\n", spec)
	var events []*ntypes.Event
	bids := make(map[uint64][]*ntypes.Event)
	for _, auc_spec := range spec.Auc {
		e := ntypes.NewAuctionEvnet(&ntypes.Auction{
			ID:       auc_spec.ID,
			DateTime: auc_spec.DateTime,
			Expires:  auc_spec.Expires,
		})
		bid_events, ok := bids[auc_spec.ID]
		if !ok {
			bid_events = make([]*ntypes.Event, 0)
		}
		events = append(events, e)
		// fmt.Fprintf(os.Stderr, "bids spec: %v\n", auc_spec.Bids)
		for k, v := range auc_spec.Bids {
			start_end := strings.Split(k, "-")
			// fmt.Fprintf(os.Stderr, "start end: %v, v: %d\n", start_end, v)
			start, err := strconv.Atoi(start_end[0])
			if err != nil {
				return err
			}
			t := start*1000 + 500
			bid_event := ntypes.NewBidEvent(&ntypes.Bid{
				Bidder:   1,
				Auction:  auc_spec.ID,
				DateTime: int64(t),
				Price:    100,
			})
			for i := 0; i < v; i++ {
				bid_events = append(bid_events, bid_event)
			}
			bids[auc_spec.ID] = bid_events
			// fmt.Fprintf(os.Stderr, "generated bids: %v\n", bids[auc_spec.ID])
		}
	}
	for _, v := range bids {
		// fmt.Fprintf(os.Stderr, "%d: %v\n", k, v)
		events = append(events, v...)
	}
	// eventSerde, err := ntypes.GetEventSerde(commtypes.JSON)
	// if err != nil {
	// 	return err
	// }
	// for _, e := range events {
	// 	e_bytes, err := eventSerde.Encode(e)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Fprintf(os.Stderr, "%s\n", e_bytes)
	// }
	return outputEvents(events, outputFile)
}
