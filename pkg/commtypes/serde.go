//go:generate stringer -type=SerdeFormat
package commtypes

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sharedlog-stream/pkg/optional"
	"testing"
	"unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/exp/constraints"
	"golang.org/x/xerrors"
)

func Require(old []byte, extra int) []byte {
	l := len(old)
	c := cap(old)
	r := l + extra
	if c >= r {
		return old
	} else if l == 0 {
		return make([]byte, 0, extra)
	}
	// the new size is the greater
	// of double the old capacity
	// and the sum of the old length
	// and the number of new bytes
	// necessary.
	c <<= 1
	if c < r {
		c = r
	}
	n := make([]byte, l, c)
	copy(n, old)
	return n
}

var (
	sizeNot8 = xerrors.New("size of value to deserialized is not 8")
	sizeNot4 = xerrors.New("size of value to deserialized is not 4")
	sizeNot2 = xerrors.New("size of value to deserialized is not 2")
	sizeNot1 = xerrors.New("size of value to deserialized is not 1")
)

type SerdeFormat uint8

const (
	JSON SerdeFormat = 0
	MSGP SerdeFormat = 1
)

type (
	DefaultEncoder   struct{}
	DefaultJSONSerde struct{}
)

func (DefaultJSONSerde) UsedBufferPool() bool { return false }
func (DefaultJSONSerde) Encode(interface{}) ([]byte, error) {
	panic("unimplemented")
}

func (DefaultJSONSerde) Decode([]byte) (interface{}, error) {
	panic("unimplemented")
}

type DefaultMsgpSerde struct{}

func (DefaultMsgpSerde) UsedBufferPool() bool { return true }
func (DefaultMsgpSerde) Encode(interface{}) ([]byte, error) {
	panic("unimplemented")
}

func (DefaultMsgpSerde) Decode([]byte) (interface{}, error) {
	panic("unimplemented")
}

type DefaultJSONSerdeG[V any] struct{}

func (DefaultJSONSerdeG[V]) UsedBufferPool() bool { return false }
func (DefaultJSONSerdeG[V]) Encode(V) ([]byte, error) {
	panic("unimplemented")
}

func (DefaultJSONSerdeG[V]) Decode([]byte) (V, error) {
	panic("unimplemented")
}

type DefaultMsgpSerdeG[V any] struct{}

func (DefaultMsgpSerdeG[V]) UsedBufferPool() bool { return true }
func (DefaultMsgpSerdeG[V]) Encode(V) ([]byte, error) {
	panic("unimplemented")
}

func (DefaultMsgpSerdeG[V]) Decode([]byte) (V, error) {
	panic("unimplemented")
}

type Encoder interface {
	Encode(interface{}) ([]byte, *[]byte, error)
}

type EncoderG[V any] interface {
	Encode(v V) ([]byte, *[]byte, error)
}

type EncoderFunc func(interface{}) ([]byte, error)

func (ef EncoderFunc) Encode(v interface{}) ([]byte, error) {
	return ef(v)
}

type Decoder interface {
	Decode([]byte) (interface{}, error)
}

type DecoderG[V any] interface {
	Decode([]byte) (V, error)
}

type DecoderFunc func([]byte) (interface{}, error)

func (df DecoderFunc) Decode(b []byte) (interface{}, error) {
	return df(b)
}

type Serde interface {
	Encoder
	Decoder
	UsedBufferPool() bool
	fmt.Stringer
}

type SerdeG[V any] interface {
	EncoderG[V]
	DecoderG[V]
	UsedBufferPool() bool
	fmt.Stringer
}

type (
	Float64Encoder struct {
		DefaultEncoder
	}
	Float64EncoderG struct {
		DefaultEncoder
	}
)

var (
	_ = Encoder(Float64Encoder{})
	_ = EncoderG[float64](Float64EncoderG{})
)

func (e Float64Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(float64)
	bits := math.Float64bits(v)
	// b := PopBuffer(8)
	// bs := *b
	// bs = Require(bs[:0], 8)
	bs := make([]byte, 0, 8)
	bs = binary.BigEndian.AppendUint64(bs, bits)
	// return bs, b, nil
	return bs, nil, nil
}

