package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type KeyValuePair[K, V any] struct {
	Key   K
	Value V
}

type KeyValuePairs[K, V any] []*KeyValuePair[K, V]

func KVPairToKVPairSer[K, V any](value *KeyValuePair[K, V], keySerde SerdeG[K], valSerde SerdeG[V]) (KeyValueSerrialized, *[]byte, *[]byte, error) {
	kenc, kbuf, err := keySerde.Encode(value.Key)
	if err != nil {
		return KeyValueSerrialized{}, nil, nil, fmt.Errorf("fail to encode key: %v", err)
	}
	venc, vbuf, err := valSerde.Encode(value.Value)
	if err != nil {
		return KeyValueSerrialized{}, nil, nil, fmt.Errorf("fail encode val: %v", err)
	}
	return KeyValueSerrialized{
		KeyEnc:   kenc,
		ValueEnc: venc,
	}, kbuf, vbuf, nil
}

func KVPairSerToKVPairG[K, V any](value KeyValueSerrialized, keySerde SerdeG[K], valSerde SerdeG[V]) (*KeyValuePair[K, V], error) {
	key, err := keySerde.Decode(value.KeyEnc)
	if err != nil {
		return nil, fmt.Errorf("fail to decode key: %v", err)
	}
	val, err := valSerde.Decode(value.ValueEnc)
	if err != nil {
		return nil, fmt.Errorf("fail to decode val: %v", err)
	}
	return &KeyValuePair[K, V]{
		Key:   key,
		Value: val,
	}, nil
}

type KeyValuePairJSONSerdeG[K, V any] struct {
	keySerde SerdeG[K]
	valSerde SerdeG[V]
	DefaultJSONSerde
}

func (s KeyValuePairJSONSerdeG[K, V]) String() string {
	return fmt.Sprintf("KeyValuePairJSONSerdeG{key: %s, val: %s}", s.keySerde.String(), s.valSerde.String())
}

