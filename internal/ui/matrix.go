package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// PrintMatrixTable writes the comparison rows as a plain aligned table.
func PrintMatrixTable(rows []MatrixRow) error {
	return renderMatrixTable(os.Stdout, rows)
}

func renderMatrixTable(out io.Writer, rows []MatrixRow) error {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "HARDWARE\tHASHRATE\tTIME TO CRACK\tCOST")
	for _, r := range rows {
		cost := fmt.Sprintf("$%.2f", r.CostUSD)
		if r.CostPerHourUSD <= 0 {
			cost = "owned" // no rental cost
		}
		fmt.Fprintf(w, "%s\t%.0f H/s\t%s\t%s\n", r.Profile, r.HashRate, FormatDuration(r.TimeToCrackSec), cost)
	}
	return w.Flush()
}
