package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type ControlMetadataJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s ControlMetadataJSONSerde) String() string {
	return "ControlMetadataJSONSerde"
}

var _ = fmt.Stringer(ControlMetadataJSONSerde{})

var _ = commtypes.Serde(ControlMetadataJSONSerde{})

func (s ControlMetadataJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*ControlMetadata)
	if !ok {
		vTmp := value.(ControlMetadata)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s ControlMetadataJSONSerde) Decode(value []byte) (interface{}, error) {
	v := ControlMetadata{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type ControlMetadataMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(ControlMetadataMsgpSerde{})

func (s ControlMetadataMsgpSerde) String() string {
	return "ControlMetadataMsgpSerde"
}

var _ = fmt.Stringer(ControlMetadataMsgpSerde{})

func (s ControlMetadataMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*ControlMetadata)
	if !ok {
		vTmp := value.(ControlMetadata)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s ControlMetadataMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := ControlMetadata{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetControlMetadataSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return ControlMetadataJSONSerde{}, nil
	case commtypes.MSGP:
		return ControlMetadataMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
