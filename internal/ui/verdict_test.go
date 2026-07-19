package ui

import (
	"strings"
	"testing"
)

func TestRenderVerdict_ByCrackTime(t *testing.T) {
	tests := []struct {
		ttc     float64
		wantSub string
	}{
		{3600, "TRIVIALLY CRACKABLE"}, // under a day
		{100 * 86400, "CRACKABLE"},    // months, under a year
		{100 * 31536000, "RESISTANT"}, // a century
	}
	for _, tt := range tests {
		d := sampleData()
		d.TimeToCrackSec = tt.ttc
		got := renderVerdict(d)
		if !strings.Contains(got, tt.wantSub) {
			t.Errorf("renderVerdict(ttc=%v) = %q, want substring %q", tt.ttc, got, tt.wantSub)
		}
	}
}

func TestRenderEntropyBar_Clamps(t *testing.T) {
	// A huge entropy fills the bar without overflowing its width.
	full := renderEntropyBar(1000)
	if strings.Count(full, "█") != entropyBarWidth {
		t.Errorf("bar should be full at high entropy, got %q", full)
	}
	// Zero entropy shows an empty bar, never a negative repeat count.
	empty := renderEntropyBar(0)
	if strings.Count(empty, "░") != entropyBarWidth {
		t.Errorf("bar should be empty at zero entropy, got %q", empty)
	}
}

func TestView_ShowsVerdictAndBar(t *testing.T) {
	out := initialModel(sampleData()).View()
	if !strings.Contains(out, "VERDICT") {
		t.Errorf("view should include a verdict banner, got:\n%s", out)
	}
	if !strings.Contains(out, "/128 bits") {
		t.Errorf("view should include the entropy gauge, got:\n%s", out)
	}
}

func TestRenderVerdict_BreachOverridesTime(t *testing.T) {
	d := sampleData()
	d.TimeToCrackSec = 100 * 31536000 // would otherwise read RESISTANT
	d.BreachChecked = true
	d.BreachCount = 5000
	got := renderVerdict(d)
	if !strings.Contains(got, "COMPROMISED") {
		t.Errorf("a breached password must read COMPROMISED regardless of crack time, got %q", got)
	}
}

func TestView_ShowsBreachRow(t *testing.T) {
	d := sampleData()
	d.BreachChecked = true
	d.BreachCount = 42
	if out := initialModel(d).View(); !strings.Contains(out, "HIBP Breaches") {
		t.Errorf("view should show the HIBP breach row when checked, got:\n%s", out)
	}
}
