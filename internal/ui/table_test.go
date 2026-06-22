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
