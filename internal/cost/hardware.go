package cost

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
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

// fallbackProfile is the profile lookupProfile returns for a key that is
// not in Profiles. Kept as a constant rather than rtx-4090 magic strings
// so the fallback target is greppable.
//
// The CLI rejects an unknown --hw before it reaches lookupProfile (see
// cmd/root.go), so this only backstops direct callers of the package.
const fallbackProfile = "rtx-4090"

// fallbackAlgo is the algorithm CalculateHashRate routes an unknown --algo
// through, and the profile whose hashrate keys define the advertised
// algorithm set. Same contract as fallbackProfile: the CLI rejects an
// unknown --algo before it gets here, so this only backstops direct callers.
const fallbackAlgo = "bcrypt"

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
	// Unknown algorithms are routed through the fallback algorithm path (see
	// CalculateHashRate). The fallback profile must therefore carry a
	// positive rate for it, otherwise that path silently returns 0.
	if rate, ok := fb.Hashrates[fallbackAlgo]; !ok || rate <= 0 {
		panic(fmt.Errorf("cost: fallback profile %q must have a positive %s hashrate", fallbackProfile, fallbackAlgo))
	}
	// AlgoNames advertises the fallback profile's key set, and the CLI
	// validates --algo against it. A profile missing one of those keys would
	// make CalculateHashRate route a *valid* --algo through the fallback
	// algorithm instead -- the same silent-downgrade false pass the --hw and
	// --algo checks exist to prevent, just triggered by a valid input. Every
	// profile therefore has to define exactly the same algorithms.
	for name, p := range f.Profiles {
		if len(p.Hashrates) != len(fb.Hashrates) {
			panic(fmt.Errorf("cost: profile %q defines %d algorithms, want the same %d as %q",
				name, len(p.Hashrates), len(fb.Hashrates), fallbackProfile))
		}
		for algo := range fb.Hashrates {
			if rate, ok := p.Hashrates[algo]; !ok || rate <= 0 {
				panic(fmt.Errorf("cost: profile %q must have a positive %s hashrate", name, algo))
			}
		}
	}
	Profiles = f.Profiles
}

// normalizeProfileKey maps a user-supplied --hw value to a lookup key, so
// lookupProfile and ResolveProfileName agree on case and surrounding space.
func normalizeProfileKey(hw string) string {
	return strings.ToLower(strings.TrimSpace(hw))
}

func lookupProfile(hw string) HardwareProfile {
	if p, ok := Profiles[normalizeProfileKey(hw)]; ok {
		return p
	}
	return Profiles[fallbackProfile]
}

