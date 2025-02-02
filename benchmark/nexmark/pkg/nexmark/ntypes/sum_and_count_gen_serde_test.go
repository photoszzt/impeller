package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeSumAndCount(t *testing.T) {
	faker := gofakeit.New(3)
	v := SumAndCount{}
	jsonSerdeG := SumAndCountJSONSerdeG{}
	jsonSerde := SumAndCountJSONSerde{}
	msgSerdeG := SumAndCountMsgpSerdeG{}
	msgSerde := SumAndCountMsgpSerde{}
	commtypes.GenTestEncodeDecode[SumAndCount](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[SumAndCount](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[SumAndCount](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[SumAndCount](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeSumAndCount(b *testing.B) {
	faker := gofakeit.New(3)
	var v SumAndCount
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := SumAndCountMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
