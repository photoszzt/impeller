package commtypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

func TestSerdeEpochMarker(t *testing.T) {
	faker := gofakeit.New(3)
	v := EpochMarker{}
	jsonSerdeG := EpochMarkerJSONSerdeG{}
	jsonSerde := EpochMarkerJSONSerde{}
	msgSerdeG := EpochMarkerMsgpSerdeG{}
	msgSerde := EpochMarkerMsgpSerde{}
	GenTestEncodeDecode[EpochMarker](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[EpochMarker](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecode[EpochMarker](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[EpochMarker](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeEpochMarker(b *testing.B) {
	faker := gofakeit.New(3)
	var v EpochMarker
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := EpochMarkerMsgpSerdeG{}
	GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
