package commtypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

func TestSerdeTableSnapshots(t *testing.T) {
	faker := gofakeit.New(3)
	v := TableSnapshots{}
	jsonSerdeG := TableSnapshotsJSONSerdeG{}
	jsonSerde := TableSnapshotsJSONSerde{}
	msgSerdeG := TableSnapshotsMsgpSerdeG{}
	msgSerde := TableSnapshotsMsgpSerde{}
	GenTestEncodeDecode[TableSnapshots](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[TableSnapshots](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecode[TableSnapshots](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[TableSnapshots](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeTableSnapshots(b *testing.B) {
	faker := gofakeit.New(3)
	var v TableSnapshots
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := TableSnapshotsMsgpSerdeG{}
	GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
