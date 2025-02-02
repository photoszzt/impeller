package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/optional"
)

type ChangeG[V any] struct {
	NewVal optional.Option[V]
	OldVal optional.Option[V]
}

func NewChangeOnlyNewValG[V any](newVal V) ChangeG[V] {
	return ChangeG[V]{
		NewVal: optional.Some(newVal),
		OldVal: optional.None[V](),
	}
}

func NewChangeOnlyOldValG[V any](oldVal V) ChangeG[V] {
	return ChangeG[V]{
		NewVal: optional.None[V](),
		OldVal: optional.Some(oldVal),
	}
}

func NewChangeG[V any](newVal, oldVal V) ChangeG[V] {
	return ChangeG[V]{
		NewVal: optional.Some(newVal),
		OldVal: optional.Some(oldVal),
	}
}

type ChangeGJSONSerdeG[V any] struct {
	ValJSONSerde SerdeG[V]
	DefaultJSONSerdeG[V]
}

func changeGToChangeSer[V any](value ChangeG[V], valSerde SerdeG[V]) (c *ChangeSerialized, newBuf *[]byte, oldBuf *[]byte, err error) {
	var newValEnc, oldValEnc []byte
	newValMsg, ok := value.NewVal.Take()
	if ok {
		newValEnc, newBuf, err = valSerde.Encode(newValMsg)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	oldValMsg, ok := value.OldVal.Take()
	if ok {
		oldValEnc, oldBuf, err = valSerde.Encode(oldValMsg)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	c = &ChangeSerialized{
		NewValSerialized: newValEnc,
		OldValSerialized: oldValEnc,
	}
	return c, newBuf, oldBuf, nil
}

func changeSerToChangeG[V any](val *ChangeSerialized, valSerde SerdeG[V]) (ChangeG[V], error) {
	newValOp := optional.None[V]()
	oldValOp := optional.None[V]()
	if val.NewValSerialized != nil {
		newVal, err := valSerde.Decode(val.NewValSerialized)
		if err != nil {
			return ChangeG[V]{}, err
		}
		newValOp = optional.Some(newVal)
	}
	if val.OldValSerialized != nil {
		oldVal, err := valSerde.Decode(val.OldValSerialized)
		if err != nil {
			return ChangeG[V]{}, err
		}
		oldValOp = optional.Some(oldVal)
	}
	return ChangeG[V]{
		NewVal: newValOp,
		OldVal: oldValOp,
	}, nil
}

var _ = SerdeG[ChangeG[int]](ChangeGJSONSerdeG[int]{})

func (s ChangeGJSONSerdeG[V]) String() string {
	return fmt.Sprintf("ChangeGJSONSerdeG{val: %s}", s.ValJSONSerde.String())
}

func (s ChangeGJSONSerdeG[V]) Encode(value ChangeG[V]) ([]byte, *[]byte, error) {
	c, newBuf, oldBuf, err := changeGToChangeSer(value, s.ValJSONSerde)
	defer func() {
		if s.ValJSONSerde.UsedBufferPool() && c != nil {
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
	r, err := json.Marshal(c)
	return r, nil, err
}

func (s ChangeGJSONSerdeG[V]) Decode(value []byte) (ChangeG[V], error) {
	val := ChangeSerialized{}
	if err := json.Unmarshal(value, &val); err != nil {
		return ChangeG[V]{}, err
	}
	return changeSerToChangeG(&val, s.ValJSONSerde)
}

type ChangeGMsgpSerdeG[V any] struct {
	DefaultMsgpSerdeG[V]
	ValMsgpSerde SerdeG[V]
}

var _ = SerdeG[ChangeG[int]](ChangeGMsgpSerdeG[int]{})

func (s ChangeGMsgpSerdeG[V]) String() string {
	return fmt.Sprintf("ChangeGMsgpSerdeG{val: %s}", s.ValMsgpSerde.String())
}

func (s ChangeGMsgpSerdeG[V]) Encode(value ChangeG[V]) ([]byte, *[]byte, error) {
	c, newBuf, oldBuf, err := changeGToChangeSer(value, s.ValMsgpSerde)
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
	b := PopBuffer(c.Msgsize())
	buf := *b
	r, err := c.MarshalMsg(buf[:0])
	return r, b, err
}

func (s ChangeGMsgpSerdeG[V]) Decode(value []byte) (ChangeG[V], error) {
	val := ChangeSerialized{}
	if _, err := val.UnmarshalMsg(value); err != nil {
		return ChangeG[V]{}, err
	}
	return changeSerToChangeG(&val, s.ValMsgpSerde)
}

func GetChangeGSerdeG[V any](serdeFormat SerdeFormat, valSerde SerdeG[V]) (SerdeG[ChangeG[V]], error) {
	if serdeFormat == JSON {
		return &ChangeGJSONSerdeG[V]{
			ValJSONSerde: valSerde,
		}, nil
	} else if serdeFormat == MSGP {
		return &ChangeGMsgpSerdeG[V]{
			ValMsgpSerde: valSerde,
		}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
