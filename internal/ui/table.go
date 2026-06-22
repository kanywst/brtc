package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintTable writes the result as a plain, colour-free aligned table. Unlike
// the TUI it emits no ANSI escapes, so it is safe to pipe, paste, or read in
// environments where the JSON output is overkill and the TUI is noise.
func PrintTable(data OutputData) error {
	return renderTable(os.Stdout, data)
}

func renderTable(out io.Writer, data OutputData) error {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)

	rows := [][2]string{
		{"Algorithm", data.Algorithm},
		{"Work Factor", fmt.Sprintf("%d", data.WorkFactor)},
	}
	if data.MemoryMB > 0 {
		rows = append(rows, [2]string{"Memory", fmt.Sprintf("%d MB", data.MemoryMB)})
	}
	rows = append(rows,
		[2]string{"Password Length", fmt.Sprintf("%d chars", data.PasswordLength)},
		[2]string{"Character Space", fmt.Sprintf("%d", data.CharSpace)},
		[2]string{"Entropy", fmt.Sprintf("%.2f bits", data.Entropy)},
		[2]string{"Combinations", data.Combinations.String()},
		[2]string{"Target Hardware", data.Hardware},
		[2]string{"Hashrate", fmt.Sprintf("%.0f H/s", data.HashRate)},
		[2]string{"Time to Crack", FormatDuration(data.TimeToCrackSec)},
		[2]string{"Estimated Cost", fmt.Sprintf("$%.2f USD", data.CostUSD)},
	)
	if data.BudgetUSD > 0 {
		maxChars := fmt.Sprintf("%d chars (within budget)", data.BudgetMaxChars)
		if data.BudgetMaxChars == 0 {
			maxChars = "0 (cannot resist this attacker)"
		}
		rows = append(rows,
			[2]string{"Budget Target", fmt.Sprintf("$%.2f USD", data.BudgetUSD)},
			[2]string{"Max Safe Chars", maxChars},
		)
	}

	lines := []string{"PROPERTY\tVALUE"}
	for _, r := range rows {
		lines = append(lines, r[0]+"\t"+r[1])
	}
	if _, err := io.WriteString(w, strings.Join(lines, "\n")+"\n"); err != nil {
		return err
	}
	return w.Flush()
}
