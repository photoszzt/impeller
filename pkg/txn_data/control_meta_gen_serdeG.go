package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type ControlMetadataJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s ControlMetadataJSONSerdeG) String() string {
	return "ControlMetadataJSONSerdeG"
}

var _ = fmt.Stringer(ControlMetadataJSONSerdeG{})

var _ = commtypes.SerdeG[ControlMetadata](ControlMetadataJSONSerdeG{})

type ControlMetadataMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s ControlMetadataMsgpSerdeG) String() string {
	return "ControlMetadataMsgpSerdeG"
}

var _ = fmt.Stringer(ControlMetadataMsgpSerdeG{})

var _ = commtypes.SerdeG[ControlMetadata](ControlMetadataMsgpSerdeG{})

func (s ControlMetadataJSONSerdeG) Encode(value ControlMetadata) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s ControlMetadataJSONSerdeG) Decode(value []byte) (ControlMetadata, error) {
	v := ControlMetadata{}
	if err := json.Unmarshal(value, &v); err != nil {
		return ControlMetadata{}, err
	}
	return v, nil
}

func (s ControlMetadataMsgpSerdeG) Encode(value ControlMetadata) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s ControlMetadataMsgpSerdeG) Decode(value []byte) (ControlMetadata, error) {
	v := ControlMetadata{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return ControlMetadata{}, err
	}
	return v, nil
}

func GetControlMetadataSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[ControlMetadata], error) {
	if serdeFormat == commtypes.JSON {
		return ControlMetadataJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return ControlMetadataMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
