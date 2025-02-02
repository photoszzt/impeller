package generator

import (
	"math"

	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/utils"
)

const (
	FIRST_AUCTION_ID  uint64 = 1000
	FIRST_PERSON_ID   uint64 = 1000
	FIRST_CATEGORY_ID uint64 = 10
)

type GeneratorConfig struct {
	Configuration     *ntypes.NexMarkConfig
	InterEventDelayUs []float64
	EventPerEpoch     uint64
	MaxEvents         uint64
	FirstEventNumber  uint64
	EpochPeriodMs     uint64
	StepLengthSec     uint64
	BaseTime          int64
	FirstEventId      uint64
	TotalProportion   uint32
	BidProportion     uint32
	AuctionProportion uint32
	PersonProportion  uint32
}

func NewGeneratorConfig(configuration *ntypes.NexMarkConfig, baseTime int64 /* millisecond */, firstEventId uint64,
	maxEventsOrZero uint64, firstEventNumber uint64) *GeneratorConfig {
	g := new(GeneratorConfig)
	g.AuctionProportion = configuration.AuctionProportion
	g.PersonProportion = configuration.PersonProportion
	g.BidProportion = configuration.BidProportion
	g.TotalProportion = configuration.AuctionProportion + configuration.PersonProportion + configuration.BidProportion
	g.Configuration = configuration
	g.InterEventDelayUs = []float64{1000000.0 / float64(configuration.FirstEventRate) * float64(configuration.NumEventGenerators)}
	g.StepLengthSec = uint64(configuration.RateShape.StepLengthSec(configuration.RatePeriodSec))
	g.BaseTime = baseTime
	g.FirstEventId = firstEventId
	if maxEventsOrZero == 0 {
		g.MaxEvents = math.MaxUint64 / (uint64(g.TotalProportion) * uint64(utils.MaxUint32(
			utils.MaxUint32(configuration.AvgPersonByteSize, configuration.AvgAuctionByteSize),
			configuration.AvgBidByteSize)))
	} else {
		g.MaxEvents = maxEventsOrZero
	}
	g.FirstEventNumber = firstEventNumber

	g.EventPerEpoch = 0
	g.EpochPeriodMs = 0
	return g
}

func (gc *GeneratorConfig) Split(n uint32) []*GeneratorConfig {
	var results []*GeneratorConfig
	if n == 1 {
		results = append(results, gc)
	} else {
		subMaxEvents := gc.MaxEvents / uint64(n)
		subFirstEventId := gc.FirstEventId
		for i := uint32(0); i < n; i++ {
			if i == n-1 {
				subMaxEvents = gc.MaxEvents - subMaxEvents*uint64(n-1)
			}
			results = append(results, gc.CopyWith(subFirstEventId, subMaxEvents, gc.FirstEventNumber))
			subFirstEventId += subMaxEvents
		}
	}
	return results
}

func (gc *GeneratorConfig) CopyWith(firstEventId, maxEvents, firstEventNumber uint64) *GeneratorConfig {
	return NewGeneratorConfig(gc.Configuration, gc.BaseTime, firstEventId, maxEvents, firstEventNumber)
}

func (gc *GeneratorConfig) EstimateBytesForEvents(numEvents uint64) uint64 {
	numPersons := (numEvents * uint64(gc.PersonProportion)) / uint64(gc.TotalProportion)
	numAuctions := (numEvents * uint64(gc.AuctionProportion)) / uint64(gc.TotalProportion)
	numBids := uint64(numEvents*uint64(gc.BidProportion)) / uint64(gc.TotalProportion)
	return uint64(numPersons*uint64(gc.Configuration.AvgPersonByteSize) + numAuctions*uint64(gc.Configuration.AvgAuctionByteSize) + numBids*uint64(gc.Configuration.AvgBidByteSize))
}

func (gc *GeneratorConfig) GetEstimatedSizeBytes() uint64 {
	return gc.EstimateBytesForEvents(gc.MaxEvents)
}

func (gc *GeneratorConfig) GetStartEventId() uint64 {
	return gc.FirstEventId + gc.FirstEventNumber
}

func (gc *GeneratorConfig) GetStopEventId() uint64 {
	return gc.FirstEventId + gc.FirstEventNumber + gc.MaxEvents
}

func (gc *GeneratorConfig) NextEventNumber(numEvents uint64) uint64 {
	return gc.FirstEventNumber + numEvents
}

func (gc *GeneratorConfig) NextAdjustedEventNumber(numEvents uint64) uint64 {
	n := gc.Configuration.OutOfOrderGroupSize
	eventNumber := gc.NextEventNumber(numEvents)
	base := (eventNumber / n) * n
	offset := (eventNumber * 953) % n
	return base + offset
}

func (gc *GeneratorConfig) NextEventNumberForWatermark(numEvents uint64) uint64 {
	n := gc.Configuration.OutOfOrderGroupSize
	eventNumber := gc.NextEventNumber(numEvents)
	return (eventNumber / n) * n
}

func (gc *GeneratorConfig) TimestampForEvent(eventNumber uint64) int64 {
	return gc.BaseTime + int64(float64(eventNumber)*gc.InterEventDelayUs[0])/int64(1000)
}
