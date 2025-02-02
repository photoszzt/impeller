package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type EventJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s EventJSONSerde) String() string {
	return "EventJSONSerde"
}

var _ = fmt.Stringer(EventJSONSerde{})

var _ = commtypes.Serde(EventJSONSerde{})

func (s EventJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*Event)
	if !ok {
		vTmp := value.(Event)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s EventJSONSerde) Decode(value []byte) (interface{}, error) {
	v := Event{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

type EventMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

func (s EventMsgpSerde) String() string {
	return "EventMsgpSerde"
}

var _ = fmt.Stringer(EventMsgpSerde{})

var _ = commtypes.Serde(EventMsgpSerde{})

func (s EventMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*Event)
	if !ok {
		vTmp := value.(Event)
		v = &vTmp
	}
	// b := commtypes.PopBuffer(v.Msgsize())
	// buf := *b
	// r, err := v.MarshalMsg(buf[:0])
	r, err := v.MarshalMsg(nil)
	return r, nil, err
}

func (s EventMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := Event{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetEventSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return EventJSONSerde{}, nil
	case commtypes.MSGP:
		return EventMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
