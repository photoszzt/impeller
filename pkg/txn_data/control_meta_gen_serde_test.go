package txn_data

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeControlMetadata(t *testing.T) {
	faker := gofakeit.New(3)
	v := ControlMetadata{}
	jsonSerdeG := ControlMetadataJSONSerdeG{}
	jsonSerde := ControlMetadataJSONSerde{}
	msgSerdeG := ControlMetadataMsgpSerdeG{}
	msgSerde := ControlMetadataMsgpSerde{}
	commtypes.GenTestEncodeDecode[ControlMetadata](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[ControlMetadata](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[ControlMetadata](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[ControlMetadata](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeControlMetadata(b *testing.B) {
	faker := gofakeit.New(3)
	var v ControlMetadata
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := ControlMetadataMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
