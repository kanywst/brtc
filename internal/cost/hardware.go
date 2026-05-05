package cost

import (
	_ "embed"
	"fmt"
	"math"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed hashrates.yaml
var hashratesYAML []byte

type HardwareProfile struct {
	Name           string             `yaml:"name"`
	CostPerHourUSD float64            `yaml:"cost_per_hour_usd"`
	Hashrates      map[string]float64 `yaml:"hashrates"`
	Source         string             `yaml:"source,omitempty"`
	LastReviewed   string             `yaml:"last_reviewed,omitempty"`
}

type profilesFile struct {
	Profiles map[string]HardwareProfile `yaml:"profiles"`
}

// Profiles is populated at init time from the embedded hashrates.yaml.
var Profiles map[string]HardwareProfile

// fallbackProfile is used when the caller passes an --hw value not
// present in Profiles. Kept as a constant rather than rtx-4090 magic
// strings so the fallback target is greppable.
const fallbackProfile = "rtx-4090"

func init() {
	var f profilesFile
	if err := yaml.Unmarshal(hashratesYAML, &f); err != nil {
		// hashrates.yaml is embedded at build time; a parse error is a
		// programmer/build error, not a runtime user error.
		panic(fmt.Errorf("cost: parse embedded hashrates.yaml: %w", err))
	}
	fb, ok := f.Profiles[fallbackProfile]
	if !ok {
		panic(fmt.Errorf("cost: hashrates.yaml is missing the fallback profile %q", fallbackProfile))
	}
	// Unknown algorithms are routed through the bcrypt path (see
	// CalculateHashRate). The fallback profile must therefore carry a
	// positive bcrypt rate, otherwise that path silently returns 0.
	if rate, ok := fb.Hashrates["bcrypt"]; !ok || rate <= 0 {
		panic(fmt.Errorf("cost: fallback profile %q must have a positive bcrypt hashrate", fallbackProfile))
	}
	Profiles = f.Profiles
}

func lookupProfile(hw string) HardwareProfile {
	if p, ok := Profiles[strings.ToLower(hw)]; ok {
		return p
	}
	return Profiles[fallbackProfile]
}

func CalculateHashRate(hw string, algo string, workFactor int) float64 {
	p := lookupProfile(hw)
	algo = strings.ToLower(algo)

	if _, ok := p.Hashrates[algo]; !ok {
		// Unknown algorithm: route through bcrypt entirely (rate AND
		// scaling). Returning the bare bcrypt baseline without applying
		// the cost-factor scaling would silently overestimate the
		// attacker for any workFactor > 5.
		algo = "bcrypt"
	}
	base := p.Hashrates[algo]

	switch algo {
	case "bcrypt":
		// Bcrypt cost is exponential (2^cost). Baseline is cost=5;
		// e.g. cost=10 is 2^5 = 32 times slower. workFactor < 5 is
		// clamped so an unrealistic input never exceeds the baseline.
		factor := math.Pow(2, float64(workFactor-5))
		if factor < 1 {
			factor = 1
		}
		return base / factor
	case "argon2id":
		factor := float64(workFactor)
		if factor < 1 {
			factor = 1
		}
		return base / factor
	default:
		return base
	}
}

func TotalCost(hw string, timeInSeconds float64) float64 {
	p := lookupProfile(hw)
	hours := timeInSeconds / 3600.0
	return hours * p.CostPerHourUSD
}
