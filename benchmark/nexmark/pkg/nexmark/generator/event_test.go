package generator

import (
	"context"
	"sharedlog-stream/benchmark/nexmark/pkg/nexmark/ntypes"
	"sharedlog-stream/pkg/commtypes"
	"testing"
	"time"
)

func BenchmarkAuctionSerde(b *testing.B) {
	nexmarkConfig := ntypes.NewNexMarkConfig()
	generatorConfig := NewGeneratorConfig(nexmarkConfig, time.Now().UnixMilli(), 1,
		uint64(nexmarkConfig.NumEvents), 1)
	g := NewSimpleNexmarkGenerator(generatorConfig, 0)
	adjustedEventTimestamp := g.Config.TimestampForEvent(g.Config.NextAdjustedEventNumber(g.EventsCountSoFar))
	id := g.GetNextEventId()
	auc := NextAuction(g.EventsCountSoFar, id, g.Random, adjustedEventTimestamp, g.Config)
	v := ntypes.NewAuctionEvnet(auc)
	msgSerdeG := ntypes.EventMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}

func BenchmarkPersonSerde(b *testing.B) {
	nexmarkConfig := ntypes.NewNexMarkConfig()
	generatorConfig := NewGeneratorConfig(nexmarkConfig, time.Now().UnixMilli(), 1,
		uint64(nexmarkConfig.NumEvents), 1)
	g := NewSimpleNexmarkGenerator(generatorConfig, 0)
	adjustedEventTimestamp := g.Config.TimestampForEvent(g.Config.NextAdjustedEventNumber(g.EventsCountSoFar))
	id := g.GetNextEventId()
	per := NextPerson(id, g.Random, adjustedEventTimestamp, g.Config)
	v := ntypes.NewPersonEvent(per)
	msgSerdeG := ntypes.EventMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}

func BenchmarkBidSerde(b *testing.B) {
	nexmarkConfig := ntypes.NewNexMarkConfig()
	generatorConfig := NewGeneratorConfig(nexmarkConfig, time.Now().UnixMilli(), 1,
		uint64(nexmarkConfig.NumEvents), 1)
	g := NewSimpleNexmarkGenerator(generatorConfig, 0)
	adjustedEventTimestamp := g.Config.TimestampForEvent(g.Config.NextAdjustedEventNumber(g.EventsCountSoFar))
	id := g.GetNextEventId()
	ctx := context.Background()
	cache := make(map[uint32]*ChannelUrl)
	bid, err := NextBid(ctx, id, g.Random, adjustedEventTimestamp, g.Config, cache)
	if err != nil {
		b.Fatal(err)
	}
	v := ntypes.NewBidEvent(bid)
	msgSerdeG := ntypes.EventMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
