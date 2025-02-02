package txn_data

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/commtypes"
)

type TopicPartitionJSONSerdeG struct {
	commtypes.DefaultJSONSerde
}

func (s TopicPartitionJSONSerdeG) String() string {
	return "TopicPartitionJSONSerdeG"
}

var _ = fmt.Stringer(TopicPartitionJSONSerdeG{})

var _ = commtypes.SerdeG[*TopicPartition](TopicPartitionJSONSerdeG{})

type TopicPartitionMsgpSerdeG struct {
	commtypes.DefaultMsgpSerde
}

func (s TopicPartitionMsgpSerdeG) String() string {
	return "TopicPartitionMsgpSerdeG"
}

var _ = fmt.Stringer(TopicPartitionMsgpSerdeG{})

var _ = commtypes.SerdeG[*TopicPartition](TopicPartitionMsgpSerdeG{})

func (s TopicPartitionJSONSerdeG) Encode(value *TopicPartition) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s TopicPartitionJSONSerdeG) Decode(value []byte) (*TopicPartition, error) {
	v := TopicPartition{}
	if err := json.Unmarshal(value, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s TopicPartitionMsgpSerdeG) Encode(value *TopicPartition) ([]byte, *[]byte, error) {
	// b := commtypes.PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s TopicPartitionMsgpSerdeG) Decode(value []byte) (*TopicPartition, error) {
	v := TopicPartition{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return &v, nil
}

func GetTopicPartitionSerdeG(serdeFormat commtypes.SerdeFormat) (commtypes.SerdeG[*TopicPartition], error) {
	if serdeFormat == commtypes.JSON {
		return TopicPartitionJSONSerdeG{}, nil
	} else if serdeFormat == commtypes.MSGP {
		return TopicPartitionMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
