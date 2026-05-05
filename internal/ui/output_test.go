package ui

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

func sampleData() OutputData {
	return OutputData{
		PasswordLength: 8,
		CharSpace:      94,
		Entropy:        52.4,
		Combinations:   big.NewInt(1234567890),
		Algorithm:      "bcrypt",
		WorkFactor:     10,
		Hardware:       "rtx-4090",
		HashRate:       100000,
		TimeToCrackSec: 3600,
		CostUSD:        0.30,
	}
}

func TestOutputData_JSONRoundtrip(t *testing.T) {
	data := sampleData()

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got OutputData
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.Algorithm != data.Algorithm || got.PasswordLength != data.PasswordLength {
		t.Errorf("roundtrip mismatch: got %+v", got)
	}
	if got.Combinations.Cmp(data.Combinations) != 0 {
		t.Errorf("combinations roundtrip: got %v want %v", got.Combinations, data.Combinations)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		secs    float64
		wantSub string
	}{
		{0.5, "Less than a second"},
		{30, "seconds"},
		{120, "minutes"},
		{7200, "hours"},
		{3 * 86400, "days"},
		{2 * 31536000, "years"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.secs)
		if !strings.Contains(got, tt.wantSub) {
			t.Errorf("FormatDuration(%v) = %q, want substring %q", tt.secs, got, tt.wantSub)
		}
	}
}
