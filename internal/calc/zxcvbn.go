package calc

import (
	"math"
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
// maxZxcvbnRunes caps the input length before analysis. zxcvbn's pattern
// matching is super-linear on very long or highly repetitive inputs, so an
// attacker-supplied megabyte string could pin a CPU (DoS). Anything past a few
// hundred characters is already unguessable, so truncating loses no signal.
const maxZxcvbnRunes = 256

func Zxcvbn(password string) (guesses *big.Int, score int) {
	// Truncate by runes, not bytes, so a multi-byte character is never split
	// into invalid UTF-8 at the boundary.
	if runes := []rune(password); len(runes) > maxZxcvbnRunes {
		password = string(runes[:maxZxcvbnRunes])
	}
	r := zxcvbn.PasswordStrength(password, nil)
	return guessesToBigInt(r.Guesses), r.Score
}

// guessesToBigInt converts a zxcvbn float64 guess count to a *big.Int.
//
// For a very strong password r.Guesses can overflow to +Inf; big.Float.Int
// then yields a nil *big.Int, which naively collapses to 0 guesses — i.e.
// "instantly crackable", the exact opposite of the truth. Map +Inf to a huge
// sentinel (2^1024, which reads back as +Inf downstream) so an unguessable
// password stays unguessable, and clamp NaN / non-positive input to 0.
func guessesToBigInt(g float64) *big.Int {
	switch {
	case math.IsNaN(g) || g <= 0:
		return big.NewInt(0)
	case math.IsInf(g, 1):
		return new(big.Int).Lsh(big.NewInt(1), 1024)
	}
	// big.NewFloat -> Int truncates the fractional part, which is fine for a
	// guess count.
	n, _ := big.NewFloat(g).Int(nil)
	if n == nil || n.Sign() < 0 {
		return big.NewInt(0)
	}
	return n
}
