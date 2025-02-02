package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdePriceTimeList(t *testing.T) {
	faker := gofakeit.New(3)
	v := PriceTimeList{}
	jsonSerdeG := PriceTimeListJSONSerdeG{}
	jsonSerde := PriceTimeListJSONSerde{}
	msgSerdeG := PriceTimeListMsgpSerdeG{}
	msgSerde := PriceTimeListMsgpSerde{}
	commtypes.GenTestEncodeDecode[PriceTimeList](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[PriceTimeList](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[PriceTimeList](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[PriceTimeList](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdePriceTimeList(b *testing.B) {
	faker := gofakeit.New(3)
	var v PriceTimeList
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := PriceTimeListMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
