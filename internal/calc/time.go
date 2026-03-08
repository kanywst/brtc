package calc

import (
	"math"
	"math/big"
)

// TimeToCrack calculates T_avg = (C / 2) / H
// Returns time as float64 seconds.
func TimeToCrack(combinations *big.Int, hashRate float64) float64 {
	if hashRate <= 0 {
		return math.Inf(1)
	}

	// (C / 2)
	c := new(big.Float).SetInt(combinations)
	halfC := new(big.Float).Quo(c, big.NewFloat(2.0))

	// T_avg = halfC / hashRate
	tAvg := new(big.Float).Quo(halfC, big.NewFloat(hashRate))

	f64, _ := tAvg.Float64()
	return f64
}
