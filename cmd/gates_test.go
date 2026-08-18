package cmd

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestCheckGates(t *testing.T) {
	const year = 365 * 24 * 3600.0
	netErr := errors.New("dial tcp: no route to host")

	tests := []struct {
		name    string
		g       gates
		wantErr string // substring; "" means the gate passes
	}{
		{"no gate enabled passes a breached, weak password", gates{
			BreachCount: 5000, EntropyBits: 10, TimeToCrack: 1,
		}, ""},
		{"fail-on-breach alone fails a breached password", gates{
			FailOnBreach: true, BreachCount: 5000,
		}, "found in 5000 known breaches"},
		{"fail-on-breach passes a clean password", gates{
			FailOnBreach: true, BreachCount: 0,
		}, ""},
		// Predates --fail-on-breach and must keep working.
		{"fail-under-time still fails on a breach", gates{
			BreachCount: 3, ReqSecs: year, TimeToCrack: 100 * year,
		}, "found in 3 known breaches"},
		{"an unevaluated lookup fails the breach gate", gates{
			FailOnBreach: true, BreachErr: netErr,
		}, "HIBP check could not be completed"},
		{"a failed lookup without the breach gate is ignored", gates{
			BreachErr: netErr, ReqSecs: year, TimeToCrack: 100 * year,
		}, ""},
		{"entropy below the threshold fails", gates{
			MinEntropy: 60, EntropyBits: 42.4,
		}, "estimated entropy (42.4 bits) is less than required (60.0 bits)"},
		{"entropy at the threshold passes", gates{
			MinEntropy: 60, EntropyBits: 60,
		}, ""},
		{"NaN entropy fails rather than passes", gates{
			MinEntropy: 60, EntropyBits: math.NaN(),
		}, "estimated entropy"},
		{"an unset entropy threshold ignores low entropy", gates{
			MinEntropy: 0, EntropyBits: 1,
		}, ""},
		{"crack time below the threshold fails", gates{
			ReqSecs: year, TimeToCrack: 60,
		}, "estimated crack time"},
		{"crack time above the threshold passes", gates{
			ReqSecs: year, TimeToCrack: 100 * year,
		}, ""},
		{"breach outranks the entropy and time gates", gates{
			FailOnBreach: true, BreachCount: 3,
			MinEntropy: 60, EntropyBits: 20, ReqSecs: year, TimeToCrack: 1,
		}, "known breaches"},
		{"entropy outranks the time gate", gates{
			MinEntropy: 60, EntropyBits: 20, ReqSecs: year, TimeToCrack: 1,
		}, "estimated entropy"},
		{"every gate enabled and satisfied passes", gates{
			FailOnBreach: true, BreachCount: 0,
			MinEntropy: 60, EntropyBits: 80, ReqSecs: year, TimeToCrack: 100 * year,
		}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkGates(tt.g)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected the gate to pass, got error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("got error %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestCheckGatesWrapsBreachError(t *testing.T) {
	netErr := errors.New("dial tcp: no route to host")
	err := checkGates(gates{FailOnBreach: true, BreachErr: netErr})
	if !errors.Is(err, netErr) {
		t.Fatalf("got %v, want an error wrapping %v", err, netErr)
	}
}
