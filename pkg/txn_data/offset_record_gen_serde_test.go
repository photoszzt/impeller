package txn_data

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeOffsetRecord(t *testing.T) {
	faker := gofakeit.New(3)
	v := OffsetRecord{}
	jsonSerdeG := OffsetRecordJSONSerdeG{}
	jsonSerde := OffsetRecordJSONSerde{}
	msgSerdeG := OffsetRecordMsgpSerdeG{}
	msgSerde := OffsetRecordMsgpSerde{}
	commtypes.GenTestEncodeDecode[OffsetRecord](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[OffsetRecord](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[OffsetRecord](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[OffsetRecord](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeOffsetRecord(b *testing.B) {
	faker := gofakeit.New(3)
	var v OffsetRecord
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := OffsetRecordMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
