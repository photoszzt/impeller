package store

import (
	"bytes"
	"math"
	"os"
	"reflect"
	"sharedlog-stream/pkg/commtypes"
	"sharedlog-stream/pkg/debug"
	"testing"
)

func TestUpperBoundWithLargeTimestamps(t *testing.T) {
	wks := &WindowKeySchema{}
	upper, err := wks.UpperRange([]byte{0xA, 0xB, 0xC}, math.MaxInt64)
	checkErr(err, t)
	shorter, err := wks.ToStoreKeyBinary([]byte{0xA}, math.MaxInt64, math.MaxInt32)
	checkErr(err, t)
	ret := bytes.Compare(upper, shorter)
	debug.Fprintf(os.Stderr, "compare out: %d\n", ret)
	if ret < 0 {
		debug.Fprint(os.Stderr, "upper: \n")
		debug.PrintByteSlice(upper)
		debug.Fprintf(os.Stderr, "len: %v\nshort: ", len(upper))
		debug.PrintByteSlice(shorter)
		debug.Fprintf(os.Stderr, "len: %v\n", len(shorter))
		t.Fatal("shorter key with max ts should be in range")
	}

	shorter, err = wks.ToStoreKeyBinary([]byte{0xA, 0xB}, math.MaxInt64, math.MaxInt32)
	checkErr(err, t)
	if ret := bytes.Compare(upper, shorter); ret < 0 {
		t.Fatal("shorter key with max ts should be in range")
	}

	eq, err := wks.ToStoreKeyBinary([]byte{0xA}, math.MaxInt64, math.MaxInt32)
	checkErr(err, t)
	if ret := bytes.Compare(upper, eq); ret != 0 {
		t.Fatal("should be equal")
	}
}

func TestUpperBoundWithKeyBytesLargerThanFirstTsByte(t *testing.T) {
	wks := &WindowKeySchema{}
	upper, err := wks.UpperRange([]byte{0xA, 0x8F, 0x9F}, math.MaxInt64)
	checkErr(err, t)
	shorter, err := wks.ToStoreKeyBinary([]byte{0xA, 0x8F}, math.MaxInt64, math.MaxInt32)
	checkErr(err, t)
	if ret := bytes.Compare(upper, shorter); ret < 0 {
		debug.Fprint(os.Stderr, "upper: \n")
		debug.PrintByteSlice(upper)
		debug.Fprintf(os.Stderr, "len: %v\nshort: ", len(upper))
		debug.PrintByteSlice(shorter)
		debug.Fprintf(os.Stderr, "len: %v\n", len(shorter))
		t.Fatal("shorter key with max ts should be in range")
	}
	eq, err := wks.ToStoreKeyBinary([]byte{0xA, 0x8F, 0x9F}, math.MaxInt64, math.MaxInt32)
	checkErr(err, t)
	if ret := bytes.Compare(upper, eq); ret != 0 {
		t.Fatal("should be equal")
	}
}

func TestUpperBoundWithKeyBytesLargerAndSmallerThanFirstTimestampByte(t *testing.T) {
	wks := &WindowKeySchema{}
	upper, err := wks.UpperRange([]byte{0xC, 0xC, 0x9}, 0x0Affffffffffffff)
	checkErr(err, t)
	shorter, err := wks.ToStoreKeyBinary(
		[]byte{0xC, 0xC},
		0x0Affffffffffffff,
		math.MaxInt32,
	)
	checkErr(err, t)
	if ret := bytes.Compare(upper, shorter); ret < 0 {
		debug.Fprint(os.Stderr, "TestUpperBoundWithKeyBytesLargerAndSmallerThanFirstTimestampByte\n")
		debug.Fprint(os.Stderr, "upper: \n")
		debug.PrintByteSlice(upper)
		debug.Fprintf(os.Stderr, "len: %v\nshort: ", len(upper))
		debug.PrintByteSlice(shorter)
		debug.Fprintf(os.Stderr, "len: %v\n", len(shorter))
		t.Fatal("shorter key with max ts should be in range")
	}
	eq, err := wks.ToStoreKeyBinary([]byte{0xC, 0xC}, 0x0Affffffffffffff, math.MaxInt32)
	checkErr(err, t)
	if ret := bytes.Compare(upper, eq); ret != 0 {
		t.Fatal("should be equal")
	}
}

