package generator

import (
	"math"
	"math/rand"
)

func NextUint64(random *rand.Rand, n uint64) uint64 {
	if n < math.MaxInt32 {
		return uint64(random.Intn(int(n)))
	} else {
		return random.Uint64() % n
	}
}

func NextInt64(random *rand.Rand, n int64) int64 {
	if n < math.MaxInt32 {
		return int64(random.Intn(int(n)))
	} else {
		return random.Int63n(n)
	}
}
