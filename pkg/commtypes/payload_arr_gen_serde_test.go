package commtypes

import (
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

func TestSerdePayloadArr(t *testing.T) {
	faker := gofakeit.New(3)
	v := PayloadArr{}
	jsonSerdeG := PayloadArrJSONSerdeG{}
	jsonSerde := PayloadArrJSONSerde{}
	msgSerdeG := PayloadArrMsgpSerdeG{}
	msgSerde := PayloadArrMsgpSerde{}
	GenTestEncodeDecode[PayloadArr](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[PayloadArr](v, t, msgSerdeG, msgSerde)
	err := faker.Struct(&v)
	if err != nil {
		t.Fatal(err)
	}
	GenTestEncodeDecode[PayloadArr](v, t, jsonSerdeG, jsonSerde)
	GenTestEncodeDecode[PayloadArr](v, t, msgSerdeG, msgSerde)
}

func BenchmarkSerdePayloadArr(b *testing.B) {
	faker := gofakeit.New(3)
	var v PayloadArr
	err := faker.Struct(&v)
	if err != nil {
		b.Fatal(err)
	}
	msgSerdeG := PayloadArrMsgpSerdeG{}
	GenBenchmarkPooledSerde(v, b, msgSerdeG)
}
