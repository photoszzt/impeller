package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type OffsetMarkerJSONSerde struct {
	DefaultJSONSerde
}

func (s OffsetMarkerJSONSerde) String() string {
	return "OffsetMarkerJSONSerde"
}

var _ = fmt.Stringer(OffsetMarkerJSONSerde{})

var _ = Serde(OffsetMarkerJSONSerde{})

func (s OffsetMarkerJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*OffsetMarker)
	if !ok {
		vTmp := value.(OffsetMarker)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s OffsetMarkerJSONSerde) Decode(value []byte) (interface{}, error) {
	v := OffsetMarker{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type OffsetMarkerMsgpSerde struct {
	DefaultMsgpSerde
}

var _ = Serde(OffsetMarkerMsgpSerde{})

func (s OffsetMarkerMsgpSerde) String() string {
	return "OffsetMarkerMsgpSerde"
}

var _ = fmt.Stringer(OffsetMarkerMsgpSerde{})

func (s OffsetMarkerMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*OffsetMarker)
	if !ok {
		vTmp := value.(OffsetMarker)
		v = &vTmp
	}
	b := PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s OffsetMarkerMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := OffsetMarker{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetOffsetMarkerSerde(serdeFormat SerdeFormat) (Serde, error) {
	switch serdeFormat {
	case JSON:
		return OffsetMarkerJSONSerde{}, nil
	case MSGP:
		return OffsetMarkerMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
