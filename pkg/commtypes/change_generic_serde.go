package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/utils"
)

type ChangeSize[V any] struct {
	ValSizeFunc func(v V) int64
}

func (cs ChangeSize[V]) SizeOfChange(c Change) int64 {
	total := int64(0)
	if !utils.IsNil(c.NewVal) {
		total += cs.ValSizeFunc(c.NewVal.(V))
	}
	if !utils.IsNil(c.OldVal) {
		total += cs.ValSizeFunc(c.OldVal.(V))
	}
	return total
}

type ChangeJSONSerdeG struct {
	DefaultJSONSerde
	ValJSONSerde Serde
}

var _ = SerdeG[Change](ChangeJSONSerdeG{})

func (s ChangeJSONSerdeG) String() string {
	return fmt.Sprintf("ChangeJSONSerdeG{val: %s}", s.ValJSONSerde.String())
}

func (s ChangeJSONSerdeG) Encode(value Change) ([]byte, *[]byte, error) {
	c, newBuf, oldBuf, err := changeToChangeSer(value, s.ValJSONSerde)
	defer func() {
		if s.ValJSONSerde.UsedBufferPool() && c != nil {
			if c.NewValSerialized != nil {
				*newBuf = c.NewValSerialized
				PushBuffer(newBuf)
			}
			if c.OldValSerialized != nil {
				*oldBuf = c.OldValSerialized
				PushBuffer(oldBuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	r, err := json.Marshal(c)
	return r, nil, err
}

func (s ChangeJSONSerdeG) Decode(value []byte) (Change, error) {
	val := ChangeSerialized{}
	if err := json.Unmarshal(value, &val); err != nil {
		return Change{}, err
	}
	return changeSerToChange(&val, s.ValJSONSerde)
}

type ChangeMsgpSerdeG struct {
	DefaultMsgpSerde
	ValMsgpSerde Serde
}

var _ = SerdeG[Change](ChangeMsgpSerdeG{})

func (s ChangeMsgpSerdeG) String() string {
	return fmt.Sprintf("ChangeMsgpSerdeG{val: %s}", s.ValMsgpSerde.String())
}

func (s ChangeMsgpSerdeG) Encode(value Change) ([]byte, *[]byte, error) {
	c, newBuf, oldBuf, err := changeToChangeSer(value, s.ValMsgpSerde)
	defer func() {
		if s.ValMsgpSerde.UsedBufferPool() && c != nil {
			if newBuf != nil {
				*newBuf = c.NewValSerialized
				PushBuffer(newBuf)
			}
			if oldBuf != nil {
				*oldBuf = c.OldValSerialized
				PushBuffer(oldBuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	// b := PopBuffer(c.Msgsize())
	// buf := *b
	// r, err := c.MarshalMsg(buf[:0])
	r, err := c.MarshalMsg(nil)
	return r, nil, err
}

func (s ChangeMsgpSerdeG) Decode(value []byte) (Change, error) {
	val := ChangeSerialized{}
	if _, err := val.UnmarshalMsg(value); err != nil {
		return Change{}, err
	}
	return changeSerToChange(&val, s.ValMsgpSerde)
}

func GetChangeSerdeG(serdeFormat SerdeFormat, valSerde Serde) (SerdeG[Change], error) {
	if serdeFormat == JSON {
		return ChangeJSONSerdeG{
			ValJSONSerde: valSerde,
		}, nil
	} else if serdeFormat == MSGP {
		return ChangeMsgpSerdeG{
			ValMsgpSerde: valSerde,
		}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
