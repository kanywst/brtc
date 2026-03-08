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
// can be cracked within the given budget USD, assuming the specified algo and hardware.
func MaxLengthForBudget(budgetUSD float64, hw string, algo string, workFactor int, charSpace int) int {
	if budgetUSD <= 0 || charSpace <= 1 {
		return 0
	}

	p, ok := Profiles[strings.ToLower(hw)]
	if !ok {
		p = Profiles["rtx-4090"]
	}

	if p.CostPerHourUSD <= 0 {
		return 999 // Effectively infinite characters if hardware cost is $0
	}

	// Max Hours = Budget / CostPerHour
	maxHours := budgetUSD / p.CostPerHourUSD
	maxSeconds := maxHours * 3600.0

	hashRate := CalculateHashRate(hw, algo, workFactor)

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
