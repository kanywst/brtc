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

// UnlimitedBudgetChars is the sentinel MaxLengthForBudget returns when the
// hardware has no rental cost (owned hardware): no finite budget can bound
// the attacker, so any length is "affordable". It is math.MaxInt rather than
// a small number so it can never collide with a genuinely computed length.
// Callers must treat it as an infinity flag, not a literal character count.
const UnlimitedBudgetChars = math.MaxInt

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
		return UnlimitedBudgetChars // Owned hardware: no budget bounds the attacker.
	}

	// Max Hours = Budget / CostPerHour
	maxHours := budgetUSD / p.CostPerHourUSD
	maxSeconds := maxHours * 3600.0

	hashRate := CalculateHashRate(hw, algo, workFactor, memoryMB)
	if maxSeconds <= 0 || hashRate <= 0 {
		return 0
	}

	// T_avg = (R^L / 2) / H  =>  R^L = 2 * T_avg * H, so
	// L = log_R(2 * T_avg * H) = (ln2 + ln(T_avg) + ln(H)) / ln(R).
	// Summing the logs avoids overflowing the 2*T_avg*H product to +Inf for
	// an astronomically large budget, which would yield a garbage int cast.
	l := (math.Ln2 + math.Log(maxSeconds) + math.Log(hashRate)) / math.Log(float64(charSpace))

	// Floor of L because one more character would exceed the budget. A tiny
	// budget (or very slow attacker) yields l < 0; clamp to 0 so the caller
	// never sees a negative "max safe length".
	if floored := int(math.Floor(l)); floored > 0 {
		return floored
	}
	return 0
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
	// An extreme threshold can overflow needed to +Inf, making l Inf/NaN;
	// converting that to int yields a garbage (negative) value that would
	// wrongly read as a 1-char recommendation. Cap it at the sentinel.
	if math.IsInf(l, 1) || math.IsNaN(l) || l > maxRecommendedChars {
		return maxRecommendedChars
	}
	if l < 1 {
		return 1
	}
	return int(l)
}

// maxRecommendedChars caps MinLengthForTime when the requirement is so extreme
// that the float math overflows; it reads as "no practical length suffices".
const maxRecommendedChars = 999
