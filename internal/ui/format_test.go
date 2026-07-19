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
