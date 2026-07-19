package calc

import (
	"math/big"

	"github.com/trustelem/zxcvbn"
)

// Zxcvbn runs Dropbox's zxcvbn pattern-based estimator over the password and
// returns its guess count as a *big.Int plus the 0-4 strength score. Unlike
// the naive R^L entropy in Analyze, zxcvbn accounts for dictionary words,
// keyboard walks, dates, l33t substitutions, repeats, and sequences — so
// "P@ssw0rd!" scores as trivially guessable rather than "strong".
//
// The returned guess count is meant to be fed through FromGuesses, giving the
// same downstream time/cost pipeline the external --guesses flag uses.
func Zxcvbn(password string) (guesses *big.Int, score int) {
	r := zxcvbn.PasswordStrength(password, nil)
	// r.Guesses is a float64; convert without losing the magnitude for large
	// counts. big.NewFloat -> Int truncates the fractional part, which is fine
	// for a guess count.
	g, _ := big.NewFloat(r.Guesses).Int(nil)
	if g == nil || g.Sign() < 0 {
		g = big.NewInt(0)
	}
	return g, r.Score
}
