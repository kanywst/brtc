package calc

import (
	"math"
	"math/big"
	"unicode"
)

type EntropyResult struct {
	CharSpace    int
	Length       int
	Entropy      float64
	Combinations *big.Int
}

// FromGuesses builds an EntropyResult from an externally supplied guess
// count (typically the `guesses` field of a zxcvbn report). CharSpace
// is left at 0 to signal "external estimate, no character-class
// inference"; downstream code that depends on CharSpace (e.g.
// MaxLengthForBudget) should treat 0 as "skip".
func FromGuesses(guesses *big.Int, password string) EntropyResult {
	length := len([]rune(password))
	return EntropyResult{
		CharSpace:    0,
		Length:       length,
		Entropy:      log2BigInt(guesses),
		Combinations: guesses,
	}
}

// log2BigInt computes log2(n) for arbitrarily large positive integers
// without overflowing float64. For n that fits in a float64, it uses
// the direct math.Log2; for larger values it falls back to the bit
// length, which is exact for powers of two and within 1 bit otherwise.
func log2BigInt(n *big.Int) float64 {
	if n == nil || n.Sign() <= 0 {
		return 0
	}
	f, _ := new(big.Float).SetInt(n).Float64()
	if math.IsInf(f, 1) || f <= 0 {
		return float64(n.BitLen() - 1)
	}
	return math.Log2(f)
}

func Analyze(password string) EntropyResult {
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, c := range password {
		switch {
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c) || unicode.IsSpace(c):
			hasSpecial = true
		default:
			hasSpecial = true
		}
	}

	charSpace := 0
	if hasLower {
		charSpace += 26
	}
	if hasUpper {
		charSpace += 26
	}
	if hasDigit {
		charSpace += 10
	}
	if hasSpecial {
		charSpace += 32
	}

	if charSpace == 0 {
		charSpace = 1
	}

	// Calculate length based on runes to properly handle multi-byte characters
	length := len([]rune(password))

	// E = L * log2(R)
	entropy := float64(length) * math.Log2(float64(charSpace))

	// C = R^L
	combinations := new(big.Int).Exp(big.NewInt(int64(charSpace)), big.NewInt(int64(length)), nil)

	return EntropyResult{
		CharSpace:    charSpace,
		Length:       length,
		Entropy:      entropy,
		Combinations: combinations,
	}
}
