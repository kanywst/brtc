package ui

import (
	"math/big"
	"testing"
)

func TestFormatCombinations(t *testing.T) {
	bigPow := new(big.Int).Exp(big.NewInt(94), big.NewInt(200), nil) // ~395 digits

	tests := []struct {
		name string
		in   *big.Int
		want string
	}{
		{"nil", nil, "0"},
		{"small stays exact", big.NewInt(1234567890), "1234567890"},
		{"21 digits stays exact", mustBig("100000000000000000000"), "100000000000000000000"},
		{"22 digits goes scientific", mustBig("1234567890123456789012"), "≈ 1.234e21"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCombinations(tt.in); got != tt.want {
				t.Errorf("formatCombinations(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	// A hundreds-of-digits value must collapse to a short token, never the
	// full number, so the TUI box stays intact.
	if got := formatCombinations(bigPow); len([]rune(got)) > 14 {
		t.Errorf("formatCombinations(94^200) = %q, want a short scientific form", got)
	}
}

func mustBig(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("bad test bigint: " + s)
	}
	return n
}
