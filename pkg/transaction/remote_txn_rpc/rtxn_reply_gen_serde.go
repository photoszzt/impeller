package remote_txn_rpc

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type RTxnReplyJSONSerde struct {
	commtypes.DefaultJSONSerde
}

func (s RTxnReplyJSONSerde) String() string {
	return "RTxnReplyJSONSerde"
}

var _ = fmt.Stringer(RTxnReplyJSONSerde{})

var _ = commtypes.Serde(RTxnReplyJSONSerde{})

func (s RTxnReplyJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*RTxnReply)
	if !ok {
		vTmp := value.(RTxnReply)
		v = &vTmp
	}
	r, err := json.Marshal(v)
	return r, nil, err
}

func (s RTxnReplyJSONSerde) Decode(value []byte) (interface{}, error) {
	v := RTxnReply{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

type RTxnReplyMsgpSerde struct {
	commtypes.DefaultMsgpSerde
}

func (s RTxnReplyMsgpSerde) String() string {
	return "RTxnReplyMsgpSerde"
}

var _ = fmt.Stringer(RTxnReplyMsgpSerde{})

var _ = commtypes.Serde(RTxnReplyMsgpSerde{})

func (s RTxnReplyMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	v, ok := value.(*RTxnReply)
	if !ok {
		vTmp := value.(RTxnReply)
		v = &vTmp
	}
	// b := commtypes.PopBuffer(v.Msgsize())
	// buf := *b
	// r, err := v.MarshalMsg(buf[:0])
	r, err := v.MarshalMsg(nil)
	return r, nil, err
}

func (s RTxnReplyMsgpSerde) Decode(value []byte) (interface{}, error) {
	v := RTxnReply{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetRTxnReplySerde(serdeFormat commtypes.SerdeFormat) (commtypes.Serde, error) {
	switch serdeFormat {
	case commtypes.JSON:
		return RTxnReplyJSONSerde{}, nil
	case commtypes.MSGP:
		return RTxnReplyMsgpSerde{}, nil
	default:
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
