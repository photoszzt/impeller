package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeNameCityStateId(t *testing.T) {
	faker := gofakeit.New(3)
	v := NameCityStateId{}
	jsonSerdeG := NameCityStateIdJSONSerdeG{}
	jsonSerde := NameCityStateIdJSONSerde{}
	msgSerdeG := NameCityStateIdMsgpSerdeG{}
	msgSerde := NameCityStateIdMsgpSerde{}
	commtypes.GenTestEncodeDecode[NameCityStateId](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[NameCityStateId](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[NameCityStateId](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[NameCityStateId](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeNameCityStateId(b *testing.B) {
	faker := gofakeit.New(3)
	var v NameCityStateId
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := NameCityStateIdMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
