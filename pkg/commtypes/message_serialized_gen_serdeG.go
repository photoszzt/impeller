package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type MessageSerializedJSONSerdeG struct {
	DefaultJSONSerde
}

func (s MessageSerializedJSONSerdeG) String() string {
	return "MessageSerializedJSONSerdeG"
}

var _ = fmt.Stringer(MessageSerializedJSONSerdeG{})

var _ = SerdeG[MessageSerialized](MessageSerializedJSONSerdeG{})

type MessageSerializedMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s MessageSerializedMsgpSerdeG) String() string {
	return "MessageSerializedMsgpSerdeG"
}

var _ = fmt.Stringer(MessageSerializedMsgpSerdeG{})

var _ = SerdeG[MessageSerialized](MessageSerializedMsgpSerdeG{})

func (s MessageSerializedJSONSerdeG) Encode(value MessageSerialized) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s MessageSerializedJSONSerdeG) Decode(value []byte) (MessageSerialized, error) {
	v := MessageSerialized{}
	if err := json.Unmarshal(value, &v); err != nil {
		return MessageSerialized{}, err
	}
	return v, nil
}

func (s MessageSerializedMsgpSerdeG) Encode(value MessageSerialized) ([]byte, *[]byte, error) {
	// b := PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s MessageSerializedMsgpSerdeG) Decode(value []byte) (MessageSerialized, error) {
	v := MessageSerialized{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return MessageSerialized{}, err
	}
	return v, nil
}

func GetMessageSerializedSerdeG(serdeFormat SerdeFormat) (SerdeG[MessageSerialized], error) {
	if serdeFormat == JSON {
		return MessageSerializedJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return MessageSerializedMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
