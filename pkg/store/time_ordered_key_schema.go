package store

import (
	"bytes"
	"encoding/binary"
)

const (
	ts_size     = 8
	seqnum_size = 4
)

type TimeOrderedKeySchema struct {
}

var _ = KeySchema(&TimeOrderedKeySchema{})

func (ks *TimeOrderedKeySchema) UpperRange(key []byte, to int64) ([]byte, error) {
	panic("not supported")
}
func (ks *TimeOrderedKeySchema) LowerRange(key []byte, from int64) []byte {
	panic("not supported")
}

func (ks *TimeOrderedKeySchema) ToStoreBinaryKeyPrefix(key []byte, ts int64) ([]byte, error) {
	return ks.toStoreKeyBinaryPrefix(key, ts)
}

func (ks *TimeOrderedKeySchema) UpperRangeFixedSize(key []byte, to int64) ([]byte, error) {
	panic("not supported")
}

func (ks *TimeOrderedKeySchema) LowerRangeFixedSize(key []byte, from int64) ([]byte, error) {
	panic("not supported")
}

func (ks *TimeOrderedKeySchema) SegmentTimestamp(key []byte) int64 {
	return ks.extractStoreTs(key)
}

func (ks *TimeOrderedKeySchema) HasNextCondition(curKey []byte,
	binaryKeyFrom []byte, binaryKeyTo []byte,
	from int64, to int64,
) (bool, int64) {
	panic("not implemented")
}

func (ks *TimeOrderedKeySchema) toStoreKeyBinaryPrefix(key []byte, ts int64) ([]byte, error) {
	buf := make([]byte, 0, len(key)+ts_size)
	ts_buf := make([]byte, 0, ts_size)
	binary.BigEndian.PutUint64(ts_buf, uint64(ts))
	buffer := bytes.NewBuffer(buf)
	buffer.Write(ts_buf)
	buffer.Write(key)
	return buffer.Bytes(), nil
}

/*
func (ks *TimeOrderedKeySchema) toStoreKeyBinary(key []byte, ts int64, seqnum int) ([]byte, error) {
	buf := make([]byte, 0, ts_size+len(key)+seqnum_size)
	ts_buf := make([]byte, 0, ts_size)
	binary.BigEndian.PutUint64(ts_buf, uint64(ts))
	seqnum_buf := make([]byte, 0, seqnum_size)
	binary.BigEndian.PutUint32(seqnum_buf, uint32(seqnum))
	buffer := bytes.NewBuffer(buf)
	buffer.Write(ts_buf)
	buffer.Write(key)
	buffer.Write(seqnum_buf)
	return buffer.Bytes(), nil
}
*/

func (ks *TimeOrderedKeySchema) extractStoreTs(key []byte) int64 {
	return int64(binary.BigEndian.Uint64(key))
}

func (ks *TimeOrderedKeySchema) ExtractStoreKeyBytes(key []byte) []byte {
	panic("not implemented")
}
