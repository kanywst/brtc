package ui

import (
	"encoding/json"
	"fmt"
	"math/big"
)

type OutputData struct {
	PasswordLength int      `json:"password_length"`
	CharSpace      int      `json:"char_space"`
	Entropy        float64  `json:"entropy_bits"`
	Combinations   *big.Int `json:"combinations"`
	Algorithm      string   `json:"algorithm"`
	WorkFactor     int      `json:"work_factor"`
	MemoryMB       int      `json:"memory_mb,omitempty"`
	Hardware       string   `json:"hardware_profile"`
	HashRate       float64  `json:"hash_rate_per_sec"`
	TimeToCrackSec float64  `json:"time_to_crack_seconds"`
	CostUSD        float64  `json:"cost_usd"`
	BudgetUSD      float64  `json:"budget_usd,omitempty"`
	BudgetMaxChars int      `json:"budget_max_chars,omitempty"`
	// RecommendedChars is the minimum password length that would survive the
	// --fail-under-time threshold on this hardware. 0 means not requested.
	RecommendedChars int `json:"recommended_chars,omitempty"`
}

// MarshalJSON emits combinations as a JSON string rather than a number.
// R^L can be hundreds of digits long, and even a 9-char password exceeds
// 2^53, so a bare JSON number silently loses precision in JavaScript and
// other IEEE-754 parsers. Serialising it as a string keeps it exact.
func (d OutputData) MarshalJSON() ([]byte, error) {
	type alias OutputData // avoids recursing into this method
	combinations := "0"
	if d.Combinations != nil {
		combinations = d.Combinations.String()
	}
	return json.Marshal(struct {
		alias
		Combinations string `json:"combinations"`
	}{alias: alias(d), Combinations: combinations})
}

// UnmarshalJSON is the inverse of MarshalJSON: it reads combinations from a
// JSON string back into the big.Int field so the format round-trips.
func (d *OutputData) UnmarshalJSON(b []byte) error {
	type alias OutputData // avoids recursing into this method
	aux := struct {
		*alias
		Combinations string `json:"combinations"`
	}{alias: (*alias)(d)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	if aux.Combinations != "" {
		n, ok := new(big.Int).SetString(aux.Combinations, 10)
		if !ok {
			return fmt.Errorf("combinations: invalid integer %q", aux.Combinations)
		}
		d.Combinations = n
	}
	return nil
}

func PrintJSON(data OutputData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// PrintSARIF outputs a basic SARIF structure.
// This is somewhat a stretch since SARIF is meant for static analysis,
// but we will output a dummy warning if entropy is low.
func PrintSARIF(data OutputData) error {
	level := "warning"
	if data.Entropy > 80 {
		level = "note" // Safe
	} else if data.Entropy < 50 {
		level = "error" // Critical
	}

	sarif := fmt.Sprintf(`{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "brtc",
          "informationUri": "https://github.com/kanywst/brtc",
          "rules": [
            {
              "id": "BRTC-001",
              "name": "WeakPasswordCost",
              "shortDescription": {"text": "Password can be cracked cheaply"}
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "BRTC-001",
          "level": "%s",
          "message": {
            "text": "Password has %.2f bits entropy. Cracking takes %s and costs $%.2f USD."
          }
        }
      ]
    }
  ]
}`, level, data.Entropy, FormatDuration(data.TimeToCrackSec), data.CostUSD)

	fmt.Println(sarif)
	return nil
}

// FormatDuration is a helper to turn seconds into human-readable duration strings.
func FormatDuration(seconds float64) string {
	if seconds < 1 {
		return "Less than a second"
	}
	if seconds < 60 {
		return fmt.Sprintf("%.1f seconds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.1f minutes", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%.1f hours", seconds/3600)
	}
	if seconds < 31536000 {
		return fmt.Sprintf("%.1f days", seconds/86400)
	}
	return fmt.Sprintf("%.1f years", seconds/31536000)
}