// ProfileNames returns every known --hw value, sorted alphabetically for a
// stable order. The CLI builds both its --hw help string and its
// unknown-profile error from this, so the advertised set is derived from
// hashrates.yaml rather than duplicated alongside it.
func ProfileNames() []string {
	names := make([]string, 0, len(Profiles))
	for name := range Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveProfileName maps a user-supplied --hw value to the canonical
// profile key actually used for the calculation. The bool reports whether
// the input matched a known profile; when it is false the returned name is
// the fallback profile, so callers can warn the user that their numbers
// describe different hardware than they typed.
func ResolveProfileName(hw string) (name string, known bool) {
	key := normalizeProfileKey(hw)
	if _, ok := Profiles[key]; ok {
		return key, true
	}
	return fallbackProfile, false
}

// normalizeAlgoKey maps a user-supplied --algo value to a lookup key, so
// ResolveAlgoName and CalculateHashRate agree on case and surrounding space.
func normalizeAlgoKey(algo string) string {
	return strings.ToLower(strings.TrimSpace(algo))
}

// AlgoNames returns every known --algo value, sorted alphabetically for a
// stable order. The CLI builds both its --algo help string and its
// unknown-algorithm error from this, so the advertised set is derived from
// hashrates.yaml rather than duplicated alongside it. init guarantees every
// profile defines the same set, so reading it off the fallback profile is
// enough.
func AlgoNames() []string {
	fb := Profiles[fallbackProfile]
	names := make([]string, 0, len(fb.Hashrates))
	for name := range fb.Hashrates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveAlgoName maps a user-supplied --algo value to the canonical key
// actually used for the calculation. The bool reports whether the input
// matched a known algorithm; when it is false the returned name is
// fallbackAlgo, mirroring what CalculateHashRate would silently do.
func ResolveAlgoName(algo string) (name string, known bool) {
	key := normalizeAlgoKey(algo)
	if _, ok := Profiles[fallbackProfile].Hashrates[key]; ok {
		return key, true
	}
	return fallbackAlgo, false
}

// AlgoTuning describes which tuning parameters an algorithm actually
// consumes, and the work-factor range CalculateHashRate can model.
//
// The CLI validates --cost and --memory against this so it never reports a
// parameter it did not apply: a fast algorithm has no work factor at all, and
// only argon2id reads --memory.
type AlgoTuning struct {
	UsesWorkFactor bool
	MinWorkFactor  int
	MaxWorkFactor  int // 0 means no upper bound
	UsesMemory     bool
}

// TuningFor returns the tuning parameters algo consumes. An unknown algorithm
// reports the fallback's, matching how CalculateHashRate treats it.
func TuningFor(algo string) AlgoTuning {
	switch normalizeAlgoKey(algo) {
	case "bcrypt":
		// 4..31 is bcrypt's own cost range: the cost is stored in two
		// decimal digits of the hash prefix and the reference implementation
		// refuses anything outside it. A --cost of 99 is not a slow bcrypt,
		// it is not a bcrypt hash at all.
		return AlgoTuning{UsesWorkFactor: true, MinWorkFactor: 4, MaxWorkFactor: 31}
	case "argon2id":
		// Argon2 caps t at 2^32-1, far past anything worth modeling, so only
		// the floor is enforced. t=0 is not a valid Argon2 parameter.
		return AlgoTuning{UsesWorkFactor: true, MinWorkFactor: 1, UsesMemory: true}
	default:
		// md5, sha1, sha256 and ntlm are single-pass: they have no work
		// factor and no memory parameter to tune.
		return AlgoTuning{}
	}
}

// argon2BaselineMemoryMB is the memory parameter the Argon2id baseline
// hashrates in hashrates.yaml are calibrated against. Doubling memory
// roughly halves attacker throughput on memory-bandwidth-bound GPUs.
const argon2BaselineMemoryMB = 64

// CalculateHashRate returns the attacker's hashes-per-second for the
// given hardware and algorithm parameters.
//
// memoryMB is only consulted for argon2id; passing 0 (or any value <=
// the baseline) leaves the rate at the YAML baseline (m=64MB).
func CalculateHashRate(hw, algo string, workFactor, memoryMB int) float64 {
	p := lookupProfile(hw)
	algo = normalizeAlgoKey(algo)

	if _, ok := p.Hashrates[algo]; !ok {
		// Unknown algorithm: route through bcrypt entirely (rate AND
		// scaling). Returning the bare bcrypt baseline without applying
		// the cost-factor scaling would silently overestimate the
		// attacker for any workFactor > 5.
		//
		// The CLI rejects an unknown --algo before it reaches here (see
		// cmd/root.go), because bcrypt is the *slowest* algorithm: a typo
		// landing here stretches the crack time and turns --fail-under-time
		// into a false pass. This only backstops direct callers.
		algo = fallbackAlgo
	}
	base := p.Hashrates[algo]

	// The CLI validates --cost against TuningFor's range, so the floors below
	// only backstop direct callers passing a work factor the algorithm itself
	// cannot represent. They clamp rather than extrapolate downward, which
	// keeps the modeled attacker at the floor instead of an arbitrarily fast
	// one, but a caller that relies on that is asking about a hash that does
	// not exist.
	tuning := TuningFor(algo)
	if workFactor < tuning.MinWorkFactor {
		workFactor = tuning.MinWorkFactor
	}

	switch algo {
	case "bcrypt":
		// Bcrypt cost is exponential (2^cost). Baseline is cost=5, so
		// cost=10 is 2^5 = 32 times slower and cost=4, the one valid cost
		// below the baseline, is twice as fast.
		return base / math.Pow(2, float64(workFactor-5))
	case "argon2id":
		timeFactor := float64(workFactor)
		memFactor := 1.0
		if memoryMB > argon2BaselineMemoryMB {
			memFactor = float64(memoryMB) / float64(argon2BaselineMemoryMB)
		}
		return base / (timeFactor * memFactor)
	default:
		return base
	}
}

func TotalCost(hw string, timeInSeconds float64) float64 {
	p := lookupProfile(hw)
	// Owned hardware is free regardless of crack time. Returning 0 explicitly
	// avoids Inf*0 = NaN when timeInSeconds overflows to +Inf for an
	// astronomically long password, which would otherwise poison the cost as
	// NaN (unmarshalable to JSON, rendered as "$NaN" in text).
	if p.CostPerHourUSD == 0 {
		return 0
	}
	hours := timeInSeconds / 3600.0
	return hours * p.CostPerHourUSD
}
