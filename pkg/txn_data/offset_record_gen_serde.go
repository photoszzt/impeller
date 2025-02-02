package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type OffsetRecordJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s OffsetRecordJSONSerde) String() string {
	return "OffsetRecordJSONSerde"
}

var _ = fmt.Stringer(OffsetRecordJSONSerde{})

var _ = commtypes.Serde(OffsetRecordJSONSerde{})

func (s OffsetRecordJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*OffsetRecord)
	if !ok {
		vTmp := value.(OffsetRecord)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s OffsetRecordJSONSerde) Decode(value []byte) (interface{}, error) {
	v := OffsetRecord{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type OffsetRecordMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(OffsetRecordMsgpSerde{})

func (s OffsetRecordMsgpSerde) String() string {
	return "OffsetRecordMsgpSerde"
}

var _ = fmt.Stringer(OffsetRecordMsgpSerde{})

func (s OffsetRecordMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*OffsetRecord)
	if !ok {
		vTmp := value.(OffsetRecord)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s OffsetRecordMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := OffsetRecord{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetOffsetRecordSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return OffsetRecordJSONSerde{}, nil
	case commtypes.MSGP:
		return OffsetRecordMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
