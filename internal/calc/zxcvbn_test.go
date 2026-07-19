package calc

import (
	"math"
	"math/big"
	"testing"
)

func TestZxcvbn_PatternPasswordIsWeak(t *testing.T) {
	// The classic case: naive R^L rates "P@ssw0rd!" as strong, but zxcvbn
	// recognizes the dictionary word + l33t and returns a tiny guess count.
	g, score := Zxcvbn("P@ssw0rd!")
	if g.Cmp(big.NewInt(1_000_000)) >= 0 {
		t.Errorf("P@ssw0rd! should be cheap to guess, got %s guesses", g)
	}
	if score > 2 {
		t.Errorf("P@ssw0rd! score = %d, want a weak score (<=2)", score)
	}
}

func TestZxcvbn_RandomPasswordIsStrong(t *testing.T) {
	g, score := Zxcvbn("7#kR9!qZ2@wX4vB")
	if g.Cmp(big.NewInt(1_000_000_000)) < 0 {
		t.Errorf("a random 15-char password should need many guesses, got %s", g)
	}
	if score < 3 {
		t.Errorf("random password score = %d, want a strong score (>=3)", score)
	}
}

func TestZxcvbn_FeedsIntoPipeline(t *testing.T) {
	// A zxcvbn guess count should flow through FromGuesses like --guesses does:
	// CharSpace 0 (external estimate) and a matching combination count.
	g, _ := Zxcvbn("password")
	res := FromGuesses(g, "password")
	if res.CharSpace != 0 {
		t.Errorf("external estimate should have CharSpace 0, got %d", res.CharSpace)
	}
	if res.Combinations.Cmp(g) != 0 {
		t.Errorf("combinations should equal the guess count %s, got %s", g, res.Combinations)
	}
}

func TestGuessesToBigInt(t *testing.T) {
	if got := guessesToBigInt(math.Inf(1)); got.BitLen() != 1025 {
		t.Errorf("+Inf should map to the 2^1024 sentinel, got BitLen %d", got.BitLen())
	}
	if got := guessesToBigInt(math.NaN()); got.Sign() != 0 {
		t.Errorf("NaN should map to 0, got %s", got)
	}
	if got := guessesToBigInt(-5); got.Sign() != 0 {
		t.Errorf("negative should map to 0, got %s", got)
	}
	if got := guessesToBigInt(12345.9); got.Cmp(big.NewInt(12345)) != 0 {
		t.Errorf("finite should truncate to 12345, got %s", got)
	}
}
