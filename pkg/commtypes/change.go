//go:generate msgp
//msgp:ignore Change ChangeJSONSerde ChangeMsgpSerde
package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/common_errors"
	"sharedlog-stream/pkg/utils"
)

type Change struct {
	NewVal interface{}
	OldVal interface{}
}

func (c Change) String() string {
	return fmt.Sprintf("Change: {NewVal: %v, OldVal: %v}", c.NewVal, c.OldVal)
}

type ChangeSerialized struct {
	NewValSerialized []byte `json:"nValSer,omitempty" msg:"nValSer,omitempty"`
	OldValSerialized []byte `json:"oValSer,omitempty" msg:"oValSer,omitempty"`
}

func CastToChangePtr(value interface{}) *Change {
	v, ok := value.(*Change)
	if !ok {
		vtmp := value.(Change)
		v = &vtmp
	}
	return v
}

func convertToChangeSer(value interface{}, valSerde Serde) (c *ChangeSerialized, newBuf *[]byte, oldBuf *[]byte, err error) {
	if value == nil {
		return nil, nil, nil, nil
	}
	val := CastToChangePtr(value)
	if val == nil {
		return nil, nil, nil, nil
	}
	var newValEnc, oldValEnc []byte
	if !utils.IsNil(val.NewVal) {
		newValEnc, newBuf, err = valSerde.Encode(val.NewVal)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if !utils.IsNil(val.OldVal) {
		oldValEnc, oldBuf, err = valSerde.Encode(val.OldVal)
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

func changeToChangeSer(value Change, valSerde Serde) (c *ChangeSerialized, newBuf *[]byte, oldBuf *[]byte, err error) {
	var newValEnc, oldValEnc []byte
	if !utils.IsNil(value.NewVal) {
		newValEnc, newBuf, err = valSerde.Encode(value.NewVal)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if !utils.IsNil(value.OldVal) {
		oldValEnc, oldBuf, err = valSerde.Encode(value.OldVal)
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

func decodeToChange(val *ChangeSerialized, valSerde Serde) (interface{}, error) {
	return changeSerToChange(val, valSerde)
}

func changeSerToChange(val *ChangeSerialized, valSerde Serde) (Change, error) {
	var newVal, oldVal interface{} = nil, nil
	var err error
	if val.NewValSerialized != nil {
		newVal, err = valSerde.Decode(val.NewValSerialized)
		if err != nil {
			return Change{}, err
		}
	}
	if val.OldValSerialized != nil {
		oldVal, err = valSerde.Decode(val.OldValSerialized)
		if err != nil {
			return Change{}, err
		}
	}
	return Change{
		NewVal: newVal,
		OldVal: oldVal,
	}, nil
}

type ChangeJSONSerde struct {
	DefaultJSONSerde
	ValJSONSerde Serde
}

var _ = Serde(ChangeJSONSerde{})

func (s ChangeJSONSerde) String() string {
	return fmt.Sprintf("ChangeJSONSerde{val: %s}", s.ValJSONSerde.String())
}

func (s ChangeJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	c, newBuf, oldBuf, err := convertToChangeSer(value, s.ValJSONSerde)
	defer func() {
		if s.ValJSONSerde.UsedBufferPool() && c != nil {
			if c.NewValSerialized != nil && newBuf != nil {
				*newBuf = c.NewValSerialized
				PushBuffer(newBuf)
			}
			if c.OldValSerialized != nil && oldBuf != nil {
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

func (s ChangeJSONSerde) Decode(value []byte) (interface{}, error) {
	val := ChangeSerialized{}
	if err := json.Unmarshal(value, &val); err != nil {
		return nil, err
	}
	return decodeToChange(&val, s.ValJSONSerde)
}

type ChangeMsgpSerde struct {
	DefaultMsgpSerde
	ValMsgpSerde Serde
}

var _ = Serde(ChangeMsgpSerde{})

func (s ChangeMsgpSerde) String() string {
	return fmt.Sprintf("ChangeMsgpSerde{val: %s}", s.ValMsgpSerde.String())
}

func (s ChangeMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	c, newBuf, oldBuf, err := convertToChangeSer(value, s.ValMsgpSerde)
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

func (s ChangeMsgpSerde) Decode(value []byte) (interface{}, error) {
	val := ChangeSerialized{}
	if _, err := val.UnmarshalMsg(value); err != nil {
		return nil, err
	}
	return decodeToChange(&val, s.ValMsgpSerde)
}

func GetChangeSerde(serdeFormat SerdeFormat, valSerde Serde) (Serde, error) {
	if serdeFormat == JSON {
		return &ChangeJSONSerde{
			ValJSONSerde: valSerde,
		}, nil
	} else if serdeFormat == MSGP {
		return &ChangeMsgpSerde{
			ValMsgpSerde: valSerde,
		}, nil
	} else {
		return nil, common_errors.ErrUnrecognizedSerdeFormat
	}
}
