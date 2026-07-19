package ui

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// maxCombinationDigits is the longest combination count printed verbatim in
// the TUI. Beyond it (a long password yields a hundreds-of-digits R^L) the
// number is shown in scientific notation so it cannot blow out the box width.
const maxCombinationDigits = 21

// formatCombinations renders a combination count for the TUI: exact digits
// while it stays short, scientific notation once it would overflow the box.
// Scientific form goes through big.Float's %.3e so it rounds (rather than
// truncates) and handles negative inputs correctly.
func formatCombinations(n *big.Int) string {
	if n == nil {
		return "0"
	}
	if len(new(big.Int).Abs(n).String()) <= maxCombinationDigits {
		return n.String()
	}
	f := new(big.Float).SetInt(n)
	return "≈ " + strings.Replace(fmt.Sprintf("%.3e", f), "e+", "e", 1)
}

// Style definitions
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6")).
			MarginBottom(1)

	propertyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD")).Bold(true).Width(20)
	valueStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2"))

	criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C")).Bold(true)
	safeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#BD93F9")).
			Padding(1, 4).
			MarginTop(1).
			MarginLeft(2)

	noteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Italic(true)
)

type model struct {
	data OutputData
}

func initialModel(data OutputData) model {
	return model{data: data}
}

func (m model) View() string {
	// Format values
	entropyStr := fmt.Sprintf("%.2f bits", m.data.Entropy)
	var entropyColored string
	if m.data.Entropy < 50 {
		entropyColored = criticalStyle.Render(entropyStr)
	} else if m.data.Entropy < 80 {
		entropyColored = warningStyle.Render(entropyStr)
	} else {
		entropyColored = safeStyle.Render(entropyStr)
	}

	timeStr := FormatDuration(m.data.TimeToCrackSec)
	var timeColored string
	if m.data.TimeToCrackSec < 86400 {
		timeColored = criticalStyle.Render(timeStr)
	} else if m.data.TimeToCrackSec < 31536000 {
		timeColored = warningStyle.Render(timeStr)
	} else {
		timeColored = safeStyle.Render(timeStr)
	}

	costStr := FormatCost(m.data.CostUSD) + " USD"
	var costColored string
	if m.data.CostUSD < 100 {
		costColored = criticalStyle.Render(costStr)
	} else if m.data.CostUSD < 10000 {
		costColored = warningStyle.Render(costStr)
	} else {
		costColored = safeStyle.Render(costStr)
	}

	rows := []string{
		titleStyle.Render(fmt.Sprintf("brtc: Brute-force Cost Analysis (%s)", m.data.Algorithm)),
		renderVerdict(m.data),
		"",
		fmt.Sprintf("%s%s", propertyStyle.Render("Password Length:"), valueStyle.Render(fmt.Sprintf("%d chars", m.data.PasswordLength))),
		fmt.Sprintf("%s%s", propertyStyle.Render("Character Space:"), valueStyle.Render(fmt.Sprintf("%d", m.data.CharSpace))),
		fmt.Sprintf("%s%s", propertyStyle.Render("Entropy:"), entropyColored),
		fmt.Sprintf("%s%s", propertyStyle.Render(""), renderEntropyBar(m.data.Entropy)),
		fmt.Sprintf("%s%s", propertyStyle.Render("Combinations:"), valueStyle.Render(formatCombinations(m.data.Combinations))),
		"",
		fmt.Sprintf("%s%s", propertyStyle.Render("Target Hardware:"), valueStyle.Render(m.data.Hardware)),
		fmt.Sprintf("%s%s", propertyStyle.Render("Hashrate:"), valueStyle.Render(FormatHashRate(m.data.HashRate))),
		fmt.Sprintf("%s%s", propertyStyle.Render("Time to Crack:"), timeColored),
		fmt.Sprintf("%s%s", propertyStyle.Render("Estimated Cost:"), costColored),
	}

	if m.data.RecommendedChars > 0 {
		rows = append(rows, "")
		rec := fmt.Sprintf("≥ %d chars (to resist this attacker)", m.data.RecommendedChars)
		// Green once the current password already meets the recommendation.
		recStyle := warningStyle
		if m.data.PasswordLength >= m.data.RecommendedChars {
			recStyle = safeStyle
		}
		rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Recommended Len:"), recStyle.Render(rec)))
	}

	if m.data.BudgetUSD > 0 {
		rows = append(rows, "")
		rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Budget Target:"), valueStyle.Render(fmt.Sprintf("$%.2f USD", m.data.BudgetUSD))))
		if m.data.BudgetUnlimited {
			rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Max Safe Chars:"), criticalStyle.Render("∞ (owned hardware — no rental cost)")))
		} else if m.data.BudgetMaxChars != nil {
			if *m.data.BudgetMaxChars > 0 {
				rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Max Safe Chars:"), safeStyle.Render(fmt.Sprintf("%d chars (Within budget)", *m.data.BudgetMaxChars))))
			} else {
				rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Max Safe Chars:"), criticalStyle.Render("0 (Cannot resist this attacker)")))
			}
		}
	}

	rows = append(rows, "", noteStyle.Render(EntropyModelNote(m.data)))

	content := strings.Join(rows, "\n")
	return boxStyle.Render(content) + "\n\n"
}

// entropyBarWidth is the cell count of the entropy gauge. entropyBarCap is the
// bit value that fills it completely; 128 bits is the "comfortably beyond any
// brute force" mark, so a full bar reads as "as strong as this gauge tracks".
const (
	entropyBarWidth = 16
	entropyBarCap   = 128.0
)

// renderEntropyBar draws a coloured gauge like "████████░░░░░░░░ 78/128 bits"
// so strength is legible at a glance, not just as a number. The fill colour
// reuses the entropy severity thresholds used elsewhere in the view.
func renderEntropyBar(entropy float64) string {
	ratio := entropy / entropyBarCap
	if ratio < 0 {
		ratio = 0
	} else if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio*entropyBarWidth + 0.5)

	style := safeStyle
	if entropy < 50 {
		style = criticalStyle
	} else if entropy < 80 {
		style = warningStyle
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", entropyBarWidth-filled)
	return fmt.Sprintf("%s %s", style.Render(bar), valueStyle.Render(fmt.Sprintf("%.0f/%.0f bits", entropy, entropyBarCap)))
}

// renderVerdict is the one-line headline under the title: the whole point of
// the tool distilled into "how bad is it", coloured red/yellow/green. It keys
// off crack time (the metric a human feels) and appends the price tag.
func renderVerdict(data OutputData) string {
	cost := FormatCost(data.CostUSD)
	time := FormatDuration(data.TimeToCrackSec)
	switch {
	case data.TimeToCrackSec < 86400: // under a day
		return criticalStyle.Render(fmt.Sprintf("VERDICT: TRIVIALLY CRACKABLE — %s for %s", time, cost))
	case data.TimeToCrackSec < 31536000: // under a year
		return warningStyle.Render(fmt.Sprintf("VERDICT: CRACKABLE — %s for %s", time, cost))
	default:
		return safeStyle.Render(fmt.Sprintf("VERDICT: RESISTANT — %s, %s", time, cost))
	}
}

// RunTUI prints the styled result box to stdout. The view is fully determined
// by data computed before this call, so there is nothing interactive to do —
// printing the Lipgloss render directly avoids spinning up a full Bubble Tea
// program (and the terminal it needs, which is absent when piped).
func RunTUI(data OutputData) error {
	fmt.Print(initialModel(data).View())
	return nil
}