func TestUpperBoundWithZeroTimestamp(t *testing.T) {
	wks := &WindowKeySchema{}
	upper, err := wks.UpperRange([]byte{0xA, 0xB, 0xC}, 0)
	checkErr(err, t)
	eq, err := wks.ToStoreKeyBinary([]byte{0xA, 0xB, 0xC}, 0, math.MaxInt32)
	checkErr(err, t)
	if ret := bytes.Compare(upper, eq); ret != 0 {
		t.Fatal("should be equal")
	}
}

func TestLowerBoundWithZeroTimestamp(t *testing.T) {
	wks := &WindowKeySchema{}
	lower := wks.LowerRange([]byte{0xA, 0xB, 0xC}, 0)
	eq, err := wks.ToStoreKeyBinary([]byte{0xA, 0xB, 0xC}, 0, 0)
	checkErr(err, t)
	if ret := bytes.Compare(lower, eq); ret != 0 {
		debug.Fprint(os.Stderr, "TestLowerBoundWithZeroTimestamp\nlower: \n")
		debug.PrintByteSlice(lower)
		debug.Fprintf(os.Stderr, "len: %d\neq: ", len(lower))
		debug.PrintByteSlice(eq)
		debug.Fprintf(os.Stderr, "len: %d\n", len(eq))
		t.Fatal("should be equal")
	}
}

func TestLowerBoundWithMonZeroTimestamp(t *testing.T) {
	wks := &WindowKeySchema{}
	lower := wks.LowerRange([]byte{0xA, 0xB, 0xC}, 42)
	eq, err := wks.ToStoreKeyBinary([]byte{0xA, 0xB, 0xC}, 0, 0)
	checkErr(err, t)
	if ret := bytes.Compare(lower, eq); ret != 0 {
		t.Fatal("should be equal")
	}
}

func TestLowerBoundMatchesTrailingZeros(t *testing.T) {
	wks := &WindowKeySchema{}
	lower := wks.LowerRange([]byte{0xA, 0xB, 0xC}, math.MaxInt64-1)
	low, err := wks.ToStoreKeyBinary([]byte{0xA, 0xB, 0xC, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, math.MaxInt64-1, 0)
	checkErr(err, t)
	if ret := bytes.Compare(lower, low); ret >= 0 {
		t.Fatal("should be lower")
	}
	eq, err := wks.ToStoreKeyBinary([]byte{0xA, 0xB, 0xC}, 0, 0)
	checkErr(err, t)
	if ret := bytes.Compare(lower, eq); ret != 0 {
		t.Fatal("should be equal")
	}
}

func TestShouldExtractKeyBytesFromBinary(t *testing.T) {
	wks := &WindowKeySchema{}
	key := "a"
	s := commtypes.StringSerde{}
	kBytes, _, err := s.Encode(key)
	if err != nil {
		t.Fatal(err.Error())
	}
	serialized, err := wks.ToStoreKeyBinary(kBytes, 50, 3)
	checkErr(err, t)
	extract := wks.ExtractStoreKeyBytes(serialized)
	if !reflect.DeepEqual(kBytes, extract) {
		t.Fatal("should equal")
	}
}

func TestShouldExtractStartTimeFromBinary(t *testing.T) {
	wks := &WindowKeySchema{}
	key := "a"
	s := commtypes.StringSerde{}
	kBytes, _, err := s.Encode(key)
	if err != nil {
		t.Fatal(err.Error())
	}
	serialized, err := wks.ToStoreKeyBinary(kBytes, 50, 3)
	checkErr(err, t)
	extractTs := wks.ExtractStoreTs(serialized)
	if extractTs != 50 {
		t.Fatal("should equal")
	}
}
