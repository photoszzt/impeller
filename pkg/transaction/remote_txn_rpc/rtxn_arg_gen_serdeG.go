package remote_txn_rpc

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type RTxnArgJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s RTxnArgJSONSerdeG) String() string {
	return "RTxnArgJSONSerdeG"
}

var _ = fmt.Stringer(RTxnArgJSONSerdeG{})

var _ = commtypes.SerdeG[*RTxnArg](RTxnArgJSONSerdeG{})

type RTxnArgMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s RTxnArgMsgpSerdeG) String() string {
	return "RTxnArgMsgpSerdeG"
}

var _ = fmt.Stringer(RTxnArgMsgpSerdeG{})

var _ = commtypes.SerdeG[*RTxnArg](RTxnArgMsgpSerdeG{})

func (s RTxnArgJSONSerdeG) Encode(value *RTxnArg) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s RTxnArgJSONSerdeG) Decode(value []byte) (*RTxnArg, error) {
	v := RTxnArg{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s RTxnArgMsgpSerdeG) Encode(value *RTxnArg) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s RTxnArgMsgpSerdeG) Decode(value []byte) (*RTxnArg, error) {
	v := RTxnArg{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetRTxnArgSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[*RTxnArg], error) {
	if serdeFormat == commtypes.JSON {
		return RTxnArgJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return RTxnArgMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
