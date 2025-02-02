package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeAuctionIdSeller(t *testing.T) {
	faker := gofakeit.New(3)
	v := AuctionIdSeller{}
	jsonSerdeG := AuctionIdSellerJSONSerdeG{}
	jsonSerde := AuctionIdSellerJSONSerde{}
	msgSerdeG := AuctionIdSellerMsgpSerdeG{}
	msgSerde := AuctionIdSellerMsgpSerde{}
	commtypes.GenTestEncodeDecode[AuctionIdSeller](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdSeller](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[AuctionIdSeller](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdSeller](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeAuctionIdSeller(b *testing.B) {
	faker := gofakeit.New(3)
	var v AuctionIdSeller
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := AuctionIdSellerMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
