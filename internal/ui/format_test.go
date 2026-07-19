package ui

import (
	"math"
	"strings"
	"testing"
)

func TestFormatDuration_LargeYearsAnchorToUniverse(t *testing.T) {
	// A ~1.3 trillion-year crack time should collapse to a named scale and
	// anchor against the age of the universe rather than dump raw digits.
	got := FormatDuration(1.3e12 * 31536000)
	if !strings.Contains(got, "trillion") {
		t.Errorf("want a named short-scale word, got %q", got)
	}
	if !strings.Contains(got, "age of the universe") {
		t.Errorf("want an age-of-universe anchor, got %q", got)
	}
}

func TestFormatDuration_ModerateYearsNoAnchor(t *testing.T) {
	// Below the age of the universe there is no anchor clause.
	got := FormatDuration(5000 * 31536000) // 5000 years
	if strings.Contains(got, "universe") {
		t.Errorf("5000 years should not anchor to the universe, got %q", got)
	}
	if !strings.Contains(got, "years") {
		t.Errorf("want a years unit, got %q", got)
	}
}

func TestFormatDuration_Infinity(t *testing.T) {
	if got := FormatDuration(math.Inf(1)); got != "effectively forever" {
		t.Errorf("Inf duration = %q, want effectively forever", got)
	}
}

func TestFormatHashRate(t *testing.T) {
	tests := []struct {
		hps  float64
		want string
	}{
		{0, "0 H/s"},
		{500, "500 H/s"},
		{5800, "5.8 kH/s"},
		{164100000000, "164.1 GH/s"},
		{1e12, "1.0 TH/s"},
	}
	for _, tt := range tests {
		if got := FormatHashRate(tt.hps); got != tt.want {
			t.Errorf("FormatHashRate(%v) = %q, want %q", tt.hps, got, tt.want)
		}
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		usd  float64
		want string
	}{
		{0, "$0.00"},
		{0.004, "< $0.01"},
		{12.5, "$12.50"},
		{6345.6, "$6,345.60"},
		{250000, "$250,000"},
		{4.0e15, "$4.0 quadrillion"},
	}
	for _, tt := range tests {
		if got := FormatCost(tt.usd); got != tt.want {
			t.Errorf("FormatCost(%v) = %q, want %q", tt.usd, got, tt.want)
		}
	}
}

func TestAddThousands(t *testing.T) {
	tests := map[string]string{
		"6345.60": "6,345.60",
		"250000":  "250,000",
		"999":     "999",
		"1000":    "1,000",
	}
	for in, want := range tests {
		if got := addThousands(in); got != want {
			t.Errorf("addThousands(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEntropyModelNote(t *testing.T) {
	naive := sampleData() // CharSpace 94 -> naive R^L estimate
	if !strings.Contains(EntropyModelNote(naive), "RANDOM") {
		t.Errorf("naive note should flag the random-password assumption, got %q", EntropyModelNote(naive))
	}
	external := sampleData()
	external.CharSpace = 0 // signals a --guesses / zxcvbn estimate
	if !strings.Contains(EntropyModelNote(external), "external") {
		t.Errorf("external note should mention the external estimate, got %q", EntropyModelNote(external))
	}
}

func TestFormatCount(t *testing.T) {
	cases := map[int]string{42: "42", 1000: "1,000", 9659365: "9,659,365"}
	for in, want := range cases {
		if got := FormatCount(in); got != want {
			t.Errorf("FormatCount(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatCost_NaN(t *testing.T) {
	if got := FormatCost(math.NaN()); got != "$0.00" {
		t.Errorf("FormatCost(NaN) = %q, want $0.00", got)
	}
}

func TestSanitizeFloat(t *testing.T) {
	if got := sanitizeFloat(math.Inf(1)); got != math.MaxFloat64 {
		t.Errorf("sanitizeFloat(+Inf) = %v, want MaxFloat64", got)
	}
	if got := sanitizeFloat(math.NaN()); got != 0 {
		t.Errorf("sanitizeFloat(NaN) = %v, want 0", got)
	}
	if got := sanitizeFloat(math.Inf(-1)); got != 0 {
		t.Errorf("sanitizeFloat(-Inf) = %v, want 0", got)
	}
	if got := sanitizeFloat(3.5); got != 3.5 {
		t.Errorf("sanitizeFloat(3.5) = %v, want 3.5", got)
	}
}

func TestRenderEntropyBar_NaNDoesNotPanic(t *testing.T) {
	// Regression: NaN slipped past the ratio clamp and int(NaN) produced a
	// negative repeat count that panicked strings.Repeat.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderEntropyBar(NaN) panicked: %v", r)
		}
	}()
	_ = renderEntropyBar(math.NaN())
}

func TestMarshalJSON_InfiniteFieldsAreFinite(t *testing.T) {
	d := sampleData()
	d.TimeToCrackSec = math.Inf(1)
	d.CostUSD = math.Inf(1)
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON with +Inf fields failed: %v", err)
	}
	if strings.Contains(string(b), "Inf") || strings.Contains(string(b), "NaN") {
		t.Errorf("JSON should not contain Inf/NaN, got: %s", b)
	}
}
