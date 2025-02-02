package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type OffsetMarkerJSONSerdeG struct {
	DefaultJSONSerde
}

func (s OffsetMarkerJSONSerdeG) String() string {
	return "OffsetMarkerJSONSerdeG"
}

var _ = fmt.Stringer(OffsetMarkerJSONSerdeG{})

var _ = SerdeG[OffsetMarker](OffsetMarkerJSONSerdeG{})

type OffsetMarkerMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s OffsetMarkerMsgpSerdeG) String() string {
	return "OffsetMarkerMsgpSerdeG"
}

var _ = fmt.Stringer(OffsetMarkerMsgpSerdeG{})

var _ = SerdeG[OffsetMarker](OffsetMarkerMsgpSerdeG{})

func (s OffsetMarkerJSONSerdeG) Encode(value OffsetMarker) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s OffsetMarkerJSONSerdeG) Decode(value []byte) (OffsetMarker, error) {
	v := OffsetMarker{}
	if err := json.Unmarshal(value, &v); err != nil {
		return OffsetMarker{}, err
	}
	return v, nil
}

func (s OffsetMarkerMsgpSerdeG) Encode(value OffsetMarker) ([]byte, *[]byte, error) {
	// b := PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s OffsetMarkerMsgpSerdeG) Decode(value []byte) (OffsetMarker, error) {
	v := OffsetMarker{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return OffsetMarker{}, err
	}
	return v, nil
}

func GetOffsetMarkerSerdeG(serdeFormat SerdeFormat) (SerdeG[OffsetMarker], error) {
	if serdeFormat == JSON {
		return OffsetMarkerJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return OffsetMarkerMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
