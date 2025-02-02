//go:generate msgp
//msgp:ignore MessageMsgpSerde MessageJSONSerde

package commtypes

import (
	"encoding/json"
	"fmt"
	"os"
	"sharedlog-stream/pkg/utils"
	"slices"
)

type MessageSerialized struct {
	KeyEnc      []byte `json:"key,omitempty" msg:"key,omitempty"`
	ValueEnc    []byte `json:"val,omitempty" msg:"val,omitempty"`
	InjTMs      int64  `msg:"injT" json:"injT"`
	TimestampMs int64  `msg:"ts,omitempty" json:"ts,omitempty"`
}

func (m *MessageSerialized) UpdateInjectTime(ts int64) {
	m.InjTMs = ts
}

func (m *MessageSerialized) ExtractInjectTimeMs() int64 {
	return m.InjTMs
}

func compareSlice(a, b []byte) bool {
	var equal bool
	if a != nil {
		if b != nil {
			equal = slices.Equal(a, b)
		} else {
			equal = len(a) == 0
		}
	} else {
		equal = len(b) == 0
	}
	return equal
}

func (m *MessageSerialized) Equal(b *MessageSerialized) bool {
	keyEqual := compareSlice(m.KeyEnc, b.KeyEnc)
	valEqual := compareSlice(m.ValueEnc, b.ValueEnc)
	return keyEqual && valEqual &&
		m.InjTMs == b.InjTMs && m.TimestampMs == b.TimestampMs
}

func convertToMsgSer(value interface{}, keySerde Serde, valSerde Serde) (msg *MessageSerialized, kbuf *[]byte, vbuf *[]byte, err error) {
	if value == nil {
		return nil, nil, nil, nil
	}
	v, ok := value.(*Message)
	if !ok {
		vtmp := value.(Message)
		v = &vtmp
	}
	if v == nil {
		return nil, nil, nil, nil
	}

	var kenc []byte
	if !utils.IsNil(v.Key) {
		kenc, kbuf, err = keySerde.Encode(v.Key)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to encode key: %v", err)
		}
	}

	var venc []byte
	if !utils.IsNil(v.Value) {
		venc, vbuf, err = valSerde.Encode(v.Value)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail encode val: %v", err)
		}
	}
	if kenc == nil && venc == nil {
		return nil, nil, nil, nil
	}
	msg = &MessageSerialized{
		KeyEnc:      kenc,
		ValueEnc:    venc,
		InjTMs:      v.InjT,
		TimestampMs: v.Timestamp,
	}
	return msg, kbuf, vbuf, nil
}

func decodeToMsg(msgSer *MessageSerialized, keySerde Serde, valSerde Serde) (interface{}, error) {
	var err error
	var key interface{} = nil
	if msgSer.KeyEnc != nil {
		key, err = keySerde.Decode(msgSer.KeyEnc)
		if err != nil {
			return nil, fmt.Errorf("fail to decode key: %v", err)
		}
	}
	var val interface{} = nil
	if msgSer.ValueEnc != nil {
		val, err = valSerde.Decode(msgSer.ValueEnc)
		if err != nil {
			return nil, fmt.Errorf("fail to decode val: %v", err)
		}
	}
	msg := Message{
		Key:       key,
		Value:     val,
		InjT:      msgSer.InjTMs,
		Timestamp: msgSer.TimestampMs,
	}
	return msg, nil
}

type MessageMsgpSerde struct {
	DefaultMsgpSerde
	keySerde Serde
	valSerde Serde
}

var _ MessageSerde = MessageMsgpSerde{}

func (s MessageMsgpSerde) String() string {
	return fmt.Sprintf("MessageMsgpSerde{key: %s, val: %s}", s.keySerde.String(), s.valSerde.String())
}

