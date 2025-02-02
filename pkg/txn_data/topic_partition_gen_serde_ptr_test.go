package txn_data

import (
	"reflect"
	"sharedlog-stream/pkg/commtypes"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func GenTestEncodeDecodeTopicPartition(v *TopicPartition, t *testing.T, serdeG commtypes.SerdeG[*TopicPartition], serde commtypes.Serde) {
	vl := reflect.ValueOf(v)
	var opts cmp.Options
	if vl.Comparable() {
		opts = append(opts,
			cmpopts.EquateComparable(),
		)
	}
	opts = append(opts, cmpopts.IgnoreUnexported(TopicPartition{}))

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

func TestSerdeTopicPartition(t *testing.T) {
	faker := gofakeit.New(3)
	v := &TopicPartition{}
	jsonSerdeG := TopicPartitionJSONSerdeG{}
	jsonSerde := TopicPartitionJSONSerde{}
	msgSerdeG := TopicPartitionMsgpSerdeG{}
	msgSerde := TopicPartitionMsgpSerde{}
	GenTestEncodeDecodeTopicPartition(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecodeTopicPartition(v, t, msgSerdeG, msgSerde)

	err := faker.Struct(v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecodeTopicPartition(v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecodeTopicPartition(v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdeTopicPartition(b *testing.B) {
	faker := gofakeit.New(3)
	var v TopicPartition
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := TopicPartitionMsgpSerdeG{}
	commtypes.GenBenchmarkPooledSerde(&v, b, msgSerdeG)
}
