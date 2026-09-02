package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestView_RendersCoreFields(t *testing.T) {
	out := initialModel(sampleData()).View()
	for _, want := range []string{"Brute-force Cost Analysis", "Time to Crack", "Estimated Cost"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q in:\n%s", want, out)
		}
	}
}

// maxBoxWidth is the widest the box may render. The entropy-model note used
// to be one 148-character line, which made lipgloss size the border to about
// 158 columns; a narrower terminal then wrapped the text but not the border
// and the box came out visibly broken in the README demo. 100 leaves room
// for the data rows while staying inside an 80-to-120 column terminal.
const maxBoxWidth = 100

func TestView_FitsInATerminal(t *testing.T) {
	cases := map[string]OutputData{
		"basic": sampleData(),
		"external guess count": func() OutputData {
			d := sampleData()
			d.CharSpace = 0 // selects the shorter note
			return d
		}(),
		"every optional section": func() OutputData {
			d := sampleData()
			d.RecommendedChars = 16
			d.BudgetUSD = 500000
			d.BudgetMaxChars = new(int)
			*d.BudgetMaxChars = 12
			d.BreachChecked = true
			d.BreachCount = 6421042
			return d
		}(),
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			for _, line := range strings.Split(initialModel(data).View(), "\n") {
				if w := lipgloss.Width(line); w > maxBoxWidth {
					t.Errorf("line renders %d columns wide, want <= %d:\n%s", w, maxBoxWidth, line)
				}
			}
		})
	}
}
