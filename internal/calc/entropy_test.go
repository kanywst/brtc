package calc

import (
	"math/big"
	"testing"
)

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedSpace int
		expectedLen   int
	}{
		{"lowercase only", "abcde", 26, 5},
		{"mixed case", "aBcDe", 52, 5},
		{"with digits", "aB123", 62, 5},
		{"with special", "aB1!@", 94, 5},
		{"empty", "", 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Analyze(tt.password)
			if res.CharSpace != tt.expectedSpace {
				t.Errorf("expected char space %d, got %d", tt.expectedSpace, res.CharSpace)
			}
			if res.Length != tt.expectedLen {
				t.Errorf("expected length %d, got %d", tt.expectedLen, res.Length)
			}

			// Check combinations R^L
			expectedCombos := new(big.Int).Exp(big.NewInt(int64(tt.expectedSpace)), big.NewInt(int64(tt.expectedLen)), nil)
			if res.Combinations.Cmp(expectedCombos) != 0 {
				t.Errorf("expected combinations %v, got %v", expectedCombos, res.Combinations)
			}
		})
	}
}

func TestFromGuesses(t *testing.T) {
	tests := []struct {
		name        string
		guesses     *big.Int
		password    string
		wantLen     int
		wantBitsApx float64 // approximate; tolerance 0.01
	}{
		{"zero guesses", big.NewInt(0), "anything", 8, 0},
		{"1024 = 10 bits exactly", big.NewInt(1024), "", 0, 10},
		{"1e10 ≈ 33.22 bits", big.NewInt(10_000_000_000), "P@ssw0rd!", 9, 33.22},
		{"unicode length is rune count", big.NewInt(100), "あいう", 3, 6.64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := FromGuesses(tt.guesses, tt.password)
			if res.CharSpace != 0 {
				t.Errorf("FromGuesses CharSpace should be 0 (external), got %d", res.CharSpace)
			}
			if res.Length != tt.wantLen {
				t.Errorf("Length = %d, want %d", res.Length, tt.wantLen)
			}
			if res.Combinations.Cmp(tt.guesses) != 0 {
				t.Errorf("Combinations = %v, want %v", res.Combinations, tt.guesses)
			}
			if diff := res.Entropy - tt.wantBitsApx; diff > 0.01 || diff < -0.01 {
				t.Errorf("Entropy bits = %v, want ~%v", res.Entropy, tt.wantBitsApx)
			}
		})
	}
}

func TestFromGuesses_HugeNumberDoesNotOverflow(t *testing.T) {
	// 2^2000: far beyond float64 range. Must fall back to bit length.
	huge := new(big.Int).Lsh(big.NewInt(1), 2000)
	res := FromGuesses(huge, "")
	// log2(2^2000) = 2000. Allow ±1 because the bit-length fallback is
	// exact only for powers of two.
	if res.Entropy < 1999 || res.Entropy > 2001 {
		t.Errorf("Entropy for 2^2000 = %v, want ~2000", res.Entropy)
	}
}
