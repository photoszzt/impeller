package commtypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

func TestSerdeOffsetMarker(t *testing.T) {
	faker := gofakeit.New(3)
	v := OffsetMarker{}
	jsonSerdeG := OffsetMarkerJSONSerdeG{}
	jsonSerde := OffsetMarkerJSONSerde{}
	msgSerdeG := OffsetMarkerMsgpSerdeG{}
	msgSerde := OffsetMarkerMsgpSerde{}
	GenTestEncodeDecode[OffsetMarker](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[OffsetMarker](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecode[OffsetMarker](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[OffsetMarker](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeOffsetMarker(b *testing.B) {
	faker := gofakeit.New(3)
	var v OffsetMarker
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := OffsetMarkerMsgpSerdeG{}
	GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
