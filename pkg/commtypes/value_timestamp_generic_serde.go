package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/optional"
	"sharedlog-stream/pkg/utils"
)

type ValueTimestampJSONSerdeG struct {
	DefaultMsgpSerde
	ValJSONSerde Serde
}

var _ = SerdeG[ValueTimestamp](ValueTimestampJSONSerdeG{})

func (s ValueTimestampJSONSerdeG) String() string {
	return fmt.Sprintf("ValueTimestampJSONSerdeG{val: %s}", s.ValJSONSerde.String())
}

func (s ValueTimestampJSONSerdeG) Encode(value ValueTimestamp) ([]byte, *[]byte, error) {
	vs, vbuf, err := valTsToValueTsSer(value, s.ValJSONSerde)
	defer func() {
		if s.ValJSONSerde.UsedBufferPool() && vbuf != nil {
			*vbuf = vs.ValueSerialized
			PushBuffer(vbuf)
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if vs == nil {
		return nil, nil, nil
	}
	r, err := json.Marshal(vs)
	return r, nil, err
}

func (s ValueTimestampJSONSerdeG) Decode(value []byte) (ValueTimestamp, error) {
	if value == nil {
		return ValueTimestamp{}, nil
	}
	vs := ValueTimestampSerialized{}
	if err := json.Unmarshal(value, &vs); err != nil {
		return ValueTimestamp{}, err
	}
	return valTsSerToValueTs(&vs, s.ValJSONSerde)
}

type ValueTimestampMsgpSerdeG struct {
	DefaultMsgpSerde
	ValMsgpSerde Serde
}

func (s ValueTimestampMsgpSerdeG) String() string {
	return fmt.Sprintf("ValueTimestampMsgpSerdeG{val: %s}", s.ValMsgpSerde.String())
}

func (s ValueTimestampMsgpSerdeG) Encode(value ValueTimestamp) ([]byte, *[]byte, error) {
	vs, vbuf, err := valTsToValueTsSer(value, s.ValMsgpSerde)
	defer func() {
		if s.ValMsgpSerde.UsedBufferPool() && vbuf != nil {
			*vbuf = vs.ValueSerialized
			PushBuffer(vbuf)
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if vs == nil {
		return nil, nil, nil
	}
	b := PopBuffer(vs.Msgsize())
	buf := *b
	r, err := vs.MarshalMsg(buf[:0])
	return r, b, err
}

func (s ValueTimestampMsgpSerdeG) Decode(value []byte) (ValueTimestamp, error) {
	if value == nil {
		return ValueTimestamp{}, nil
	}
	vs := ValueTimestampSerialized{}
	_, err := vs.UnmarshalMsg(value)
	if err != nil {
		return ValueTimestamp{}, err
	}
	return valTsSerToValueTs(&vs, s.ValMsgpSerde)
}

func GetValueTsSerdeG(serdeFormat SerdeFormat, valSerde Serde) (SerdeG[ValueTimestamp], error) {
	if serdeFormat == JSON {
		return ValueTimestampJSONSerdeG{
			ValJSONSerde: valSerde,
		}, nil
	} else if serdeFormat == MSGP {
		return ValueTimestampMsgpSerdeG{
			ValMsgpSerde: valSerde,
		}, nil
	} else {
		return nil, fmt.Errorf("serde format should be either json or msgp; but %v is given", serdeFormat)
	}
}

func CreateValueTimestampOptional[V any](val optional.Option[V], ts int64) optional.Option[ValueTimestamp] {
	v, hasV := val.Take()
	if hasV {
		return optional.Some(ValueTimestamp{
			Value:     v,
			Timestamp: ts,
		})
	} else {
		return optional.None[ValueTimestamp]()
	}
}

func CreateValueTimestampOptionalWithIntrVal(val interface{}, ts int64) optional.Option[ValueTimestamp] {
	if utils.IsNil(val) {
		return optional.None[ValueTimestamp]()
	} else {
		return optional.Some(ValueTimestamp{
			Value:     val,
			Timestamp: ts,
		})
	}
}

func CreateValueTimestampGOptionalWithIntrVal[V any](val interface{}, ts int64) optional.Option[ValueTimestampG[V]] {
	if utils.IsNil(val) {
		return optional.None[ValueTimestampG[V]]()
	} else {
		return optional.Some(ValueTimestampG[V]{
			Value:     val.(V),
			Timestamp: ts,
		})
	}
}

func valTsGToValueTsSer[V any](value ValueTimestampG[V], valSerde SerdeG[V]) (vser *ValueTimestampSerialized, vbuf *[]byte, err error) {
	enc, vbuf, err := valSerde.Encode(value.Value)
	if err != nil {
		return nil, nil, err
	}
	vser = &ValueTimestampSerialized{
		Timestamp:       value.Timestamp,
		ValueSerialized: enc,
	}
	return vser, vbuf, nil
}

func valTsSerToValueTsG[V any](vtsSer *ValueTimestampSerialized, valSerde SerdeG[V]) (ValueTimestampG[V], error) {
	var v V
	var err error
	if vtsSer.ValueSerialized != nil {
		v, err = valSerde.Decode(vtsSer.ValueSerialized)
		if err != nil {
			return ValueTimestampG[V]{}, err
		}
	}
	return ValueTimestampG[V]{
		Timestamp: vtsSer.Timestamp,
		Value:     v,
	}, nil
}

type OptionalValTsG[V any] struct {
	Val       optional.Option[V]
	Timestamp int64
}

type OptionalValTsGSize[V any] struct {
	ValSizeFunc func(V) int64
}

func (s OptionalValTsGSize[V]) SizeOfOptionalValTsG(vOp OptionalValTsG[V]) int64 {
	v, ok := vOp.Val.Take()
	if ok {
		return 8 + s.ValSizeFunc(v)
	} else {
		return 8
	}
}

type ValueTimestampG[V any] struct {
	Value     V
	Timestamp int64
}

type ValueTimestampGSize[V any] struct {
	ValSizeFunc func(V) int64
}

func (s ValueTimestampGSize[V]) SizeOfValueTimestamp(v ValueTimestampG[V]) int64 {
	return 8 + s.ValSizeFunc(v.Value)
}

type ValueTimestampGJSONSerdeG[V any] struct {
	DefaultJSONSerde
	ValJSONSerde SerdeG[V]
}

func (s ValueTimestampGJSONSerdeG[V]) String() string {
	return fmt.Sprintf("ValueTimestampGJSONSerdeG{val: %s}", s.ValJSONSerde.String())
}

func (s ValueTimestampGJSONSerdeG[V]) Encode(value ValueTimestampG[V]) ([]byte, *[]byte, error) {
	vs, vbuf, err := valTsGToValueTsSer(value, s.ValJSONSerde)
	defer func() {
		if s.ValJSONSerde.UsedBufferPool() && vbuf != nil {
			*vbuf = vs.ValueSerialized
			PushBuffer(vbuf)
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if vs == nil {
		return nil, nil, nil
	}
	r, err := json.Marshal(vs)
	return r, nil, err
}

func (s ValueTimestampGJSONSerdeG[V]) Decode(value []byte) (ValueTimestampG[V], error) {
	if value == nil {
		return ValueTimestampG[V]{}, nil
	}
	vs := ValueTimestampSerialized{}
	if err := json.Unmarshal(value, &vs); err != nil {
		return ValueTimestampG[V]{}, err
	}
	return valTsSerToValueTsG(&vs, s.ValJSONSerde)
}

type ValueTimestampGMsgpSerdeG[V any] struct {
	DefaultMsgpSerde
	ValMsgpSerde SerdeG[V]
}

func (s ValueTimestampGMsgpSerdeG[V]) String() string {
	return fmt.Sprintf("ValueTimestampGMsgpSerdeG{val: %s}", s.ValMsgpSerde.String())
}

func (s ValueTimestampGMsgpSerdeG[V]) Encode(value ValueTimestampG[V]) ([]byte, *[]byte, error) {
	vs, vbuf, err := valTsGToValueTsSer(value, s.ValMsgpSerde)
	defer func() {
		if s.ValMsgpSerde.UsedBufferPool() && vbuf != nil {
			*vbuf = vs.ValueSerialized
			PushBuffer(vbuf)
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if vs == nil {
		return nil, nil, nil
	}
	b := PopBuffer(vs.Msgsize())
	buf := *b
	r, err := vs.MarshalMsg(buf[:0])
	return r, b, err
}

func (s ValueTimestampGMsgpSerdeG[V]) Decode(value []byte) (ValueTimestampG[V], error) {
	if value == nil {
		return ValueTimestampG[V]{}, nil
	}
	vs := ValueTimestampSerialized{}
	_, err := vs.UnmarshalMsg(value)
	if err != nil {
		return ValueTimestampG[V]{}, err
	}
	return valTsSerToValueTsG(&vs, s.ValMsgpSerde)
}

func CreateValueTimestampGOptional[V any](val optional.Option[V], ts int64) optional.Option[ValueTimestampG[V]] {
	v, hasV := val.Take()
	if hasV {
		return optional.Some(ValueTimestampG[V]{
			Value:     v,
			Timestamp: ts,
		})
	} else {
		return optional.None[ValueTimestampG[V]]()
	}
}

func GetValueTsGSerdeG[V any](serdeFormat SerdeFormat, valSerde SerdeG[V]) (SerdeG[ValueTimestampG[V]], error) {
	if serdeFormat == JSON {
		return ValueTimestampGJSONSerdeG[V]{
			ValJSONSerde: valSerde,
		}, nil
	} else if serdeFormat == MSGP {
		return ValueTimestampGMsgpSerdeG[V]{
			ValMsgpSerde: valSerde,
		}, nil
	} else {
		return nil, fmt.Errorf("serde format should be either json or msgp; but %v is given", serdeFormat)
	}
}
