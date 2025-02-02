package ntypes

import (
	"reflect"
	"sharedlog-stream/pkg/commtypes"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func GenTestEncodeDecodeAuctionBid(v *AuctionBid, t *testing.T, serdeG commtypes.SerdeG[*AuctionBid], serde commtypes.Serde) {
	vl := reflect.ValueOf(v)
	var opts cmp.Options
	if vl.Comparable() {
		opts = append(opts,
			cmpopts.EquateComparable(),
		)
	}
	opts = append(opts, cmpopts.IgnoreUnexported(AuctionBid{}))

	bts, buf, err := serdeG.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := serdeG.Decode(bts)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(v, ret, opts...) {
		t.Fatal("encode and decode doesn't give same value")
	}
	if serdeG.UsedBufferPool() {
		*buf = bts
		commtypes.PushBuffer(buf)
	}

	bts, buf, err = serde.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	r, err := serde.Decode(bts)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(v, r, opts...) {
		t.Fatal("encode and decode doesn't give same value")
	}
	if serde.UsedBufferPool() {
		*buf = bts
		commtypes.PushBuffer(buf)
	}
}

func TestSerdeAuctionBid(t *testing.T) {
	faker := gofakeit.New(3)
	v := &AuctionBid{}
	jsonSerdeG := AuctionBidJSONSerdeG{}
	jsonSerde := AuctionBidJSONSerde{}
	msgSerdeG := AuctionBidMsgpSerdeG{}
	msgSerde := AuctionBidMsgpSerde{}
	GenTestEncodeDecodeAuctionBid(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecodeAuctionBid(v, t, msgSerdeG, msgSerde)

	err := faker.Struct(v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecodeAuctionBid(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecodeAuctionBid(v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeAuctionBid(b *testing.B) {
	faker := gofakeit.New(3)
	var v AuctionBid
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := AuctionBidMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(&v, b, msgSerdeG)
}
