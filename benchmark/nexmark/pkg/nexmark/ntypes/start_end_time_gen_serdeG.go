package ntypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type StartEndTimeJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s StartEndTimeJSONSerdeG) String() string {
	return "StartEndTimeJSONSerdeG"
}

var _ = fmt.Stringer(StartEndTimeJSONSerdeG{})

var _ = commtypes.SerdeG[StartEndTime](StartEndTimeJSONSerdeG{})

type StartEndTimeMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s StartEndTimeMsgpSerdeG) String() string {
	return "StartEndTimeMsgpSerdeG"
}

var _ = fmt.Stringer(StartEndTimeMsgpSerdeG{})

var _ = commtypes.SerdeG[StartEndTime](StartEndTimeMsgpSerdeG{})

func (s StartEndTimeJSONSerdeG) Encode(value StartEndTime) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s StartEndTimeJSONSerdeG) Decode(value []byte) (StartEndTime, error) {
	v := StartEndTime{}
	if err := json.Unmarshal(value, &v); err != nil {
		return StartEndTime{}, err
	}
	return v, nil
}

func (s StartEndTimeMsgpSerdeG) Encode(value StartEndTime) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s StartEndTimeMsgpSerdeG) Decode(value []byte) (StartEndTime, error) {
	v := StartEndTime{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return StartEndTime{}, err
	}
	return v, nil
}

func GetStartEndTimeSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[StartEndTime], error) {
	if serdeFormat == commtypes.JSON {
		return StartEndTimeJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return StartEndTimeMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
