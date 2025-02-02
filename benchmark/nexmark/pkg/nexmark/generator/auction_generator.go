package generator

import (
	"math/rand"

	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/utils"
)

const (
	NUM_CATEGORIES   uint32 = 5
	AUCTION_ID_LEAD  uint32 = 10
	HOT_SELLER_RATIO uint32 = 100
)

func NextAuction(eventsCountSoFar uint64,
	eventId uint64, random *rand.Rand, timestamp int64,
	config *GeneratorConfig) *ntypes.Auction {
	id := LastBase0AuctionId(config, eventId) + FIRST_AUCTION_ID
	seller := uint64(0)
	// Here P(auction will be for a hot seller) = 1 - 1/hotSellersRatio.
	if random.Int31n(int32(config.Configuration.HotSellersRatio)) > 0 {
		seller = (LastBase0PersonId(config, eventId) / uint64(HOT_SELLER_RATIO)) * uint64(HOT_SELLER_RATIO)
	} else {
		seller = NextBase0PersonId(eventId, random, config)
	}
	seller += FIRST_PERSON_ID
	category := FIRST_CATEGORY_ID + uint64(random.Intn(int(NUM_CATEGORIES)))
	initialBid := NextPrice(random)
	expires := timestamp + nextAuctionLenghMs(eventsCountSoFar, random, timestamp, config)
	name := NextString(random, 20)
	desc := NextString(random, 100)
	reserve := initialBid + NextPrice(random)
	currentSize := 8 + len(name) + len(desc) + 8 + 8 + 8 + 8 + 8
	extra := NextExtra(random, uint32(currentSize), config.Configuration.AvgAuctionByteSize)
	return &ntypes.Auction{
		ID:          id,
		ItemName:    name,
		Description: desc,
		InitialBid:  initialBid,
		Reserve:     reserve,
		DateTime:    timestamp,
		Expires:     expires,
		Seller:      seller,
		Category:    category,
		Extra:       extra,
	}
}

func LastBase0AuctionId(config *GeneratorConfig, eventId uint64) uint64 {
	epoch := eventId / uint64(config.TotalProportion)
	offset := eventId % uint64(config.TotalProportion)
	if offset < uint64(config.PersonProportion) {
		epoch -= 1
		offset = uint64(config.AuctionProportion - 1)
	} else if offset >= uint64(config.PersonProportion+config.AuctionProportion) {
		offset = uint64(config.AuctionProportion) - 1
	} else {
		offset -= uint64(config.PersonProportion)
	}
	return epoch*uint64(config.AuctionProportion) + offset
}

func NextBase0AuctionId(nextEventId uint64, random *rand.Rand, config *GeneratorConfig) uint64 {
	minAuction := utils.MaxUint64(LastBase0AuctionId(config, nextEventId)-uint64(config.Configuration.NumInFlightAuctions), 0)
	maxAuction := LastBase0AuctionId(config, nextEventId)
	return minAuction + NextUint64(random, maxAuction-minAuction+1+uint64(AUCTION_ID_LEAD))
}

// Return a random time delay, in milliseconds, for length of auctions.
func nextAuctionLenghMs(eventsCountSoFar uint64, random *rand.Rand, timestamp int64, config *GeneratorConfig) int64 {
	// What's our current event number?
	currentEventNumber := config.NextAdjustedEventNumber(eventsCountSoFar)
	// How many events till we've generated numInFlightAuctions?
	numEventsForAuctions := uint64(config.Configuration.NumInFlightAuctions) * uint64(config.TotalProportion) / uint64(config.AuctionProportion)
	// When will the auction numInFlightAuctions beyond now be generated?
	futureAuction := config.TimestampForEvent(currentEventNumber + numEventsForAuctions)
	// Choose a length with average horizonMs.
	horizonMs := futureAuction - timestamp
	return 1 + NextInt64(random, utils.MaxInt64(horizonMs*2, 1))
}
