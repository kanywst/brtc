package ui

import (
	"fmt"
	"math"
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

// wrapStyle re-flows a row that would otherwise stretch the box. It carries
// no colour of its own, so a row that arrives already styled keeps its own.
var wrapStyle = lipgloss.NewStyle().Width(maxContentWidth)

const (
	// minNoteWidth is the floor for wrapping the entropy-model note. The data
	// rows are normally wider than this; the floor only matters if they ever
	// get much narrower, where wrapping the sentence to the box would leave
	// one or two words per line.
	minNoteWidth = 60

	// maxContentWidth is the ceiling for any row inside the box. boxStyle adds
	// 8 columns of padding, 2 of border, and 2 of left margin, so the rendered
	// box stays within 100 columns and fits a standard terminal.
	maxContentWidth = 88
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

	if m.data.BreachChecked {
		rows = append(rows, "")
		if m.data.BreachCount > 0 {
			breach := criticalStyle.Render(fmt.Sprintf("⚠ found %s times — change it", FormatCount(m.data.BreachCount)))
			rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("HIBP Breaches:"), breach))
		} else {
			rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("HIBP Breaches:"), safeStyle.Render("not found in the corpus")))
		}
	}

	// Nothing here may drive the border past a normal terminal. lipgloss
	// sizes the box to its widest line, and the terminal then wraps the text
	// but not the border, which leaves the box visibly broken (the README
	// demo showed it). Two rows can get long enough to do that: the note,
	// which is a full sentence, and a verdict carrying an astronomical crack
	// time and cost ("9.9 decillion years (7.2 sextillion x the age of the
	// universe), $9.9 decillion"). Wrapping every row that exceeds the cap
	// covers both, and any future row that grows.
	for i, row := range rows {
		if lipgloss.Width(row) > maxContentWidth {
			rows[i] = wrapStyle.Render(row)
		}
	}

	// Wrap the note to the width the data rows already need, so it never
	// drives the box on its own. The floor keeps it from shredding into
	// fragments if those rows are narrow; the cap applies when they are not.
	noteWidth := lipgloss.Width(strings.Join(rows, "\n"))
	if noteWidth < minNoteWidth {
		noteWidth = minNoteWidth
	}
	if noteWidth > maxContentWidth {
		noteWidth = maxContentWidth
	}
	rows = append(rows, "", noteStyle.Width(noteWidth).Render(EntropyModelNote(m.data)))

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
	// Guard NaN before the ratio math: NaN fails every comparison below, so it
	// would slip past the clamp and int(NaN) yields a huge negative value that
	// panics strings.Repeat with a negative count.
	if math.IsNaN(entropy) || entropy < 0 {
		entropy = 0
	}
	ratio := entropy / entropyBarCap
	if ratio > 1 {
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
	// A breached password is game over no matter how high its entropy is:
	// the attacker looks it up instead of cracking it. This overrides the
	// time-based verdict.
	if data.BreachChecked && data.BreachCount > 0 {
		return criticalStyle.Render(fmt.Sprintf("VERDICT: COMPROMISED — found in %s known breaches, crack time is irrelevant",
			FormatCount(data.BreachCount)))
	}
	cost := FormatCost(data.CostUSD)
	time := FormatDuration(data.TimeToCrackSec)
	switch {
	case math.IsNaN(data.TimeToCrackSec):
		// NaN fails every comparison below and would slip into the green
		// RESISTANT default — a false sense of security. Flag it instead.
		return criticalStyle.Render("VERDICT: UNKNOWN — crack time could not be calculated")
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
