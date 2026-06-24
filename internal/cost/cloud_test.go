package cost

import (
	"math/big"
	"testing"
)

func TestParseBudget(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{"", 0, false},
		{"1000", 1000, false},
		{"1000usd", 1000, false},
		{"$2500", 2500, false},
		{"  500 USD  ", 500, false},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseBudget(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseBudget(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaxLengthForBudget_OwnedHardwareReturnsSentinel(t *testing.T) {
	// mac-m3 has CostPerHourUSD = 0, so any budget covers infinite time.
	got := MaxLengthForBudget(100, "mac-m3", "md5", 10, 0, 94)
	if got != UnlimitedBudgetChars {
		t.Errorf("owned hardware should return the unlimited sentinel, got %d", got)
	}
}

func TestMaxLengthForBudget_ScalesWithBudget(t *testing.T) {
	small := MaxLengthForBudget(1, "rtx-4090", "md5", 10, 0, 94)
	large := MaxLengthForBudget(1_000_000, "rtx-4090", "md5", 10, 0, 94)
	if large <= small {
		t.Errorf("larger budget should crack longer passwords: small=%d large=%d", small, large)
	}
}

func TestMaxLengthForBudget_ZeroBudget(t *testing.T) {
	if got := MaxLengthForBudget(0, "rtx-4090", "md5", 10, 0, 94); got != 0 {
		t.Errorf("zero budget should return 0, got %d", got)
	}
}

func TestMaxLengthForBudget_TinyBudgetClampsToZero(t *testing.T) {
	// A budget too small to afford even one character must clamp to exactly 0,
	// never a negative length.
	if got := MaxLengthForBudget(0.0001, "aws-p5.48xlarge", "argon2id", 10, 0, 94); got != 0 {
		t.Errorf("tiny budget should clamp to 0, got %d", got)
	}
}

func TestMaxLengthForBudget_HugeBudgetStaysFinite(t *testing.T) {
	// An astronomically large budget on rented hardware must yield a large but
	// finite length, never overflow to garbage or hit the owned-hardware
	// unlimited sentinel.
	got := MaxLengthForBudget(1e300, "rtx-4090", "md5", 10, 0, 94)
	if got <= 0 || got == UnlimitedBudgetChars {
		t.Errorf("huge budget on rented hw = %d, want a large finite length", got)
	}
}

func TestMinLengthForTime(t *testing.T) {
	// Zero threshold and a trivial charset have no answer.
	if got := MinLengthForTime(0, "rtx-4090", "md5", 10, 0, 94); got != 0 {
		t.Errorf("zero threshold should return 0, got %d", got)
	}
	if got := MinLengthForTime(31536000, "rtx-4090", "md5", 10, 0, 1); got != 0 {
		t.Errorf("trivial charset should return 0, got %d", got)
	}

	// An extreme threshold overflows float64; it must clamp to the sentinel,
	// not wrap around to a 1-char recommendation.
	if got := MinLengthForTime(1e300, "rtx-4090", "md5", 10, 0, 94); got != maxRecommendedChars {
		t.Errorf("extreme threshold should clamp to %d, got %d", maxRecommendedChars, got)
	}

	// The recommended length must actually survive the threshold, and one
	// character shorter must not — i.e. it is the *minimum* safe length.
	const oneYear = 31536000.0
	hw, algo, wf, cs := "rtx-4090", "md5", 10, 94
	n := MinLengthForTime(oneYear, hw, algo, wf, 0, cs)
	if n < 1 {
		t.Fatalf("expected a positive length, got %d", n)
	}
	hr := CalculateHashRate(hw, algo, wf, 0)
	ttcAt := func(l int) float64 {
		c := new(big.Int).Exp(big.NewInt(int64(cs)), big.NewInt(int64(l)), nil)
		return calcTimeToCrack(c, hr)
	}
	if ttcAt(n) < oneYear {
		t.Errorf("recommended length %d does not survive one year", n)
	}
	if n > 1 && ttcAt(n-1) >= oneYear {
		t.Errorf("length %d already survives, so %d is not the minimum", n-1, n)
	}

	// A stronger attacker (more hashes/sec) must require at least as many chars.
	slow := MinLengthForTime(oneYear, "raspberry-pi-4", algo, wf, 0, cs)
	fast := MinLengthForTime(oneYear, "aws-p5.48xlarge", algo, wf, 0, cs)
	if fast < slow {
		t.Errorf("faster attacker should need >= chars: slow=%d fast=%d", slow, fast)
	}
}

// calcTimeToCrack mirrors calc.TimeToCrack locally to keep this package's
// tests free of an import cycle while still checking the inverse relationship.
func calcTimeToCrack(combinations *big.Int, hashRate float64) float64 {
	c := new(big.Float).SetInt(combinations)
	half := new(big.Float).Quo(c, big.NewFloat(2))
	t := new(big.Float).Quo(half, big.NewFloat(hashRate))
	f, _ := t.Float64()
	return f
}
