package generator

import (
	"math/rand"
	"strings"
)

const (
	MIN_STRING_LENGTH uint32 = 3
	letterBytes              = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits            = 6                    // 6 bits to represent a letter index
	letterIdxMask            = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax             = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func NextExactString(rand *rand.Rand, length uint32) string {
	var b strings.Builder
	b.Grow(int(length))
	for i, cache, remain := int(length-1), rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b.String()
}

func NextString(rand *rand.Rand, maxLength uint32) string {
	length := MIN_STRING_LENGTH + uint32(rand.Intn(int(maxLength-MIN_STRING_LENGTH)))
	return NextExactString(rand, length)
}

func NextExtra(random *rand.Rand, currentSize, desiredAverageSize uint32) string {
	if currentSize > desiredAverageSize {
		return ""
	}
	desiredAverageSize -= currentSize
	delta := uint32(float32(desiredAverageSize) * 0.2)
	minSize := desiredAverageSize - delta
	additional := 0
	if delta != 0 {
		additional = rand.Intn(int(2 * delta))
	}
	desiredSize := minSize + uint32(additional)
	return NextExactString(random, desiredSize)
}
