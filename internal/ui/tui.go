package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

	costStr := fmt.Sprintf("$%.2f USD", m.data.CostUSD)
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
		fmt.Sprintf("%s%s", propertyStyle.Render("Password Length:"), valueStyle.Render(fmt.Sprintf("%d chars", m.data.PasswordLength))),
		fmt.Sprintf("%s%s", propertyStyle.Render("Character Space:"), valueStyle.Render(fmt.Sprintf("%d", m.data.CharSpace))),
		fmt.Sprintf("%s%s", propertyStyle.Render("Entropy:"), entropyColored),
		fmt.Sprintf("%s%s", propertyStyle.Render("Combinations:"), valueStyle.Render(m.data.Combinations.String())),
		"",
		fmt.Sprintf("%s%s", propertyStyle.Render("Target Hardware:"), valueStyle.Render(m.data.Hardware)),
		fmt.Sprintf("%s%s", propertyStyle.Render("Hashrate:"), valueStyle.Render(fmt.Sprintf("%.0f H/s", m.data.HashRate))),
		fmt.Sprintf("%s%s", propertyStyle.Render("Time to Crack:"), timeColored),
		fmt.Sprintf("%s%s", propertyStyle.Render("Estimated Cost:"), costColored),
	}

	if m.data.BudgetUSD > 0 {
		rows = append(rows, "")
		rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Budget Target:"), valueStyle.Render(fmt.Sprintf("$%.2f USD", m.data.BudgetUSD))))
		if m.data.BudgetMaxChars > 0 {
			rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Max Safe Chars:"), safeStyle.Render(fmt.Sprintf("%d chars (Within budget)", m.data.BudgetMaxChars))))
		} else {
			rows = append(rows, fmt.Sprintf("%s%s", propertyStyle.Render("Max Safe Chars:"), criticalStyle.Render("0 (Cannot resist this attacker)")))
		}
	}

	content := strings.Join(rows, "\n")
	return boxStyle.Render(content) + "\n\n"
}

// RunTUI prints the styled result box to stdout. The view is fully determined
// by data computed before this call, so there is nothing interactive to do —
// printing the Lipgloss render directly avoids spinning up a full Bubble Tea
// program (and the terminal it needs, which is absent when piped).
func RunTUI(data OutputData) error {
	fmt.Print(initialModel(data).View())
	return nil
}
