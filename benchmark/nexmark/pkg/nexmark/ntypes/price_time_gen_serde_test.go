package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdePriceTime(t *testing.T) {
	faker := gofakeit.New(3)
	v := PriceTime{}
	jsonSerdeG := PriceTimeJSONSerdeG{}
	jsonSerde := PriceTimeJSONSerde{}
	msgSerdeG := PriceTimeMsgpSerdeG{}
	msgSerde := PriceTimeMsgpSerde{}
	commtypes.GenTestEncodeDecode[PriceTime](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[PriceTime](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[PriceTime](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[PriceTime](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdePriceTime(b *testing.B) {
	faker := gofakeit.New(3)
	var v PriceTime
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := PriceTimeMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