func (e Float64EncoderG) Encode(value float64) ([]byte, *[]byte, error) {
	bits := math.Float64bits(value)
	// b := PopBuffer(8)
	// bs := *b
	// bs = Require(bs[:0], 8)
	bs := make([]byte, 0, 8)
	bs = binary.BigEndian.AppendUint64(bs, bits)
	return bs, nil, nil
}

type (
	Float64Decoder  struct{}
	Float64DecoderG struct{}
)

var (
	_ = Decoder(&Float64Decoder{})
	_ = DecoderG[float64](&Float64DecoderG{})
)

func (e Float64Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 8 {
		return nil, sizeNot8
	}
	bits := binary.BigEndian.Uint64(value)
	return math.Float64frombits(bits), nil
}

func (e Float64DecoderG) Decode(value []byte) (float64, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 8 {
		return 0, sizeNot8
	}
	bits := binary.BigEndian.Uint64(value)
	return math.Float64frombits(bits), nil
}

type Float64Serde struct {
	Float64Encoder
	Float64Decoder
}

type Float64SerdeG struct {
	Float64EncoderG
	Float64DecoderG
}

func (Float64SerdeG) UsedBufferPool() bool { return false }
func (Float64Serde) UsedBufferPool() bool  { return false }
func (Float64SerdeG) String() string       { return "Float64SerdeG" }
func (Float64Serde) String() string        { return "Float64Serde" }

var (
	_ = Serde(Float64Serde{})
	_ = SerdeG[float64](Float64SerdeG{})
)

type (
	Float32Encoder struct {
		DefaultEncoder
	}
	Float32EncoderG struct {
		DefaultEncoder
	}
)

var (
	_ = Encoder(Float32Encoder{})
	_ = EncoderG[float32](Float32EncoderG{})
)

func (e Float32Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(float32)
	bits := math.Float32bits(v)
	b := PopBuffer(8)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, bits)
	return bs, b, nil
}

func (e Float32EncoderG) Encode(value float32) ([]byte, *[]byte, error) {
	bits := math.Float32bits(value)
	b := PopBuffer(8)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, bits)
	return bs, b, nil
}

type (
	Float32Decoder  struct{}
	Float32DecoderG struct{}
)

var (
	_ = Decoder(Float32Decoder{})
	_ = DecoderG[float32](Float32DecoderG{})
)

