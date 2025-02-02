package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type EpochMarkerJSONSerde struct {
	DefaultJSONSerde
}

func (s EpochMarkerJSONSerde) String() string {
	return "EpochMarkerJSONSerde"
}

var _ = fmt.Stringer(EpochMarkerJSONSerde{})

var _ = Serde(EpochMarkerJSONSerde{})

func (s EpochMarkerJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*EpochMarker)
	if !ok {
		vTmp := value.(EpochMarker)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s EpochMarkerJSONSerde) Decode(value []byte) (interface{}, error) {
	v := EpochMarker{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type EpochMarkerMsgpSerde struct {
	DefaultMsgpSerde
}

var _ = Serde(EpochMarkerMsgpSerde{})

func (s EpochMarkerMsgpSerde) String() string {
	return "EpochMarkerMsgpSerde"
}

var _ = fmt.Stringer(EpochMarkerMsgpSerde{})

func (s EpochMarkerMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*EpochMarker)
	if !ok {
		vTmp := value.(EpochMarker)
		v = &vTmp
	}
	b := PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s EpochMarkerMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := EpochMarker{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetEpochMarkerSerde(serdeFormat SerdeFormat) (Serde, error) {
	switch serdeFormat {
	case JSON:
		return EpochMarkerJSONSerde{}, nil
	case MSGP:
		return EpochMarkerMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
