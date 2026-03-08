package calc

import (
	"math/big"
	"testing"
)

func TestTimeToCrack(t *testing.T) {
	// Let's say we have 100 combinations. And Hashrate is 10 H/s.
	// T_avg = (100 / 2) / 10 = 50 / 10 = 5 seconds
	combinations := big.NewInt(100)
	hr := 10.0
	expectedSecs := 5.0

	secs := TimeToCrack(combinations, hr)
	if secs != expectedSecs {
		t.Errorf("expected %f seconds, got %f", expectedSecs, secs)
	}

	// Test zero combinations
	secsZero := TimeToCrack(big.NewInt(0), hr)
	if secsZero != 0.0 {
		t.Errorf("expected %f seconds, got %f", 0.0, secsZero)
	}
}