func (e Float32Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 4 {
		return nil, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return math.Float32frombits(bits), nil
}

func (e Float32DecoderG) Decode(value []byte) (float32, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 4 {
		return 0, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return math.Float32frombits(bits), nil
}

type Float32Serde struct {
	Float32Encoder
	Float32Decoder
}

type Float32SerdeG struct {
	Float32EncoderG
	Float32DecoderG
}

var (
	_ = Serde(Float32Serde{})
	_ = SerdeG[float32](Float32SerdeG{})
)

func (Float32SerdeG) UsedBufferPool() bool { return true }
func (Float32Serde) UsedBufferPool() bool  { return true }
func (Float32SerdeG) String() string       { return "Float32SerdeG" }
func (Float32Serde) String() string        { return "Float32Serde" }

type (
	Uint64Encoder struct {
		DefaultEncoder
	}
	Uint64EncoderG struct {
		DefaultEncoder
	}
)

var (
	_ = Encoder(Uint64Encoder{})
	_ = EncoderG[uint64](Uint64EncoderG{})
)

func (e Uint64Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(uint64)
	// b := PopBuffer(8)
	// bs := *b
	// bs = Require(bs[:0], 8)
	bs := make([]byte, 0, 8)
	bs = binary.BigEndian.AppendUint64(bs, v)
	// return bs, b, nil
	return bs, nil, nil
}

func (e Uint64EncoderG) Encode(value uint64) ([]byte, *[]byte, error) {
	// b := PopBuffer(8)
	// bs := *b
	// bs = Require(bs[:0], 8)
	bs := make([]byte, 0, 8)
	bs = binary.BigEndian.AppendUint64(bs, value)
	// return bs, b, nil
	return bs, nil, nil
}

type (
	Uint64Decoder  struct{}
	Uint64DecoderG struct{}
)

var (
	_ = Decoder(Uint64Decoder{})
	_ = DecoderG[uint64](Uint64DecoderG{})
)

func (d Uint64Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 8 {
		return nil, sizeNot8
	}
	bits := binary.BigEndian.Uint64(value)
	return bits, nil
}

func (d Uint64DecoderG) Decode(value []byte) (uint64, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 8 {
		return 0, sizeNot8
	}
	bits := binary.BigEndian.Uint64(value)
	return bits, nil
}

type Uint64Serde struct {
	Uint64Encoder
	Uint64Decoder
}

type Uint64SerdeG struct {
	Uint64EncoderG
	Uint64DecoderG
}

func (Uint64SerdeG) UsedBufferPool() bool { return false }
func (Uint64Serde) UsedBufferPool() bool  { return false }
func (Uint64SerdeG) String() string       { return "Uint64SerdeG" }
func (Uint64Serde) String() string        { return "Uint64Serde" }

var (
	_ = Serde(Uint64Serde{})
	_ = SerdeG[uint64](Uint64SerdeG{})
)

type (
	Int64Encoder struct {
		DefaultEncoder
	}
	Int64EncoderG struct {
		DefaultEncoder
	}
)

var (
	_ = Encoder(Int64Encoder{})
	_ = EncoderG[int64](Int64EncoderG{})
)

func (e Int64Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(int64)
	b := PopBuffer(8)
	bs := *b
	bs = Require(bs[:0], 8)
	bs = binary.BigEndian.AppendUint64(bs, uint64(v))
	return bs, b, nil
}

func (e Int64EncoderG) Encode(value int64) ([]byte, *[]byte, error) {
	b := PopBuffer(8)
	bs := *b
	bs = Require(bs[:0], 8)
	bs = binary.BigEndian.AppendUint64(bs, uint64(value))
	return bs, b, nil
}

type (
	Int64Decoder  struct{}
	Int64DecoderG struct{}
)

var (
	_ = Decoder(Int64Decoder{})
	_ = DecoderG[int64](Int64DecoderG{})
)

func (d Int64Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 8 {
		return nil, sizeNot8
	}
	bits := binary.BigEndian.Uint64(value)
	return int64(bits), nil
}

func (d Int64DecoderG) Decode(value []byte) (int64, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 8 {
		return 0, sizeNot8
	}
	bits := binary.BigEndian.Uint64(value)
	return int64(bits), nil
}

type Int64Serde struct {
	Int64Encoder
	Int64Decoder
}

type Int64SerdeG struct {
	Int64EncoderG
	Int64DecoderG
}

func (Int64SerdeG) UsedBufferPool() bool { return true }
func (Int64Serde) UsedBufferPool() bool  { return true }
func (Int64SerdeG) String() string       { return "Int64SerdeG" }
func (Int64Serde) String() string        { return "Int64Serde" }

var (
	_ = Serde(Int64Serde{})
	_ = SerdeG[int64](Int64SerdeG{})
)

type (
	Uint32Encoder struct {
		DefaultEncoder
	}
	Uint32EncoderG struct {
		DefaultEncoder
	}
)

var (
	_ = Encoder(&Uint32Encoder{})
	_ = EncoderG[uint32](&Uint32EncoderG{})
)

func (e Uint32Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(uint32)
	b := PopBuffer(4)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, v)
	return bs, b, nil
}

func (e Uint32EncoderG) Encode(value uint32) ([]byte, *[]byte, error) {
	b := PopBuffer(4)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, value)
	return bs, b, nil
}

type (
	Uint32Decoder  struct{}
	Uint32DecoderG struct{}
)

var (
	_ = Decoder(Uint32Decoder{})
	_ = DecoderG[uint32](Uint32DecoderG{})
)

