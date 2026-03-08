package cost

import (
	"math"
	"strings"
)

// Hardware profiles and their base hashrates (Hashes/second).
// Base hashrates are approximated for realistic conditions.
type HardwareProfile struct {
	Name             string
	BaseHashesMD5    float64
	BaseHashesSHA256 float64
	BaseHashesBcrypt float64 // At cost = 5
	BaseHashesArgon2 float64
	CostPerHourUSD   float64 // Cloud instance spot price or rental cost
}

var Profiles = map[string]HardwareProfile{
	"mac-m3": {
		Name:             "Apple M3 (Base)",
		BaseHashesMD5:    8_000_000_000,
		BaseHashesSHA256: 1_500_000_000,
		BaseHashesBcrypt: 5_500,
		BaseHashesArgon2: 60,
		CostPerHourUSD:   0.0, // Owned
	},
	"rtx-4090": {
		Name:             "NVIDIA RTX 4090 (Single)",
		BaseHashesMD5:    164_000_000_000,
		BaseHashesSHA256: 23_000_000_000,
		BaseHashesBcrypt: 100_000,
		BaseHashesArgon2: 1_000,
		CostPerHourUSD:   0.30, // Approx Vast.ai spot price
	},
	"aws-p5.48xlarge": {
		Name:             "AWS p5.48xlarge (8x H100)",
		BaseHashesMD5:    164_000_000_000 * 25, // roughly 25x a single RTX 4090
		BaseHashesSHA256: 23_000_000_000 * 25,
		BaseHashesBcrypt: 100_000 * 25,
		BaseHashesArgon2: 1_000 * 25,
		CostPerHourUSD:   40.0, // Approx spot price
	},
	"rtx-3060": {
		Name:             "NVIDIA RTX 3060",
		BaseHashesMD5:    22_000_000_000,
		BaseHashesSHA256: 3_000_000_000,
		BaseHashesBcrypt: 12_000,
		BaseHashesArgon2: 120,
		CostPerHourUSD:   0.05, // e.g. Vast.ai low end spot
	},
	"gtx-1080ti": {
		Name:             "NVIDIA GTX 1080 Ti (Historical, 2017)",
		BaseHashesMD5:    34_000_000_000,
		BaseHashesSHA256: 4_500_000_000,
		BaseHashesBcrypt: 22_700,
		BaseHashesArgon2: 80,
		CostPerHourUSD:   0.0, // Owned
	},
	"cpu-standard": {
		Name:             "Standard 8-Core CPU",
		BaseHashesMD5:    1_000_000_000,
		BaseHashesSHA256: 200_000_000,
		BaseHashesBcrypt: 5_000,
		BaseHashesArgon2: 50,
		CostPerHourUSD:   0.05, // e.g., t3.large spot
	},
}

// CalculateHashRate computes the exact hashes per second for a given algo and workfactor.
func CalculateHashRate(hw string, algo string, workFactor int) float64 {
	p, ok := Profiles[strings.ToLower(hw)]
	if !ok {
		p = Profiles["rtx-4090"]
	}

	switch strings.ToLower(algo) {
	case "md5":
		return p.BaseHashesMD5
	case "sha256":
		return p.BaseHashesSHA256
	case "bcrypt":
		// Bcrypt cost is exponential (2^cost). Base is cost=5.
		// Ex: cost=10 is 2^5 = 32 times slower.
		factor := math.Pow(2, float64(workFactor-5))
		if factor < 1 {
			factor = 1
		}
		return p.BaseHashesBcrypt / factor
	case "argon2id":
		// Argon2id workfactor scaling (roughly linear with time/memory limits, assuming default memory)
		// Assuming base is t=1. If workFactor > 1, divided by workFactor.
		factor := float64(workFactor)
		if factor < 1 {
			factor = 1
		}
		return p.BaseHashesArgon2 / factor
	default:
		return p.BaseHashesBcrypt // Safe fallback
	}
}

// TotalCost calculates the total cost to crack based on the time it takes.
// timeInSeconds allows calculation of fractions of hours.
func TotalCost(hw string, timeInSeconds float64) float64 {
	p, ok := Profiles[strings.ToLower(hw)]
	if !ok {
		p = Profiles["rtx-4090"]
	}

	hours := timeInSeconds / 3600.0
	return hours * p.CostPerHourUSD
}
