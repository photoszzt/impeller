package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type EpochMarkerJSONSerdeG struct {
	DefaultJSONSerde
}

func (s EpochMarkerJSONSerdeG) String() string {
	return "EpochMarkerJSONSerdeG"
}

var _ = fmt.Stringer(EpochMarkerJSONSerdeG{})

var _ = SerdeG[EpochMarker](EpochMarkerJSONSerdeG{})

type EpochMarkerMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s EpochMarkerMsgpSerdeG) String() string {
	return "EpochMarkerMsgpSerdeG"
}

var _ = fmt.Stringer(EpochMarkerMsgpSerdeG{})

var _ = SerdeG[EpochMarker](EpochMarkerMsgpSerdeG{})

func (s EpochMarkerJSONSerdeG) Encode(value EpochMarker) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s EpochMarkerJSONSerdeG) Decode(value []byte) (EpochMarker, error) {
	v := EpochMarker{}
	if err := json.Unmarshal(value, &v); err != nil {
		return EpochMarker{}, err
	}
	return v, nil
}

func (s EpochMarkerMsgpSerdeG) Encode(value EpochMarker) ([]byte, *[]byte, error) {
	// b := PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s EpochMarkerMsgpSerdeG) Decode(value []byte) (EpochMarker, error) {
	v := EpochMarker{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return EpochMarker{}, err
	}
	return v, nil
}

func GetEpochMarkerSerdeG(serdeFormat SerdeFormat) (SerdeG[EpochMarker], error) {
	if serdeFormat == JSON {
		return EpochMarkerJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return EpochMarkerMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
