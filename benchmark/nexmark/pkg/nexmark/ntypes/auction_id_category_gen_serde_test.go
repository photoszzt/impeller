package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeAuctionIdCategory(t *testing.T) {
	faker := gofakeit.New(3)
	v := AuctionIdCategory{}
	jsonSerdeG := AuctionIdCategoryJSONSerdeG{}
	jsonSerde := AuctionIdCategoryJSONSerde{}
	msgSerdeG := AuctionIdCategoryMsgpSerdeG{}
	msgSerde := AuctionIdCategoryMsgpSerde{}
	commtypes.GenTestEncodeDecode[AuctionIdCategory](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdCategory](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[AuctionIdCategory](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdCategory](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeAuctionIdCategory(b *testing.B) {
	faker := gofakeit.New(3)
	var v AuctionIdCategory
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := AuctionIdCategoryMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
