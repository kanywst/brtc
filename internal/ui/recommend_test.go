package ui

import (
	"strings"
	"testing"
)

func TestView_ShowsRecommendedLength(t *testing.T) {
	data := sampleData()
	data.RecommendedChars = 16

	// loaded:true so View renders the result box rather than the spinner.
	out := model{data: data, loaded: true}.View()
	if !strings.Contains(out, "Recommended Len") || !strings.Contains(out, "16") {
		t.Errorf("View() should surface the recommended length, got:\n%s", out)
	}
}
