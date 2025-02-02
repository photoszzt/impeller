package remote_txn_rpc

import (
	"reflect"
	"sharedlog-stream/pkg/commtypes"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func GenTestEncodeDecodeRTxnReply(v *RTxnReply, t *testing.T, serdeG commtypes.SerdeG[*RTxnReply], serde commtypes.Serde) {
	vl := reflect.ValueOf(v)
	var opts cmp.Options
	if vl.Comparable() {
		opts = append(opts,
			cmpopts.EquateComparable(),
		)
	}
	opts = append(opts, cmpopts.IgnoreUnexported(RTxnReply{}))

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

func TestSerdeRTxnReply(t *testing.T) {
	faker := gofakeit.New(3)
	v := &RTxnReply{}
	jsonSerdeG := RTxnReplyJSONSerdeG{}
	jsonSerde := RTxnReplyJSONSerde{}
	msgSerdeG := RTxnReplyMsgpSerdeG{}
	msgSerde := RTxnReplyMsgpSerde{}
	GenTestEncodeDecodeRTxnReply(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecodeRTxnReply(v, t, msgSerdeG, msgSerde)

	err := faker.Struct(v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecodeRTxnReply(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecodeRTxnReply(v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeRTxnReply(b *testing.B) {
	faker := gofakeit.New(3)
	var v RTxnReply
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := RTxnReplyMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(&v, b, msgSerdeG)
}
