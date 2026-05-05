package calc

import (
	"math/big"
	"testing"
)

func TestParseMemory(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"64", 64, false},
		{"64m", 64, false},
		{"64M", 64, false},
		{"  128m  ", 128, false},
		{"1g", 1024, false},
		{"2G", 2048, false},

		{"", 0, true},
		{"64k", 0, true},
		{"abc", 0, true},
		{"64mb", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseMemory(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %v", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseMemory(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseGuesses(t *testing.T) {
	bigOne := func(s string) *big.Int {
		n, _ := new(big.Int).SetString(s, 10)
		return n
	}

	tests := []struct {
		in      string
		want    *big.Int
		wantErr bool
	}{
		{"0", big.NewInt(0), false},
		{"12345", big.NewInt(12345), false},
		{"  100  ", big.NewInt(100), false},
		{"1e10", big.NewInt(10_000_000_000), false},
		{"1.5e3", big.NewInt(1500), false},
		{"99999999999999999999999999999", bigOne("99999999999999999999999999999"), false},

		{"", nil, true},
		{"-5", nil, true},
		{"abc", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseGuesses(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %v", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Cmp(tt.want) != 0 {
				t.Errorf("ParseGuesses(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
