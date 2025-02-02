package datatype

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdePayloadTs(t *testing.T) {
	faker := gofakeit.New(3)
	v := PayloadTs{}
	jsonSerdeG := PayloadTsJSONSerdeG{}
	jsonSerde := PayloadTsJSONSerde{}
	msgSerdeG := PayloadTsMsgpSerdeG{}
	msgSerde := PayloadTsMsgpSerde{}
	commtypes.GenTestEncodeDecode[PayloadTs](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[PayloadTs](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[PayloadTs](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[PayloadTs](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdePayloadTs(b *testing.B) {
	faker := gofakeit.New(3)
	var v PayloadTs
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := PayloadTsMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