func (e Uint32Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 4 {
		return nil, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return uint32(bits), nil
}

func (e Uint32DecoderG) Decode(value []byte) (uint32, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 4 {
		return 0, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return uint32(bits), nil
}

type Uint32Serde struct {
	Uint32Encoder
	Uint32Decoder
}

type Uint32SerdeG struct {
	Uint32EncoderG
	Uint32DecoderG
}

var (
	_ = Serde(Uint32Serde{})
	_ = SerdeG[uint32](Uint32SerdeG{})
)

func (Uint32SerdeG) UsedBufferPool() bool { return true }
func (Uint32Serde) UsedBufferPool() bool  { return true }
func (Uint32SerdeG) String() string       { return "Uint32SerdeG" }
func (Uint32Serde) String() string        { return "Uint32Serde" }

type (
	Int32Encoder struct {
		DefaultEncoder
	}
	Int32EncoderG struct {
		DefaultEncoder
	}
)

var (
	_ = Encoder(Int32Encoder{})
	_ = EncoderG[int32](Int32EncoderG{})
)

func (e Int32Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(int32)
	b := PopBuffer(4)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, uint32(v))
	return bs, b, nil
}

func (e Int32EncoderG) Encode(value int32) ([]byte, *[]byte, error) {
	b := PopBuffer(4)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, uint32(value))
	return bs, b, nil
}

type (
	Int32Decoder  struct{}
	Int32DecoderG struct{}
)

var (
	_ = Decoder(Int32Decoder{})
	_ = DecoderG[int32](Int32DecoderG{})
)

