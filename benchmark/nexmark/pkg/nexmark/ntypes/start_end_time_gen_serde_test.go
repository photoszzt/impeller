package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeStartEndTime(t *testing.T) {
	faker := gofakeit.New(3)
	v := StartEndTime{}
	jsonSerdeG := StartEndTimeJSONSerdeG{}
	jsonSerde := StartEndTimeJSONSerde{}
	msgSerdeG := StartEndTimeMsgpSerdeG{}
	msgSerde := StartEndTimeMsgpSerde{}
	commtypes.GenTestEncodeDecode[StartEndTime](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[StartEndTime](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[StartEndTime](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[StartEndTime](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeStartEndTime(b *testing.B) {
	faker := gofakeit.New(3)
	var v StartEndTime
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := StartEndTimeMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
