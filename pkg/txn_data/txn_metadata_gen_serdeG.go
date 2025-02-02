package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type TxnMetadataJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s TxnMetadataJSONSerdeG) String() string {
	return "TxnMetadataJSONSerdeG"
}

var _ = fmt.Stringer(TxnMetadataJSONSerdeG{})

var _ = commtypes.SerdeG[TxnMetadata](TxnMetadataJSONSerdeG{})

type TxnMetadataMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s TxnMetadataMsgpSerdeG) String() string {
	return "TxnMetadataMsgpSerdeG"
}

var _ = fmt.Stringer(TxnMetadataMsgpSerdeG{})

var _ = commtypes.SerdeG[TxnMetadata](TxnMetadataMsgpSerdeG{})

func (s TxnMetadataJSONSerdeG) Encode(value TxnMetadata) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s TxnMetadataJSONSerdeG) Decode(value []byte) (TxnMetadata, error) {
	v := TxnMetadata{}
	if err := json.Unmarshal(value, &v); err != nil {
		return TxnMetadata{}, err
	}
	return v, nil
}

func (s TxnMetadataMsgpSerdeG) Encode(value TxnMetadata) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s TxnMetadataMsgpSerdeG) Decode(value []byte) (TxnMetadata, error) {
	v := TxnMetadata{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return TxnMetadata{}, err
	}
	return v, nil
}

func GetTxnMetadataSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[TxnMetadata], error) {
	if serdeFormat == commtypes.JSON {
		return TxnMetadataJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return TxnMetadataMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
