package calc

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var memoryRe = regexp.MustCompile(`^(\d+)([mg])?$`)

// ParseMemory parses an Argon2id memory parameter from CLI input.
//
// Accepts a bare integer (interpreted as MB), "<n>m" for MB, or "<n>g"
// for GB. KB is intentionally not supported because Argon2id is never
// configured below MB granularity in practice and accepting it would
// only make the unit ambiguous.
func ParseMemory(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	m := memoryRe.FindStringSubmatch(s)
	if len(m) != 3 {
		return 0, fmt.Errorf("expected like 64m, 1g, or bare integer (MB)")
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, err
	}
	switch m[2] {
	case "", "m":
		return n, nil
	case "g":
		return n * 1024, nil
	}
	return 0, fmt.Errorf("unknown unit %s", m[2])
}

// ParseGuesses parses an external guess count, typically the `guesses`
// field from a zxcvbn report. Accepts decimal integers ("12345") and
// scientific notation ("1.234e+10"). Returns *big.Int because guess
// counts can comfortably exceed int64 for long passwords.
func ParseGuesses(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("guesses: empty")
	}
	if n, ok := new(big.Int).SetString(s, 10); ok {
		if n.Sign() < 0 {
			return nil, fmt.Errorf("guesses: must be non-negative")
		}
		return n, nil
	}
	f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return nil, fmt.Errorf("guesses: cannot parse %q", s)
	}
	if f.Sign() < 0 {
		return nil, fmt.Errorf("guesses: must be non-negative")
	}
	if f.IsInf() {
		// big.ParseFloat happily accepts "inf"/"+inf", and big.Float.Int
		// returns a nil *big.Int for infinities — which would then panic
		// downstream in TimeToCrack via big.Float.SetInt(nil).
		return nil, fmt.Errorf("guesses: must be finite")
	}
	n, _ := f.Int(nil)
	return n, nil
}
