package cost

import (
	"math"
	"strconv"
	"strings"
)

// ParseBudget converts a budget string (e.g. "1000usd") into a float64.
func ParseBudget(budgetStr string) (float64, error) {
	if budgetStr == "" {
		return 0, nil
	}

	// basic parsing, strip "usd" and "$"
	clean := strings.ToLower(strings.TrimSpace(budgetStr))
	clean = strings.ReplaceAll(clean, "usd", "")
	clean = strings.ReplaceAll(clean, "$", "")

	return strconv.ParseFloat(strings.TrimSpace(clean), 64)
}

// MaxLengthForBudget calculates how many characters of a given charSpace
// can be cracked within the given budget USD, assuming the specified
// algo, hardware, work factor, and (for argon2id) memory parameter.
//
// memoryMB is forwarded verbatim to CalculateHashRate; pass 0 to use
// the YAML baseline.
func MaxLengthForBudget(budgetUSD float64, hw, algo string, workFactor, memoryMB, charSpace int) int {
	if budgetUSD <= 0 || charSpace <= 1 {
		return 0
	}

	p := lookupProfile(hw)

	if p.CostPerHourUSD <= 0 {
		return 999 // Effectively infinite characters if hardware cost is $0
	}

	// Max Hours = Budget / CostPerHour
	maxHours := budgetUSD / p.CostPerHourUSD
	maxSeconds := maxHours * 3600.0

	hashRate := CalculateHashRate(hw, algo, workFactor, memoryMB)

	// T_avg = (R^L / 2) / H  =>  R^L / 2 = T_avg * H  =>  R^L = 2 * T_avg * H
	// L * log(R) = log(2 * T_avg * H)
	maxCombinations := 2.0 * maxSeconds * hashRate
	if maxCombinations <= 0 {
		return 0
	}

	// L = ln(maxCombinations) / ln(R)
	l := math.Log(maxCombinations) / math.Log(float64(charSpace))

	// Return the floor of L because adding one more character would exceed the budget.
	return int(math.Floor(l))
}

// MinLengthForTime returns the smallest password length over the given
// charSpace whose average crack time meets or exceeds thresholdSeconds on the
// specified hardware and algorithm. It inverts the time formula:
//
//	T_avg = (R^L / 2) / H  >=  threshold   =>   R^L >= 2 * threshold * H
//	L >= log_R(2 * threshold * H)
//
// It is the defender's actionable counterpart to a --fail-under-time gate:
// "use at least N characters to survive this attacker for this long".
func MinLengthForTime(thresholdSeconds float64, hw, algo string, workFactor, memoryMB, charSpace int) int {
	if thresholdSeconds <= 0 || charSpace <= 1 {
		return 0
	}

	hashRate := CalculateHashRate(hw, algo, workFactor, memoryMB)
	if hashRate <= 0 {
		return 1 // An attacker that cannot hash is held off by any password.
	}

	needed := 2.0 * thresholdSeconds * hashRate
	l := math.Ceil(math.Log(needed) / math.Log(float64(charSpace)))
	if l < 1 {
		return 1
	}
	return int(l)
}
