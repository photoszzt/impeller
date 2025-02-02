package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type MessageSerializedJSONSerde struct {
	DefaultJSONSerde
}

func (s MessageSerializedJSONSerde) String() string {
	return "MessageSerializedJSONSerde"
}

var _ = fmt.Stringer(MessageSerializedJSONSerde{})

var _ = Serde(MessageSerializedJSONSerde{})

func (s MessageSerializedJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*MessageSerialized)
	if !ok {
		vTmp := value.(MessageSerialized)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s MessageSerializedJSONSerde) Decode(value []byte) (interface{}, error) {
	v := MessageSerialized{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type MessageSerializedMsgpSerde struct {
	DefaultMsgpSerde
}

var _ = Serde(MessageSerializedMsgpSerde{})

func (s MessageSerializedMsgpSerde) String() string {
	return "MessageSerializedMsgpSerde"
}

var _ = fmt.Stringer(MessageSerializedMsgpSerde{})

func (s MessageSerializedMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*MessageSerialized)
	if !ok {
		vTmp := value.(MessageSerialized)
		v = &vTmp
	}
	b := PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s MessageSerializedMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := MessageSerialized{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetMessageSerializedSerde(serdeFormat SerdeFormat) (Serde, error) {
	switch serdeFormat {
	case JSON:
		return MessageSerializedJSONSerde{}, nil
	case MSGP:
		return MessageSerializedMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
