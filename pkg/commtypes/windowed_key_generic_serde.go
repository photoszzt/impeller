package commtypes

import (
	"encoding/json"
	"fmt"
)

type WindowedKeyJSONSerdeG struct {
	DefaultJSONSerde
	KeyJSONSerde    Serde
	WindowJSONSerde Serde
}

var _ = SerdeG[WindowedKey](WindowedKeyJSONSerdeG{})

func (s WindowedKeyJSONSerdeG) String() string {
	return fmt.Sprintf("WindowedKeyJSONSerdeG{key: %s, win: %s}", s.KeyJSONSerde.String(), s.WindowJSONSerde.String())
}

func (s WindowedKeyJSONSerdeG) Encode(value WindowedKey) ([]byte, *[]byte, error) {
	wk, kbuf, wbuf, err := winKeyToWindowedKeySer(value, s.KeyJSONSerde, s.WindowJSONSerde)
	defer func() {
		if wk != nil {
			if s.KeyJSONSerde.UsedBufferPool() && wk.KeySerialized != nil {
				*kbuf = wk.KeySerialized
				PushBuffer(kbuf)
			}
			if s.WindowJSONSerde.UsedBufferPool() && wk.WindowSerialized != nil {
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

func (s WindowedKeyJSONSerdeG) Decode(value []byte) (WindowedKey, error) {
	wk := WindowedKeySerialized{}
	if err := json.Unmarshal(value, &wk); err != nil {
		return WindowedKey{}, err
	}
	return winKeySerToWindowedKey(&wk, s.KeyJSONSerde, s.WindowJSONSerde)
}

type WindowedKeyMsgpSerdeG struct {
	DefaultMsgpSerde
	KeyMsgpSerde    Serde
	WindowMsgpSerde Serde
}

var _ = SerdeG[WindowedKey](WindowedKeyMsgpSerdeG{})

func (s WindowedKeyMsgpSerdeG) String() string {
	return fmt.Sprintf("WindowedKeyMsgpSerdeG{key: %s, win: %s}", s.KeyMsgpSerde.String(), s.WindowMsgpSerde.String())
}

func (s WindowedKeyMsgpSerdeG) Encode(value WindowedKey) ([]byte, *[]byte, error) {
	wk, kbuf, wbuf, err := winKeyToWindowedKeySer(value, s.KeyMsgpSerde, s.WindowMsgpSerde)
	defer func() {
		if wk != nil {
			if s.KeyMsgpSerde.UsedBufferPool() && wk.KeySerialized != nil {
				*kbuf = wk.KeySerialized
				PushBuffer(kbuf)
			}
			if s.WindowMsgpSerde.UsedBufferPool() && wk.WindowSerialized != nil {
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
	b := PopBuffer(wk.Msgsize())
	buf := *b
	r, err := wk.MarshalMsg(buf[:0])
	return r, b, err
}

func (s WindowedKeyMsgpSerdeG) Decode(value []byte) (WindowedKey, error) {
	wk := WindowedKeySerialized{}
	_, err := wk.UnmarshalMsg(value)
	if err != nil {
		return WindowedKey{}, err
	}
	return winKeySerToWindowedKey(&wk, s.KeyMsgpSerde, s.WindowMsgpSerde)
}
