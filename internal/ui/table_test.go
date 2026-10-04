package ui

import (
	"strings"
	"testing"
)

func TestRenderTable(t *testing.T) {
	var buf strings.Builder
	if err := renderTable(&buf, sampleData()); err != nil {
		t.Fatalf("renderTable failed: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"PROPERTY", "Algorithm", "Time to Crack", "Estimated Cost"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q in:\n%s", want, out)
		}
	}
	// The table is the plain-text format: it must never emit ANSI escapes.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("table output should be ANSI-free, got:\n%q", out)
	}
}

func TestRenderTable_Baseline(t *testing.T) {
	tests := []struct {
		name      string
		stale     bool
		wantStale bool
	}{
		{"fresh", false, false},
		{"stale", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := sampleData()
			d.BaselineReviewed, d.BaselineStale = "2026-07-19", tt.stale
			var buf strings.Builder
			if err := renderTable(&buf, d); err != nil {
				t.Fatalf("renderTable failed: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, "Baseline Date") || !strings.Contains(out, "2026-07-19") {
				t.Errorf("table missing the baseline date in:\n%s", out)
			}
			if got := strings.Contains(out, "over a year old"); got != tt.wantStale {
				t.Errorf("stale note present = %v, want %v in:\n%s", got, tt.wantStale, out)
			}
		})
	}
}
