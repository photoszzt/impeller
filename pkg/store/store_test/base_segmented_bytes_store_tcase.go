package store_test

import (
	"context"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/store"
	"testing"
)

const (
	windowSizeForTimeWindow = 500
	// retention               = 1000
	// segmentInterval = 60_000
)

func getSerde() (commtypes.Serde, commtypes.Serde) {
	kSerde := commtypes.WindowedKeyJSONSerde{
		KeyJSONSerde:    commtypes.StringSerde{},
		WindowJSONSerde: commtypes.TimeWindowJSONSerde{},
	}
	vSerde := commtypes.IntSerde{}
	return kSerde, vSerde
}

func getWindows(t testing.TB) []*commtypes.TimeWindow {
	windows := make([]*commtypes.TimeWindow, 0)
	w, err := commtypes.TimeWindowForSize(10, windowSizeForTimeWindow)
	if err != nil {
		t.Fatal(err.Error())
	}
	windows = append(windows, w)
	w, err = commtypes.TimeWindowForSize(500, windowSizeForTimeWindow)
	if err != nil {
		t.Fatal(err.Error())
	}
	windows = append(windows, w)
	w, err = commtypes.TimeWindowForSize(1_000, windowSizeForTimeWindow)
	if err != nil {
		t.Fatal(err.Error())
	}
	windows = append(windows, w)
	w, err = commtypes.TimeWindowForSize(60_000, windowSizeForTimeWindow)
	if err != nil {
		t.Fatal(err.Error())
	}
	windows = append(windows, w)
	return windows
}

/*
func getNextSegmentWindow(t testing.TB) *processor.TimeWindow {
	w, err := processor.TimeWindowForSize(segmentInterval+retention, windowSizeForTimeWindow)
	if err != nil {
		t.Fatal(err.Error())
	}
	return w
}
*/

func putKV(ctx context.Context, key string, window *commtypes.TimeWindow, value int, kSerde commtypes.Serde,
	vSerde commtypes.Serde, byteStore store.SegmentedBytesStore, t testing.TB,
) {
	wk := commtypes.WindowedKey{
		Key:    key,
		Window: window,
	}
	wkBytes, _, err := kSerde.Encode(wk)
	if err != nil {
		t.Fatal(err.Error())
	}
	vBytes, _, err := vSerde.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	err = byteStore.Put(ctx, wkBytes, vBytes)
	if err != nil {
		t.Fatal(err.Error())
	}
}

type KeyValue struct {
	Key interface{}
	Val interface{}
}

func ShouldPutAndFetch(ctx context.Context, byteStore store.SegmentedBytesStore, t testing.TB) {
	key := "a"
	kSerde, vSerde := getSerde()

	windows := getWindows(t)
	putKV(ctx, key, windows[0], 10, kSerde, vSerde, byteStore, t)
	putKV(ctx, key, windows[1], 50, kSerde, vSerde, byteStore, t)
	putKV(ctx, key, windows[2], 100, kSerde, vSerde, byteStore, t)

	strSerde := commtypes.StringSerde{}
	kBytes, _, err := strSerde.Encode(key)
	if err != nil {
		t.Fatal(err.Error())
	}
	kv := make([]*KeyValue, 0)
	err = byteStore.Fetch(ctx, kBytes, 1, 999, func(i int64, kt []byte, vt []byte) error {
		w, err := commtypes.NewTimeWindow(i, i+windowSizeForTimeWindow)
		if err != nil {
			return err
		}
		wk := commtypes.WindowedKey{
			Key:    kt,
			Window: w,
		}
		kv = append(kv, &KeyValue{
			Key: wk,
			Val: vt,
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
