package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeBidPrice(t *testing.T) {
	faker := gofakeit.New(3)
	v := BidPrice{}
	jsonSerdeG := BidPriceJSONSerdeG{}
	jsonSerde := BidPriceJSONSerde{}
	msgSerdeG := BidPriceMsgpSerdeG{}
	msgSerde := BidPriceMsgpSerde{}
	commtypes.GenTestEncodeDecode[BidPrice](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[BidPrice](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[BidPrice](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[BidPrice](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeBidPrice(b *testing.B) {
	faker := gofakeit.New(3)
	var v BidPrice
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := BidPriceMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