func (e Int32Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 4 {
		return nil, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return int32(bits), nil
}

func (e Int32DecoderG) Decode(value []byte) (int32, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 4 {
		return 0, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return int32(bits), nil
}

type Int32Serde struct {
	Int32Encoder
	Int32Decoder
}

type Int32SerdeG struct {
	Int32EncoderG
	Int32DecoderG
}

var (
	_ = Serde(Int32Serde{})
	_ = SerdeG[int32](Int32SerdeG{})
)

func (Int32SerdeG) UsedBufferPool() bool { return true }
func (Int32Serde) UsedBufferPool() bool  { return true }
func (Int32SerdeG) String() string       { return "Int32SerdeG" }
func (Int32Serde) String() string        { return "Int32Serde" }

type (
	IntSerde struct {
		DefaultEncoder
	}
	IntSerdeG struct {
		DefaultEncoder
	}
)

var (
	_ = SerdeG[int](IntSerdeG{})
	_ = Serde(IntSerde{})
)

func (IntSerdeG) UsedBufferPool() bool { return true }
func (IntSerde) UsedBufferPool() bool  { return true }
func (IntSerdeG) String() string       { return "IntSerdeG" }
func (IntSerde) String() string        { return "IntSerde" }

func (s IntSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(int)
	b := PopBuffer(4)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, uint32(v))
	return bs, b, nil
}

func (s IntSerdeG) Encode(value int) ([]byte, *[]byte, error) {
	b := PopBuffer(4)
	bs := *b
	bs = Require(bs[:0], 4)
	bs = binary.BigEndian.AppendUint32(bs, uint32(value))
	return bs, b, nil
}

func (s IntSerde) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 4 {
		return nil, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return int(bits), nil
}

func (s IntSerdeG) Decode(value []byte) (int, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 4 {
		return 0, sizeNot4
	}
	bits := binary.BigEndian.Uint32(value)
	return int(bits), nil
}

type (
	Uint16Encoder  struct{}
	Uint16EncoderG struct{}
)

var (
	_ = Encoder(Uint16Encoder{})
	_ = EncoderG[uint16](Uint16EncoderG{})
)

func (e Uint16Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(uint16)
	bs := make([]byte, 0, 2)
	bs = binary.BigEndian.AppendUint16(bs, v)
	return bs, nil, nil
}

func (e Uint16EncoderG) Encode(value uint16) ([]byte, *[]byte, error) {
	bs := make([]byte, 0, 2)
	bs = binary.BigEndian.AppendUint16(bs, value)
	return bs, nil, nil
}

type (
	Uint16Decoder  struct{}
	Uint16DecoderG struct{}
)

var (
	_ = Decoder(Uint16Decoder{})
	_ = DecoderG[uint16](Uint16DecoderG{})
)

func (e Uint16Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 2 {
		return nil, sizeNot2
	}
	bits := binary.BigEndian.Uint16(value)
	return bits, nil
}

func (e Uint16DecoderG) Decode(value []byte) (uint16, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 2 {
		return 0, sizeNot2
	}
	bits := binary.BigEndian.Uint16(value)
	return bits, nil
}

type Uint16Serde struct {
	Uint16Encoder
	Uint16Decoder
}

type Uint16SerdeG struct {
	Uint16EncoderG
	Uint16DecoderG
}

var (
	_ = Serde(Uint16Serde{})
	_ = SerdeG[uint16](Uint16SerdeG{})
)

func (Uint16SerdeG) UsedBufferPool() bool { return false }
func (Uint16Serde) UsedBufferPool() bool  { return false }
func (Uint16SerdeG) String() string       { return "Uint16SerdeG" }
func (Uint16Serde) String() string        { return "Uint16Serde" }

type (
	Int16Encoder  struct{}
	Int16EncoderG struct{}
)

var (
	_ = Encoder(Int16Encoder{})
	_ = EncoderG[int16](Int16EncoderG{})
)

func (e Int16Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(int16)
	bs := make([]byte, 0, 2)
	bs = binary.BigEndian.AppendUint16(bs, uint16(v))
	return bs, nil, nil
}

func (e Int16EncoderG) Encode(value int16) ([]byte, *[]byte, error) {
	bs := make([]byte, 0, 2)
	bs = binary.BigEndian.AppendUint16(bs, uint16(value))
	return bs, nil, nil
}

type (
	Int16Decoder  struct{}
	Int16DecoderG struct{}
)

var (
	_ = Decoder(Int16Decoder{})
	_ = DecoderG[int16](Int16DecoderG{})
)

func (e Int16Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 2 {
		return nil, sizeNot2
	}
	bits := binary.BigEndian.Uint16(value)
	return int16(bits), nil
}

func (e Int16DecoderG) Decode(value []byte) (int16, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 2 {
		return 0, sizeNot2
	}
	bits := binary.BigEndian.Uint16(value)
	return int16(bits), nil
}

type Int16Serde struct {
	Int16Encoder
	Int16Decoder
}

type Int16SerdeG struct {
	Int16EncoderG
	Int16DecoderG
}

func (Int16SerdeG) UsedBufferPool() bool { return false }
func (Int16Serde) UsedBufferPool() bool  { return false }
func (Int16SerdeG) String() string       { return "Int16SerdeG" }
func (Int16Serde) String() string        { return "Int16Serde" }

var (
	_ = Serde(Int16Serde{})
	_ = SerdeG[int16](Int16SerdeG{})
)

type (
	Uint8Encoder  struct{}
	Uint8EncoderG struct{}
)

var (
	_ = Encoder(Uint8Encoder{})
	_ = EncoderG[uint8](Uint8EncoderG{})
)

func (e Uint8Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(uint8)
	bs := make([]byte, 1)
	bs[0] = v
	return bs, nil, nil
}

func (e Uint8EncoderG) Encode(value uint8) ([]byte, *[]byte, error) {
	bs := make([]byte, 1)
	bs[0] = value
	return bs, nil, nil
}

type (
	Uint8Decoder  struct{}
	Uint8DecoderG struct{}
)

var (
	_ = Decoder(Uint8Decoder{})
	_ = DecoderG[uint8](Uint8DecoderG{})
)

func (e Uint8Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 1 {
		return nil, sizeNot1
	}
	return value[0], nil
}

func (e Uint8DecoderG) Decode(value []byte) (uint8, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 1 {
		return 0, sizeNot1
	}
	return value[0], nil
}

type Uint8Serde struct {
	Uint8Encoder
	Uint8Decoder
}

type Uint8SerdeG struct {
	Uint8EncoderG
	Uint8DecoderG
}

var (
	_ = Serde(Uint8Serde{})
	_ = SerdeG[uint8](Uint8SerdeG{})
)

func (Uint8SerdeG) UsedBufferPool() bool { return false }
func (Uint8Serde) UsedBufferPool() bool  { return false }
func (Uint8SerdeG) String() string       { return "Uint8SerdeG" }
func (Uint8Serde) String() string        { return "Uint8Serde" }

type (
	Int8Encoder  struct{}
	Int8EncoderG struct{}
)

var (
	_ = Encoder(&Int8Encoder{})
	_ = EncoderG[int8](&Int8EncoderG{})
)

func (e Int8Encoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(int8)
	bs := make([]byte, 1)
	bs[0] = uint8(v)
	return bs, nil, nil
}

func (e Int8EncoderG) Encode(value int8) ([]byte, *[]byte, error) {
	bs := make([]byte, 1)
	bs[0] = uint8(value)
	return bs, nil, nil
}

type (
	Int8Decoder  struct{}
	Int8DecoderG struct{}
)

var (
	_ = Decoder(&Int8Decoder{})
	_ = DecoderG[int8](&Int8DecoderG{})
)

func (e Int8Decoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	if len(value) != 1 {
		return nil, sizeNot1
	}
	return int8(value[0]), nil
}

func (e Int8DecoderG) Decode(value []byte) (int8, error) {
	if value == nil {
		return 0, nil
	}
	if len(value) != 1 {
		return 0, sizeNot1
	}
	return int8(value[0]), nil
}

type Int8Serde struct {
	Int8Encoder
	Int8Decoder
}

type Int8SerdeG struct {
	Int8EncoderG
	Int8DecoderG
}

var (
	_ = Serde(Int8Serde{})
	_ = SerdeG[int8](Int8SerdeG{})
)

func (Int8SerdeG) UsedBufferPool() bool { return false }
func (Int8Serde) UsedBufferPool() bool  { return false }
func (Int8SerdeG) String() string       { return "Int8SerdeG" }
func (Int8Serde) String() string        { return "Int8Serde" }

// From gvisor: https://cs.opensource.google/gvisor/gvisor/+/master:pkg/gohacks/string_go120_unsafe.go;l=23-39
// ImmutableBytesFromString is equivalent to []byte(s), except that it uses the
// same memory backing s instead of making a heap-allocated copy. This is only
// valid if the returned slice is never mutated.
func ImmutableBytesFromString(s string) []byte {
	b := unsafe.StringData(s)
	return unsafe.Slice(b, len(s))
}

// StringFromImmutableBytes is equivalent to string(bs), except that it uses
// the same memory backing bs instead of making a heap-allocated copy. This is
// only valid if bs is never mutated after StringFromImmutableBytes returns.
func StringFromImmutableBytes(bs []byte) string {
	if len(bs) == 0 {
		return ""
	}
	return unsafe.String(&bs[0], len(bs))
}

type (
	StringEncoder  struct{}
	StringEncoderG struct{}
)

var (
	_ = Encoder(&StringEncoder{})
	_ = EncoderG[string](&StringEncoderG{})
)

func (e StringEncoder) Encode(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	v := value.(string)
	return ImmutableBytesFromString(v), nil, nil
}

func (e StringEncoderG) Encode(value string) ([]byte, *[]byte, error) {
	return ImmutableBytesFromString(value), nil, nil
}

type (
	StringDecoder  struct{}
	StringDecoderG struct{}
)

var (
	_ = Decoder(&StringDecoder{})
	_ = DecoderG[string](&StringDecoderG{})
)

func (d StringDecoder) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	return StringFromImmutableBytes(value), nil
}

