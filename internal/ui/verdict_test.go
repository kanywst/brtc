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
