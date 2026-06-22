package ui

import (
	"strings"
	"testing"
)

func TestView_RendersCoreFields(t *testing.T) {
	out := initialModel(sampleData()).View()
	for _, want := range []string{"Brute-force Cost Analysis", "Time to Crack", "Estimated Cost"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q in:\n%s", want, out)
		}
	}
}
