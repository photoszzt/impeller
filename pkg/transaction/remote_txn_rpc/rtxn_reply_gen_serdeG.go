package remote_txn_rpc

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type RTxnReplyJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s RTxnReplyJSONSerdeG) String() string {
	return "RTxnReplyJSONSerdeG"
}

var _ = fmt.Stringer(RTxnReplyJSONSerdeG{})

var _ = commtypes.SerdeG[*RTxnReply](RTxnReplyJSONSerdeG{})

type RTxnReplyMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s RTxnReplyMsgpSerdeG) String() string {
	return "RTxnReplyMsgpSerdeG"
}

var _ = fmt.Stringer(RTxnReplyMsgpSerdeG{})

var _ = commtypes.SerdeG[*RTxnReply](RTxnReplyMsgpSerdeG{})

func (s RTxnReplyJSONSerdeG) Encode(value *RTxnReply) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s RTxnReplyJSONSerdeG) Decode(value []byte) (*RTxnReply, error) {
	v := RTxnReply{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s RTxnReplyMsgpSerdeG) Encode(value *RTxnReply) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s RTxnReplyMsgpSerdeG) Decode(value []byte) (*RTxnReply, error) {
	v := RTxnReply{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetRTxnReplySerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[*RTxnReply], error) {
	if serdeFormat == commtypes.JSON {
		return RTxnReplyJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return RTxnReplyMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
