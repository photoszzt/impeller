package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
)

type TableSnapshotsJSONSerdeG struct {
	DefaultJSONSerde
}

func (s TableSnapshotsJSONSerdeG) String() string {
	return "TableSnapshotsJSONSerdeG"
}

var _ = fmt.Stringer(TableSnapshotsJSONSerdeG{})

var _ = SerdeG[TableSnapshots](TableSnapshotsJSONSerdeG{})

type TableSnapshotsMsgpSerdeG struct {
	DefaultMsgpSerde
}

func (s TableSnapshotsMsgpSerdeG) String() string {
	return "TableSnapshotsMsgpSerdeG"
}

var _ = fmt.Stringer(TableSnapshotsMsgpSerdeG{})

var _ = SerdeG[TableSnapshots](TableSnapshotsMsgpSerdeG{})

func (s TableSnapshotsJSONSerdeG) Encode(value TableSnapshots) ([]byte, *[]byte, error) {
	r, err := json.Marshal(value)
	return r, nil, err
}

func (s TableSnapshotsJSONSerdeG) Decode(value []byte) (TableSnapshots, error) {
	v := TableSnapshots{}
	if err := json.Unmarshal(value, &v); err != nil {
		return TableSnapshots{}, err
	}
	return v, nil
}

func (s TableSnapshotsMsgpSerdeG) Encode(value TableSnapshots) ([]byte, *[]byte, error) {
	// b := PopBuffer(value.Msgsize())
	// buf := *b
	// r, err := value.MarshalMsg(buf[:0])
	r, err := value.MarshalMsg(nil)
	return r, nil, err
}

func (s TableSnapshotsMsgpSerdeG) Decode(value []byte) (TableSnapshots, error) {
	v := TableSnapshots{}
	if _, err := v.UnmarshalMsg(value); err != nil {
		return TableSnapshots{}, err
	}
	return v, nil
}

func GetTableSnapshotsSerdeG(serdeFormat SerdeFormat) (SerdeG[TableSnapshots], error) {
	if serdeFormat == JSON {
		return TableSnapshotsJSONSerdeG{}, nil
	} else if serdeFormat == MSGP {
		return TableSnapshotsMsgpSerdeG{}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