func (s KeyValuePairJSONSerdeG[K, V]) Encode(v *KeyValuePair[K, V]) ([]byte, *[]byte, error) {
	kvser, kbuf, vbuf, err := KVPairToKVPairSer(v, s.keySerde, s.valSerde)
	defer func() {
		if s.keySerde.UsedBufferPool() && kbuf != nil {
			*kbuf = kvser.KeyEnc
			PushBuffer(kbuf)
		}
		if s.valSerde.UsedBufferPool() && vbuf != nil {
			*vbuf = kvser.ValueEnc
			PushBuffer(vbuf)
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	r, err := json.Marshal(kvser)
	return r, nil, err
}

func (s KeyValuePairJSONSerdeG[K, V]) Decode(v []byte) (*KeyValuePair[K, V], error) {
	var kvser KeyValueSerrialized
	err := json.Unmarshal(v, &kvser)
	if err != nil {
		return nil, err
	}
	return KVPairSerToKVPairG(kvser, s.keySerde, s.valSerde)
}

type KeyValuePairMsgpSerdeG[K, V any] struct {
	DefaultMsgpSerde
	keySerde SerdeG[K]
	valSerde SerdeG[V]
}

func (s KeyValuePairMsgpSerdeG[K, V]) UsedBufferPool() bool { return false }

func (s KeyValuePairMsgpSerdeG[K, V]) String() string {
	return fmt.Sprintf("KeyValuePairMsgpSerdeG{key: %s, val: %s}", s.keySerde.String(), s.valSerde.String())
}

func (s KeyValuePairMsgpSerdeG[K, V]) Encode(v *KeyValuePair[K, V]) ([]byte, *[]byte, error) {
	kvser, kbuf, vbuf, err := KVPairToKVPairSer(v, s.keySerde, s.valSerde)
	if err != nil {
		if s.keySerde.UsedBufferPool() && kbuf != nil {
			*kbuf = kvser.KeyEnc
			PushBuffer(kbuf)
		}
		if s.valSerde.UsedBufferPool() && vbuf != nil {
			*vbuf = kvser.ValueEnc
			PushBuffer(vbuf)
		}
		return nil, nil, err
	}
	// b := PopBuffer(kvser.Msgsize())
	// buf := *b
	r, err := kvser.MarshalMsg(nil)
	if s.keySerde.UsedBufferPool() && kbuf != nil {
		*kbuf = kvser.KeyEnc
		PushBuffer(kbuf)
	}
	if s.valSerde.UsedBufferPool() && vbuf != nil {
		*vbuf = kvser.ValueEnc
		PushBuffer(vbuf)
	}
	return r, nil, err
}

func (s KeyValuePairMsgpSerdeG[K, V]) Decode(v []byte) (*KeyValuePair[K, V], error) {
	var kvser KeyValueSerrialized
	_, err := kvser.UnmarshalMsg(v)
	if err != nil {
		return nil, err
	}
	return KVPairSerToKVPairG(kvser, s.keySerde, s.valSerde)
}

func GetKeyValuePairSerdeG[K, V any](format SerdeFormat, keySerde SerdeG[K], valSerde SerdeG[V]) (SerdeG[*KeyValuePair[K, V]], error) {
	switch format {
	case JSON:
		return KeyValuePairJSONSerdeG[K, V]{keySerde: keySerde, valSerde: valSerde}, nil
	case MSGP:
		return KeyValuePairMsgpSerdeG[K, V]{keySerde: keySerde, valSerde: valSerde}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}

type KeyValuePairsJSONSerdeG[K, V any] struct {
	DefaultJSONSerde
	payloadEnc PayloadArrJSONSerdeG
	s          KeyValuePairJSONSerdeG[K, V]
}

func (s KeyValuePairsJSONSerdeG[K, V]) String() string {
	return fmt.Sprintf("KeyValuePairsJSONSerdeG{%s, %s}", s.payloadEnc.String(), s.s.String())
}

func (s KeyValuePairsJSONSerdeG[K, V]) Encode(v KeyValuePairs[K, V]) ([]byte, *[]byte, error) {
	payloadArr := PayloadArr{
		Payloads: make([][]byte, 0, len(v)),
	}
	for _, kv := range v {
		enc, _, err := s.s.Encode(kv)
		if err != nil {
			return nil, nil, err
		}
		payloadArr.Payloads = append(payloadArr.Payloads, enc)
	}
	r, b, err := s.payloadEnc.Encode(payloadArr)
	// useBuf := s.s.UsedBufferPool()
	// if useBuf {
	// 	buf := new([]byte)
	// 	for _, p := range payloadArr.Payloads {
	// 		*buf = p
	// 		PushBuffer(buf)
	// 	}
	// }
	return r, b, err
}

func (s KeyValuePairsJSONSerdeG[K, V]) Decode(v []byte) (KeyValuePairs[K, V], error) {
	payloadArr, err := s.payloadEnc.Decode(v)
	if err != nil {
		return KeyValuePairs[K, V]{}, err
	}
	kvps := make(KeyValuePairs[K, V], len(payloadArr.Payloads))
	for i, payload := range payloadArr.Payloads {
		kvp, err := s.s.Decode(payload)
		if err != nil {
			return KeyValuePairs[K, V]{}, err
		}
		kvps[i] = kvp
	}
	return kvps, nil
}

type KeyValuePairsMsgpSerdeG[K, V any] struct {
	DefaultMsgpSerde
	payloadEnc PayloadArrMsgpSerdeG
	s          KeyValuePairMsgpSerdeG[K, V]
}

func (s KeyValuePairsMsgpSerdeG[K, V]) String() string {
	return fmt.Sprintf("KeyValuePairsMsgpSerdeG{%s, %s}", s.payloadEnc.String(), s.s.String())
}

func EncodeKVPairs[K, V any](kvs KeyValuePairs[K, V], payloadSerde SerdeG[PayloadArr],
	kvSerde SerdeG[*KeyValuePair[K, V]],
) ([]byte, *[]byte, error) {
	payloadArr := PayloadArr{
		Payloads: make([][]byte, 0, len(kvs)),
	}
	for _, kv := range kvs {
		enc, _, err := kvSerde.Encode(kv)
		if err != nil {
			return nil, nil, err
		}
		payloadArr.Payloads = append(payloadArr.Payloads, enc)
	}
	r, b, err := payloadSerde.Encode(payloadArr)
	// if kvSerde.UsedBufferPool() {
	// 	buf := new([]byte)
	// 	for _, p := range payloadArr.Payloads {
	// 		*buf = p
	// 		PushBuffer(buf)
	// 	}
	// }
	return r, b, err
}

func (s KeyValuePairsMsgpSerdeG[K, V]) Encode(v KeyValuePairs[K, V]) ([]byte, *[]byte, error) {
	payloadArr := PayloadArr{
		Payloads: make([][]byte, 0, len(v)),
	}
	for _, kv := range v {
		enc, _, err := s.s.Encode(kv)
		if err != nil {
			return nil, nil, err
		}
		payloadArr.Payloads = append(payloadArr.Payloads, enc)
	}
	r, b, err := s.payloadEnc.Encode(payloadArr)
	// if s.s.UsedBufferPool() {
	// 	buf := new([]byte)
	// 	for _, p := range payloadArr.Payloads {
	// 		*buf = p
	// 		PushBuffer(buf)
	// 	}
	// }
	return r, b, err
}

func (s KeyValuePairsMsgpSerdeG[K, V]) Decode(v []byte) (KeyValuePairs[K, V], error) {
	payloadArr, err := s.payloadEnc.Decode(v)
	if err != nil {
		return KeyValuePairs[K, V]{}, err
	}
	kvps := make(KeyValuePairs[K, V], len(payloadArr.Payloads))
	for i, payload := range payloadArr.Payloads {
		kvp, err := s.s.Decode(payload)
		if err != nil {
			return KeyValuePairs[K, V]{}, err
		}
		kvps[i] = kvp
	}
	return kvps, nil
}

func GetKeyValuePairsSerdeG[K, V any](format SerdeFormat, keySerde SerdeG[K], valSerde SerdeG[V]) (SerdeG[KeyValuePairs[K, V]], error) {
	switch format {
	case JSON:
		return KeyValuePairsJSONSerdeG[K, V]{
			payloadEnc: PayloadArrJSONSerdeG{},
			s: KeyValuePairJSONSerdeG[K, V]{
				keySerde: keySerde,
				valSerde: valSerde,
			},
		}, nil
	case MSGP:
		return KeyValuePairsMsgpSerdeG[K, V]{
			payloadEnc: PayloadArrMsgpSerdeG{},
			s: KeyValuePairMsgpSerdeG[K, V]{
				keySerde: keySerde,
				valSerde: valSerde,
			},
		}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
