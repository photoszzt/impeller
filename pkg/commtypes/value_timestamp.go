//go:generate msgp
//msgp:ignore ValueTimestampJSONSerde ValueTimestampMsgpSerde ValueTimestamp
package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/utils"
)

type ValueTimestamp struct {
	Value     interface{}
	Timestamp int64
}

type ValueTimestampSize struct {
	ValSizeFunc func(interface{}) int64
}

func (s ValueTimestampSize) SizeOfValueTimestamp(v ValueTimestamp) int64 {
	return 8 + s.ValSizeFunc(v.Value)
}

var _ = fmt.Stringer(ValueTimestamp{})

func (vts ValueTimestamp) String() string {
	return fmt.Sprintf("ValueTs: {Value: %v, Ts: %d}", vts.Value, vts.Timestamp)
}

func CreateValueTimestamp(val interface{}, ts int64) *ValueTimestamp {
	if utils.IsNil(val) {
		return nil
	} else {
		return &ValueTimestamp{
			Value:     val,
			Timestamp: ts,
		}
	}
}

type ValueTimestampSerialized struct {
	ValueSerialized []byte `json:"vs,omitempty" msg:"vs,omitempty"`
	Timestamp       int64  `json:"ts,omitempty" msg:"ts,omitempty"`
}

func (s *ValueTimestamp) ExtractEventTime() (int64, error) {
	return s.Timestamp, nil
}

func GetValOrNil(valTs *ValueTimestamp) interface{} {
	if valTs == nil {
		return nil
	} else {
		return valTs.Value
	}
}

var _ = EventTimeExtractor(&ValueTimestamp{})

func CastToValTsPtr(value interface{}) *ValueTimestamp {
	v, ok := value.(*ValueTimestamp)
	if !ok {
		vtmp := value.(ValueTimestamp)
		v = &vtmp
	}
	return v
}

func convertToValueTsSer(value interface{}, valSerde Serde) (vser *ValueTimestampSerialized, vbuf *[]byte, err error) {
	if value == nil {
		return nil, nil, nil
	}
	v := CastToValTsPtr(value)
	if v == nil {
		return nil, nil, nil
	}
	var enc []byte
	if !utils.IsNil(v.Value) {
		enc, vbuf, err = valSerde.Encode(v.Value)
		if err != nil {
			return nil, nil, err
		}
	}
	vser = &ValueTimestampSerialized{
		Timestamp:       v.Timestamp,
		ValueSerialized: enc,
	}
	return vser, vbuf, nil
}

func valTsToValueTsSer(value ValueTimestamp, valSerde Serde) (vser *ValueTimestampSerialized, vbuf *[]byte, err error) {
	var enc []byte
	if !utils.IsNil(value.Value) {
		enc, vbuf, err = valSerde.Encode(value.Value)
		if err != nil {
			return nil, nil, err
		}
	}
	vser = &ValueTimestampSerialized{
		Timestamp:       value.Timestamp,
		ValueSerialized: enc,
	}
	return vser, vbuf, nil
}

func decodeToValueTs(vtsSer *ValueTimestampSerialized, valSerde Serde) (interface{}, error) {
	var v interface{}
	var err error
	if vtsSer.ValueSerialized != nil {
		v, err = valSerde.Decode(vtsSer.ValueSerialized)
		if err != nil {
			return nil, err
		}
	}
	return ValueTimestamp{
		Timestamp: vtsSer.Timestamp,
		Value:     v,
	}, nil
}

func valTsSerToValueTs(vtsSer *ValueTimestampSerialized, valSerde Serde) (ValueTimestamp, error) {
	var v interface{}
	var err error
	if vtsSer.ValueSerialized != nil {
		v, err = valSerde.Decode(vtsSer.ValueSerialized)
		if err != nil {
			return ValueTimestamp{}, err
		}
	}
	return ValueTimestamp{
		Timestamp: vtsSer.Timestamp,
		Value:     v,
	}, nil
}

type ValueTimestampJSONSerde struct {
	ValJSONSerde Serde
	DefaultJSONSerde
}

var _ = Serde(ValueTimestampJSONSerde{})

func (s ValueTimestampJSONSerde) String() string {
	return fmt.Sprintf("ValueTimestampJSONSerde{val: %s}", s.ValJSONSerde.String())
}

func (s ValueTimestampJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	vs, vbuf, err := convertToValueTsSer(value, s.ValJSONSerde)
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

func (s ValueTimestampJSONSerde) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	vs := ValueTimestampSerialized{}
	if err := json.Unmarshal(value, &vs); err != nil {
		return nil, err
	}
	return decodeToValueTs(&vs, s.ValJSONSerde)
}

type ValueTimestampMsgpSerde struct {
	DefaultMsgpSerde
	ValMsgpSerde Serde
}

func (s ValueTimestampMsgpSerde) String() string {
	return fmt.Sprintf("ValueTimestampMsgpSerde{val: %s}", s.ValMsgpSerde.String())
}

func (s ValueTimestampMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	vs, vbuf, err := convertToValueTsSer(value, s.ValMsgpSerde)
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

func (s ValueTimestampMsgpSerde) Decode(value []byte) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	vs := ValueTimestampSerialized{}
	_, err := vs.UnmarshalMsg(value)
	if err != nil {
		return nil, err
	}
	return decodeToValueTs(&vs, s.ValMsgpSerde)
}

func GetValueTsSerde(serdeFormat SerdeFormat, valSerde Serde) (Serde, error) {
	var vtSerde Serde
	if serdeFormat == JSON {
		vtSerde = ValueTimestampJSONSerde{
			ValJSONSerde: valSerde,
		}
	} else if serdeFormat == MSGP {
		vtSerde = ValueTimestampMsgpSerde{
			ValMsgpSerde: valSerde,
		}
	} else {
		return nil, fmt.Errorf("serde format should be either json or msgp; but %v is given", serdeFormat)
	}
	return vtSerde, nil
}
