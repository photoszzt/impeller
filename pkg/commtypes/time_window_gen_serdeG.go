package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type TimeWindowJSONSerdeG struct {
	DefaultJSONSerde
}

func (s TimeWindowJSONSerdeG) String() string {
	return "TimeWindowJSONSerdeG"
}

var _ = fmt.Stringer(TimeWindowJSONSerdeG{})

var _ = SerdeG[TimeWindow](TimeWindowJSONSerdeG{})

type TimeWindowMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s TimeWindowMsgpSerdeG) String() string {
	return "TimeWindowMsgpSerdeG"
}

var _ = fmt.Stringer(TimeWindowMsgpSerdeG{})

var _ = SerdeG[TimeWindow](TimeWindowMsgpSerdeG{})

func (s TimeWindowJSONSerdeG) Encode(value TimeWindow) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s TimeWindowJSONSerdeG) Decode(value []byte) (TimeWindow, error) {
	v := TimeWindow{}
	if err := json.Unmarshal(value, &v); err != nil {
		return TimeWindow{}, err
	}
	return v, nil
}

func (s TimeWindowMsgpSerdeG) Encode(value TimeWindow) ([]byte, *[]byte, error) {
	// b := PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s TimeWindowMsgpSerdeG) Decode(value []byte) (TimeWindow, error) {
	v := TimeWindow{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return TimeWindow{}, err
	}
	return v, nil
}

func GetTimeWindowSerdeG(serdeFormat SerdeFormat) (SerdeG[TimeWindow], error) {
	if serdeFormat == JSON {
		return TimeWindowJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return TimeWindowMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
