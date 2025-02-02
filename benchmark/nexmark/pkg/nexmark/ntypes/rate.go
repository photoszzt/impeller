package ntypes

import (
	"fmt"
	"math"

	"github.com/rs/zerolog/log"
)

// / Number of us per unit
type RateUnit uint32

const (
	PER_SECOND        RateUnit = 1_000_000
	PER_MINUTE        RateUnit = 60_000_000
	UNKNOWN_RATE_UNIT RateUnit = 0
)

// / Number of us between events at given rate
func (ru RateUnit) RateToPeriodUs(rate uint64) uint64 {
	return (uint64(ru) + rate/2) / rate
}

type RateShape uint8

const (
	SQUARE RateShape = iota
	SINE
	UNKNOWN_RATE_SHAPE
	NUM_STEPS uint32 = 10
)

func StrToRateShape(rateShape string) (RateShape, error) {
	if rateShape == "square" || rateShape == "SQUARE" {
		return SQUARE, nil
	} else if rateShape == "sine" || rateShape == "SINE" {
		return SINE, nil
	} else {
		return UNKNOWN_RATE_SHAPE, fmt.Errorf("unknown rate shape: %v", rateShape)
	}
}

// / Return inter-event  delay, in us, for each generator to follow in order
// / to achieve `rate` at `unit` using `numGenerators`.
func (rs RateShape) InterEventDelayUs(rate uint64, unit RateUnit, numGenerators uint32) uint64 {
	return unit.RateToPeriodUs(rate) * uint64(numGenerators)
}

// / Return array of seccessive inter-event delays, in us, for each generator to follow in order to
// / achieve this shape with `firatRate/nextRate` at `unit` using `numGenerators`.
func (rs RateShape) InterEventDelayUsArr(firstRate uint64, secondRate uint64, unit RateUnit, numGenerators uint32) ([]uint64, error) {
	if firstRate == secondRate {
		ret := make([]uint64, 1)
		ret[0] = unit.RateToPeriodUs(firstRate) * uint64(numGenerators)
		return ret, nil
	}

	switch rs {
	case SQUARE:
		ret := make([]uint64, 2)
		ret[0] = unit.RateToPeriodUs(firstRate) * uint64(numGenerators)
		ret[1] = unit.RateToPeriodUs(secondRate) * uint64(numGenerators)
		return ret, nil
	case SINE:
		mid := float64(firstRate+secondRate) / 2.0
		amp := float64(firstRate-secondRate) / 2.0
		ret := make([]uint64, NUM_STEPS)
		for i := uint32(0); i < NUM_STEPS; i++ {
			r := (2.0 * math.Pi * float64(i)) / float64(NUM_STEPS)
			rate := mid + amp*math.Cos(r)
			ret[i] = unit.RateToPeriodUs(uint64(math.Round(rate))) * uint64(numGenerators)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("unknown rate shape %v", rs)
	}
}

func (rs RateShape) StepLengthSec(ratePerSec uint32) uint32 {
	n := uint32(0)
	switch rs {
	case SQUARE:
		n = 2
	case SINE:
		n = NUM_STEPS
	default:
		log.Fatal().Uint32("RateShape", uint32(rs)).Msg("Unknown rate shape")
	}
	return (ratePerSec + n - 1) / n
}
