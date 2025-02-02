package commtypes

func SizeOfInt(k int) int64 {
	return 4
}

func SizeOfInt32(k int32) int64 {
	return 4
}

func SizeOfInt64(k int64) int64 {
	return 8
}

func SizeOfUint(k uint) int64 {
	return 4
}

func SizeOfUint32(k uint32) int64 {
	return 4
}

func SizeOfUint64(k uint64) int64 {
	return 8
}

func SizeOfFloat32(k float32) int64 {
	return 4
}

func SizeOfFloat64(k float64) int64 {
	return 8
}

func SizeOfString(k string) int64 {
	return int64(len(k))
}

func SizeOfByteSlice(k []byte) int64 {
	return int64(len(k))
}
