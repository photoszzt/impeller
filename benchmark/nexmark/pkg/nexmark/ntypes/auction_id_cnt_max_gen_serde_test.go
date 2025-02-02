package ntypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"sharedlog-stream/pkg/commtypes"
)

func TestSerdeAuctionIdCntMax(t *testing.T) {
	faker := gofakeit.New(3)
	v := AuctionIdCntMax{}
	jsonSerdeG := AuctionIdCntMaxJSONSerdeG{}
	jsonSerde := AuctionIdCntMaxJSONSerde{}
	msgSerdeG := AuctionIdCntMaxMsgpSerdeG{}
	msgSerde := AuctionIdCntMaxMsgpSerde{}
	commtypes.GenTestEncodeDecode[AuctionIdCntMax](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdCntMax](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	commtypes.GenTestEncodeDecode[AuctionIdCntMax](v, t, jsonSerdeG, jsonSerde)
	commtypes.GenTestEncodeDecode[AuctionIdCntMax](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeAuctionIdCntMax(b *testing.B) {
	faker := gofakeit.New(3)
	var v AuctionIdCntMax
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := AuctionIdCntMaxMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
