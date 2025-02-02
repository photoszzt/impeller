package txn_data

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeTxnMetadata(t *testing.T) {
	faker := gofakeit.New(3)
	v := TxnMetadata{}
	jsonSerdeG := TxnMetadataJSONSerdeG{}
	jsonSerde := TxnMetadataJSONSerde{}
	msgSerdeG := TxnMetadataMsgpSerdeG{}
	msgSerde := TxnMetadataMsgpSerde{}
	commtypes.GenTestEncodeDecode[TxnMetadata](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[TxnMetadata](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[TxnMetadata](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[TxnMetadata](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeTxnMetadata(b *testing.B) {
	faker := gofakeit.New(3)
	var v TxnMetadata
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := TxnMetadataMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
