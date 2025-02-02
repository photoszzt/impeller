package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeAuctionIdCount(t *testing.T) {
	faker := gofakeit.New(3)
	v := AuctionIdCount{}
	jsonSerdeG := AuctionIdCountJSONSerdeG{}
	jsonSerde := AuctionIdCountJSONSerde{}
	msgSerdeG := AuctionIdCountMsgpSerdeG{}
	msgSerde := AuctionIdCountMsgpSerde{}
	commtypes.GenTestEncodeDecode[AuctionIdCount](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdCount](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[AuctionIdCount](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdCount](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeAuctionIdCount(b *testing.B) {
	faker := gofakeit.New(3)
	var v AuctionIdCount
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := AuctionIdCountMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