func (d StringDecoderG) Decode(value []byte) (string, error) {
	if value == nil {
		return "", nil
	}
	return StringFromImmutableBytes(value), nil
}

type StringSerde struct {
	StringEncoder
	StringDecoder
}

type StringSerdeG struct {
	StringEncoderG
	StringDecoderG
}

var (
	_ = Serde(StringSerde{})
	_ = SerdeG[string](StringSerdeG{})
)

func (StringSerdeG) UsedBufferPool() bool { return false }
func (StringSerde) UsedBufferPool() bool  { return false }
func (StringSerdeG) String() string       { return "StringSerdeG" }
func (StringSerde) String() string        { return "StringSerde" }

type OptionalValSerde[V any] struct {
	valSerde SerdeG[V]
}

func NewOptionalValSerde[V any](valSerde SerdeG[V]) SerdeG[optional.Option[V]] {
	return OptionalValSerde[V]{valSerde: valSerde}
}

func (s OptionalValSerde[V]) UsedBufferPool() bool { return s.valSerde.UsedBufferPool() }
func (s OptionalValSerde[V]) String() string {
	return fmt.Sprintf("OptionalValSerde{val: %s}", s.valSerde.String())
}

var _ = SerdeG[optional.Option[int]](OptionalValSerde[int]{})

