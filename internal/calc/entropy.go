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