func (s MessageMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	msgSer, kbuf, vbuf, err := convertToMsgSer(value, s.keySerde, s.valSerde)
	defer func() {
		if msgSer != nil {
			if s.keySerde.UsedBufferPool() && kbuf != nil {
				*kbuf = msgSer.KeyEnc
				PushBuffer(kbuf)
			}
			if s.valSerde.UsedBufferPool() && vbuf != nil {
				*vbuf = msgSer.ValueEnc
				PushBuffer(vbuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if msgSer == nil {
		return nil, nil, nil
	}
	b := PopBuffer(msgSer.Msgsize())
	buf := *b
	r, err := msgSer.MarshalMsg(buf[:0])
	return r, b, err
}

func (s MessageMsgpSerde) EncodeAndRtnKVBin(value interface{}) (msgEnc []byte, kenc []byte /* kEnc */, venc []byte /* vEnc */, b *[]byte, err error) {
	msgSer, kbuf, vbuf, err := convertToMsgSer(value, s.keySerde, s.valSerde)
	defer func() {
		if msgSer != nil {
			if s.keySerde.UsedBufferPool() && kbuf != nil {
				*kbuf = msgSer.KeyEnc
				PushBuffer(kbuf)
			}
			if s.valSerde.UsedBufferPool() && vbuf != nil {
				*vbuf = msgSer.ValueEnc
				PushBuffer(vbuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if msgSer == nil {
		return nil, nil, nil, nil, nil
	}
	b = PopBuffer(msgSer.Msgsize())
	buf := *b
	msgEnc, err = msgSer.MarshalMsg(buf[:0])
	kenc = msgSer.KeyEnc
	venc = msgSer.ValueEnc
	return msgEnc, kenc, venc, b, err
}

func (s MessageMsgpSerde) EncodeKey(key interface{}) ([]byte, *[]byte, error) {
	if key == nil {
		return nil, nil, nil
	}
	return s.keySerde.Encode(key)
}

func (s MessageMsgpSerde) DecodeVal(val []byte) (interface{}, error) {
	return s.valSerde.Decode(val)
}

func (s MessageMsgpSerde) EncodeVal(value interface{}) ([]byte, *[]byte, error) {
	if value == nil {
		return nil, nil, nil
	}
	return s.valSerde.Encode(value)
}

func (s MessageMsgpSerde) Decode(value []byte) (interface{}, error) {
	msgSer := MessageSerialized{}
	_, err := msgSer.UnmarshalMsg(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] fail to unmarshal this msg3: %v", string(value))
		return nil, fmt.Errorf("fail to unmarshal msg: %v", err)
	}
	return decodeToMsg(&msgSer, s.keySerde, s.valSerde)
}

type MessageJSONSerde struct {
	DefaultJSONSerde
	KeySerde Serde
	ValSerde Serde
}

var _ MessageSerde = MessageJSONSerde{}

func (s MessageJSONSerde) String() string {
	return fmt.Sprintf("MessageJSONSerde{key: %s, val: %s}", s.KeySerde.String(), s.ValSerde.String())
}

func (s MessageJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	msgSer, kbuf, vbuf, err := convertToMsgSer(value, s.KeySerde, s.ValSerde)
	defer func() {
		if msgSer != nil {
			if s.KeySerde.UsedBufferPool() && msgSer.KeyEnc != nil {
				*kbuf = msgSer.KeyEnc
				PushBuffer(kbuf)
			}
			if s.ValSerde.UsedBufferPool() && msgSer.ValueEnc != nil {
				*vbuf = msgSer.ValueEnc
				PushBuffer(vbuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if msgSer == nil {
		return nil, nil, nil
	}
	r, err := json.Marshal(msgSer)
	return r, nil, err
}

func (s MessageJSONSerde) EncodeAndRtnKVBin(value interface{}) ([]byte, []byte /* kEnc */, []byte /* vEnc */, *[]byte, error) {
	msgSer, kbuf, vbuf, err := convertToMsgSer(value, s.KeySerde, s.ValSerde)
	defer func() {
		if msgSer != nil {
			if s.KeySerde.UsedBufferPool() && msgSer.KeyEnc != nil {
				*kbuf = msgSer.KeyEnc
				PushBuffer(kbuf)
			}
			if s.ValSerde.UsedBufferPool() && msgSer.ValueEnc != nil {
				*vbuf = msgSer.ValueEnc
				PushBuffer(vbuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if msgSer == nil {
		return nil, nil, nil, nil, nil
	}
	msgEnc, err := json.Marshal(msgSer)
	return msgEnc, msgSer.KeyEnc, msgSer.ValueEnc, nil, err
}

func (s MessageJSONSerde) EncodeKey(key interface{}) ([]byte, *[]byte, error) {
	return s.KeySerde.Encode(key)
}

func (s MessageJSONSerde) EncodeVal(value interface{}) ([]byte, *[]byte, error) {
	return s.ValSerde.Encode(value)
}

func (s MessageJSONSerde) DecodeVal(v []byte) (interface{}, error) {
	return s.ValSerde.Decode(v)
}

func (s MessageJSONSerde) Decode(value []byte) (interface{}, error) {
	msgSer := MessageSerialized{}
	if err := json.Unmarshal(value, &msgSer); err != nil {
		return nil, err
	}
	return decodeToMsg(&msgSer, s.KeySerde, s.ValSerde)
}

func GetMsgSerde(serdeFormat SerdeFormat, keySerde Serde, valSerde Serde) (MessageSerde, error) {
	if serdeFormat == JSON {
		return MessageJSONSerde{
			KeySerde: keySerde,
			ValSerde: valSerde,
		}, nil
	} else if serdeFormat == MSGP {
		return MessageMsgpSerde{
			keySerde: keySerde,
			valSerde: valSerde,
		}, nil
	} else {
		return nil, fmt.Errorf("serde format should be either json or msgp; but %v is given", serdeFormat)
	}
}
