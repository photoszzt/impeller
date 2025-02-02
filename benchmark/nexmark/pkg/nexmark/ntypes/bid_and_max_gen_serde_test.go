package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeBidAndMax(t *testing.T) {
	faker := gofakeit.New(3)
	v := BidAndMax{}
	jsonSerdeG := BidAndMaxJSONSerdeG{}
	jsonSerde := BidAndMaxJSONSerde{}
	msgSerdeG := BidAndMaxMsgpSerdeG{}
	msgSerde := BidAndMaxMsgpSerde{}
	commtypes.GenTestEncodeDecode[BidAndMax](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[BidAndMax](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[BidAndMax](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[BidAndMax](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeBidAndMax(b *testing.B) {
	faker := gofakeit.New(3)
	var v BidAndMax
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := BidAndMaxMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
