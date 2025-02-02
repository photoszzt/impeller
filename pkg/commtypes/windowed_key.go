//go:generate msgp
//msgp:ignore WindowedKey WindowedKeyJSONSerde WindowedKeyMsgpSerde
package commtypes

import (
	"encoding/json"
	"fmt"
	"sharedlog-stream/pkg/utils"
)

type WindowedKey struct {
	Key    interface{}
	Window Window
}

type WindowedKeySerialized struct {
	KeySerialized    []byte `json:"ks" msg:"ks"`
	WindowSerialized []byte `json:"ws" msg:"ws"`
}

func convertToWindowedKeySer(value interface{}, keySerde Serde, windowSerde Serde) (wk *WindowedKeySerialized, kbuf, wbuf *[]byte, err error) {
	if value == nil {
		return nil, nil, nil, nil
	}
	v, ok := value.(*WindowedKey)
	if !ok {
		vtmp := value.(WindowedKey)
		v = &vtmp
	}
	if v == nil {
		return nil, nil, nil, nil
	}
	var kenc, wenc []byte = nil, nil
	if !utils.IsNil(v.Key) {
		kenc, kbuf, err = keySerde.Encode(v.Key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if !utils.IsNil(v.Window) {
		wenc, wbuf, err = windowSerde.Encode(v.Window)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if kenc == nil && wenc == nil {
		return nil, nil, nil, nil
	}
	wk = &WindowedKeySerialized{
		KeySerialized:    kenc,
		WindowSerialized: wenc,
	}
	return wk, kbuf, wbuf, nil
}

func winKeyToWindowedKeySer(value WindowedKey, keySerde Serde, windowSerde Serde) (wk *WindowedKeySerialized, kbuf, wbuf *[]byte, err error) {
	var kenc, wenc []byte = nil, nil
	if !utils.IsNil(value.Key) {
		kenc, kbuf, err = keySerde.Encode(value.Key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if !utils.IsNil(value.Window) {
		wenc, wbuf, err = windowSerde.Encode(value.Window)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if kenc == nil && wenc == nil {
		return nil, nil, nil, nil
	}
	wk = &WindowedKeySerialized{
		KeySerialized:    kenc,
		WindowSerialized: wenc,
	}
	return wk, kbuf, wbuf, nil
}

func decodeToWindowedKey(wkSer *WindowedKeySerialized, keySerde Serde, windowSerde Serde) (interface{}, error) {
	var err error
	var k, w interface{}
	if wkSer.KeySerialized != nil {
		k, err = keySerde.Decode(wkSer.KeySerialized)
		if err != nil {
			return nil, err
		}
	}
	if wkSer.WindowSerialized != nil {
		w, err = windowSerde.Decode(wkSer.WindowSerialized)
		if err != nil {
			return nil, err
		}
	}
	return WindowedKey{
		Key:    k,
		Window: w.(Window),
	}, nil
}

func winKeySerToWindowedKey(wkSer *WindowedKeySerialized, keySerde Serde, windowSerde Serde) (WindowedKey, error) {
	var err error
	var k, w interface{}
	if wkSer.KeySerialized != nil {
		k, err = keySerde.Decode(wkSer.KeySerialized)
		if err != nil {
			return WindowedKey{}, err
		}
	}
	if wkSer.WindowSerialized != nil {
		w, err = windowSerde.Decode(wkSer.WindowSerialized)
		if err != nil {
			return WindowedKey{}, err
		}
	}
	return WindowedKey{
		Key:    k,
		Window: w.(Window),
	}, nil
}

type WindowedKeyJSONSerde struct {
	DefaultJSONSerde
	KeyJSONSerde    Serde
	WindowJSONSerde Serde
}

var _ = Serde(WindowedKeyJSONSerde{})

func (s WindowedKeyJSONSerde) String() string {
	return fmt.Sprintf("WindowedKeyJSONSerde{key: %s, win: %s}", s.KeyJSONSerde.String(), s.WindowJSONSerde.String())
}

func (s WindowedKeyJSONSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	wk, kbuf, wbuf, err := convertToWindowedKeySer(value, s.KeyJSONSerde, s.WindowJSONSerde)
	defer func() {
		if wk != nil {
			if s.KeyJSONSerde.UsedBufferPool() && kbuf != nil {
				*kbuf = wk.KeySerialized
				PushBuffer(kbuf)
			}
			if s.WindowJSONSerde.UsedBufferPool() && wbuf != nil {
				*wbuf = wk.WindowSerialized
				PushBuffer(wbuf)
			}
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if wk == nil {
		return nil, nil, nil
	}
	r, err := json.Marshal(&wk)
	return r, nil, err
}

func (s WindowedKeyJSONSerde) Decode(value []byte) (interface{}, error) {
	wk := WindowedKeySerialized{}
	if err := json.Unmarshal(value, &wk); err != nil {
		return nil, err
	}
	return decodeToWindowedKey(&wk, s.KeyJSONSerde, s.WindowJSONSerde)
}

type WindowedKeyMsgpSerde struct {
	DefaultMsgpSerde
	KeyMsgpSerde    Serde
	WindowMsgpSerde Serde
}

var _ = Serde(WindowedKeyMsgpSerde{})

func (s WindowedKeyMsgpSerde) String() string {
	return fmt.Sprintf("WindowedKeyMsgpSerde{key: %s, win: %s}", s.KeyMsgpSerde.String(), s.WindowMsgpSerde.String())
}

func (s WindowedKeyMsgpSerde) Encode(value interface{}) ([]byte, *[]byte, error) {
	wk, kbuf, wbuf, err := convertToWindowedKeySer(value, s.KeyMsgpSerde, s.WindowMsgpSerde)
	defer func() {
		if wk != nil {
			if s.KeyMsgpSerde.UsedBufferPool() && kbuf != nil {
				*kbuf = wk.KeySerialized
				PushBuffer(&wk.KeySerialized)
			}
			if s.WindowMsgpSerde.UsedBufferPool() && wbuf != nil {
				*wbuf = wk.WindowSerialized
				PushBuffer(&wk.WindowSerialized)
			}
		}
	}()
	if err != nil {
		return nil, nil, err
	}
	if wk == nil {
		return nil, nil, nil
	}
	b := PopBuffer(wk.Msgsize())
	buf := *b
	r, err := wk.MarshalMsg(buf[:0])
	return r, b, err
}

func (s WindowedKeyMsgpSerde) Decode(value []byte) (interface{}, error) {
	wk := WindowedKeySerialized{}
	_, err := wk.UnmarshalMsg(value)
	if err != nil {
		return nil, err
	}
	return decodeToWindowedKey(&wk, s.KeyMsgpSerde, s.WindowMsgpSerde)
}
