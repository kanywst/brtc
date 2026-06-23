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

func TestOutputData_CombinationsMarshalsAsString(t *testing.T) {
	// 94^9 exceeds 2^53, so it must be a JSON string to survive parsers that
	// represent numbers as IEEE-754 doubles.
	data := sampleData()
	data.Combinations, _ = new(big.Int).SetString("572994802228616704", 10)

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(b), `"combinations": "572994802228616704"`) &&
		!strings.Contains(string(b), `"combinations":"572994802228616704"`) {
		t.Errorf("combinations should be a quoted string, got:\n%s", b)
	}
}

func TestOutputData_NilCombinationsIsNull(t *testing.T) {
	data := sampleData()
	data.Combinations = nil

	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(b), `"combinations":null`) {
		t.Errorf("nil combinations should marshal as null, got:\n%s", b)
	}
}

func TestOutputData_UnmarshalNullClearsExisting(t *testing.T) {
	// An explicit null must clear a pre-existing value, not leave it stale.
	got := OutputData{Combinations: big.NewInt(42)}
	if err := json.Unmarshal([]byte(`{"combinations": null}`), &got); err != nil {
		t.Fatalf("unmarshal null failed: %v", err)
	}
	if got.Combinations != nil {
		t.Errorf("null should clear combinations, got %v", got.Combinations)
	}
}

func TestOutputData_UnmarshalLegacyNumber(t *testing.T) {
	// Older payloads carried combinations as a bare JSON number; those must
	// still decode for backward compatibility.
	const legacy = `{"combinations": 572994802228616704}`
	var got OutputData
	if err := json.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal legacy number failed: %v", err)
	}
	if got.Combinations == nil || got.Combinations.String() != "572994802228616704" {
		t.Errorf("legacy number did not decode: got %v", got.Combinations)
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