func (s OptionalValSerde[V]) Encode(value optional.Option[V]) ([]byte, *[]byte, error) {
	v, ok := value.Take()
	if ok {
		return s.valSerde.Encode(v)
	} else {
		return nil, nil, nil
	}
}

func (s OptionalValSerde[V]) Decode(val []byte) (optional.Option[V], error) {
	if val == nil {
		return optional.None[V](), nil
	} else {
		v, err := s.valSerde.Decode(val)
		if err != nil {
			return optional.None[V](), err
		}
		return optional.Some(v), nil
	}
}

const float64EqualityThreshold = 1e-9

func almostEqual[V constraints.Float](a, b V) bool {
	return math.Abs(float64(a-b)) <= float64EqualityThreshold
}

func GenTestEncodeDecodeFloat[V constraints.Float](v V, t *testing.T, serdeG SerdeG[V], serde Serde) {
	bts, buf, err := serdeG.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := serdeG.Decode(bts)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(v, ret) {
		t.Fatal("encode and decode doesn't give same value")
	}
	if serdeG.UsedBufferPool() {
		*buf = bts
		PushBuffer(buf)
	}

	bts, buf, err = serde.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	r, err := serde.Decode(bts)
	if err != nil {
		t.Fatal(err)
	}
	if !almostEqual(v, r.(V)) {
		t.Fatal("encode and decode doesn't give same value")
	}
	if serde.UsedBufferPool() {
		*buf = bts
		PushBuffer(buf)
	}
}

func GenBenchmarkPooledSerde[V any](v V, b *testing.B, serdeG SerdeG[V]) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret, buf, err := serdeG.Encode(v)
		if err != nil {
			b.Fatal(err)
		}
		*buf = ret
		PushBuffer(buf)
	}
}

func GenTestEncodeDecode[V any](v V, t *testing.T, serdeG SerdeG[V], serde Serde) {
	vl := reflect.ValueOf(v)
	var opts cmp.Options
	if vl.Comparable() {
		opts = append(opts,
			cmpopts.EquateComparable(),
		)
	}
	if vl.Kind() == reflect.Struct {
		opts = append(opts, cmpopts.IgnoreUnexported(v), cmpopts.EquateEmpty())
	}

	bts, buf, err := serdeG.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := serdeG.Decode(bts)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(v, ret, opts...) {
		t.Logf("serdeG input %v, decoded %v\n", v, ret)
		t.Fatal("encode and decode doesn't give same value")
	}
	if serdeG.UsedBufferPool() && buf != nil {
		*buf = bts
		PushBuffer(buf)
	}

	bts, buf, err = serde.Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	r, err := serde.Decode(bts)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(v, r.(V), opts...) {
		t.Logf("serde input %v, decoded %v\n", v, ret)
		t.Fatal("encode and decode doesn't give same value")
	}
	if serde.UsedBufferPool() && buf != nil {
		*buf = bts
		PushBuffer(buf)
	}
}
