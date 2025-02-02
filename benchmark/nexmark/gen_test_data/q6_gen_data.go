package main

import (
	ntypes "sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
)

func q6_gen_data(outputFile string) error {
	aucs_per_seller := 10
	bidsPerAuction := 2
	seller_spec := map[ntypes.StartEndTime]map[int]int{
		{StartTimeMs: 1000, EndTimeMs: 1999}: {1: 10, 2: 20, 3: 30},
		{StartTimeMs: 2000, EndTimeMs: 2999}: {1: 40, 2: 50, 3: 60},
	}
	var events []*ntypes.Event
	aucId := uint64(1)
	sellers := []int{1, 2, 3}
	for startEnd, sellerAucMap := range seller_spec {
		for _, seller := range sellers {
			startWinningBids := sellerAucMap[seller]
			for i := 0; i < aucs_per_seller; i++ {
				events = append(events, ntypes.NewAuctionEvnet(
					&ntypes.Auction{
						Seller:   uint64(seller),
						DateTime: startEnd.StartTimeMs,
						Expires:  startEnd.EndTimeMs,
						ID:       aucId,
					}))
				ts := startEnd.StartTimeMs + int64(seller*aucs_per_seller+i)
				for j := 0; j < bidsPerAuction-1; j++ {
					events = append(events, ntypes.NewBidEvent(&ntypes.Bid{
						DateTime: ts,
						Price:    uint64(startWinningBids+i-j) - 1,
						Auction:  aucId,
					}))
				}
				events = append(events, ntypes.NewBidEvent(&ntypes.Bid{
					DateTime: ts,
					Price:    uint64(startWinningBids + i),
					Auction:  aucId,
				}))
				aucId += 1
			}
		}
	}
	return outputEvents(events, outputFile)
}
