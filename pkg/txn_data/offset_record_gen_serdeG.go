package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type OffsetRecordJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s OffsetRecordJSONSerdeG) String() string {
	return "OffsetRecordJSONSerdeG"
}

var _ = fmt.Stringer(OffsetRecordJSONSerdeG{})

var _ = commtypes.SerdeG[OffsetRecord](OffsetRecordJSONSerdeG{})

type OffsetRecordMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s OffsetRecordMsgpSerdeG) String() string {
	return "OffsetRecordMsgpSerdeG"
}

var _ = fmt.Stringer(OffsetRecordMsgpSerdeG{})

var _ = commtypes.SerdeG[OffsetRecord](OffsetRecordMsgpSerdeG{})

func (s OffsetRecordJSONSerdeG) Encode(value OffsetRecord) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s OffsetRecordJSONSerdeG) Decode(value []byte) (OffsetRecord, error) {
	v := OffsetRecord{}
	if err := json.Unmarshal(value, &v); err != nil {
		return OffsetRecord{}, err
	}
	return v, nil
}

func (s OffsetRecordMsgpSerdeG) Encode(value OffsetRecord) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s OffsetRecordMsgpSerdeG) Decode(value []byte) (OffsetRecord, error) {
	v := OffsetRecord{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return OffsetRecord{}, err
	}
	return v, nil
}

func GetOffsetRecordSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[OffsetRecord], error) {
	if serdeFormat == commtypes.JSON {
		return OffsetRecordJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return OffsetRecordMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
