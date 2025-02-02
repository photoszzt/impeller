package generator

import (
	"math"
	"math/rand"
)

func NextPrice(random *rand.Rand) uint64 {
	return uint64(math.Pow(10.0, rand.Float64()*6.0) * 100.0)
}
