package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// MatrixRow is one hardware profile's verdict against a single password, used
// by the --all-hw comparison view.
type MatrixRow struct {
	Profile        string  `json:"profile"`
	Name           string  `json:"name"`
	HashRate       float64 `json:"hash_rate_per_sec"`
	TimeToCrackSec float64 `json:"time_to_crack_seconds"`
	CostUSD        float64 `json:"cost_usd"`
	CostPerHourUSD float64 `json:"cost_per_hour_usd"`
}

// PrintMatrixJSON writes the comparison rows as a JSON array.
func PrintMatrixJSON(rows []MatrixRow) error {
	return renderMatrixJSON(os.Stdout, rows)
}

func renderMatrixJSON(out io.Writer, rows []MatrixRow) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// PrintMatrixTable writes the comparison rows as a plain aligned table.
func PrintMatrixTable(rows []MatrixRow) error {
	return renderMatrixTable(os.Stdout, rows)
}

func renderMatrixTable(out io.Writer, rows []MatrixRow) error {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)

	lines := []string{"HARDWARE\tHASHRATE\tTIME TO CRACK\tCOST"}
	for _, r := range rows {
		cost := fmt.Sprintf("$%.2f", r.CostUSD)
		if r.CostPerHourUSD <= 0 {
			cost = "owned" // no rental cost
		}
		lines = append(lines, fmt.Sprintf("%s\t%.0f H/s\t%s\t%s", r.Profile, r.HashRate, FormatDuration(r.TimeToCrackSec), cost))
	}
	if _, err := io.WriteString(w, strings.Join(lines, "\n")+"\n"); err != nil {
		return err
	}
	return w.Flush()
}
