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
