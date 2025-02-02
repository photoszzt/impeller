package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type TxnMetadataJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s TxnMetadataJSONSerde) String() string {
	return "TxnMetadataJSONSerde"
}

var _ = fmt.Stringer(TxnMetadataJSONSerde{})

var _ = commtypes.Serde(TxnMetadataJSONSerde{})

func (s TxnMetadataJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*TxnMetadata)
	if !ok {
		vTmp := value.(TxnMetadata)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s TxnMetadataJSONSerde) Decode(value []byte) (interface{}, error) {
	v := TxnMetadata{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return v, nil
}

type TxnMetadataMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

var _ = commtypes.Serde(TxnMetadataMsgpSerde{})

func (s TxnMetadataMsgpSerde) String() string {
	return "TxnMetadataMsgpSerde"
}

var _ = fmt.Stringer(TxnMetadataMsgpSerde{})

func (s TxnMetadataMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*TxnMetadata)
	if !ok {
		vTmp := value.(TxnMetadata)
		v = &vTmp
	}
	b := commtypes.PopBuffer(v.Msgsize())
	buf := *b
	r, err := v.MarshalMsg(buf[:0])
	return r, b, err
}

func (s TxnMetadataMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := TxnMetadata{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return v, nil
}

func GetTxnMetadataSerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return TxnMetadataJSONSerde{}, nil
	case commtypes.MSGP:
		return TxnMetadataMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
