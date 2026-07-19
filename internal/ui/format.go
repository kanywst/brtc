package ui

import (
	"fmt"
	"math"
	"strings"
)

// ageOfUniverseYears is the standard cosmological figure (~13.8 billion
// years). brtc anchors astronomically long crack times against it because a
// bare "2414613469377.2 years" is meaningless to a human, whereas "175x the
// age of the universe" lands.
const ageOfUniverseYears = 13.8e9

// scaleUnits maps thresholds to their short-scale names, largest first, so a
// value can be rendered as e.g. "2.4 trillion" instead of a wall of digits.
var scaleUnits = []struct {
	v    float64
	name string
}{
	{1e33, "decillion"},
	{1e30, "nonillion"},
	{1e27, "octillion"},
	{1e24, "septillion"},
	{1e21, "sextillion"},
	{1e18, "quintillion"},
	{1e15, "quadrillion"},
	{1e12, "trillion"},
	{1e9, "billion"},
	{1e6, "million"},
	{1e3, "thousand"},
}

// humanScale renders a non-negative magnitude compactly: exact-ish for small
// values, named short-scale words up to decillion, and scientific notation
// beyond that (or for non-finite input). It carries no unit — callers append
// "years", "H/s", etc.
func humanScale(x float64) string {
	if math.IsInf(x, 1) || math.IsNaN(x) || x >= 1e34 {
		return fmt.Sprintf("%.1e", x)
	}
	if x < 0 {
		return "-" + humanScale(-x)
	}
	for _, u := range scaleUnits {
		if x >= u.v {
			return fmt.Sprintf("%.1f %s", x/u.v, u.name)
		}
	}
	if x >= 100 {
		return fmt.Sprintf("%.0f", x)
	}
	return fmt.Sprintf("%.1f", x)
}

// FormatDuration turns a number of seconds into a human-readable duration.
// Below a year it uses the familiar unit ladder; above it, it collapses the
// year count with humanScale and, once the count exceeds the age of the
// universe, anchors it against that figure so the number stays legible.
func FormatDuration(seconds float64) string {
	switch {
	case math.IsInf(seconds, 1):
		return "effectively forever"
	case seconds < 1:
		return "Less than a second"
	case seconds < 60:
		return fmt.Sprintf("%.1f seconds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%.1f minutes", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("%.1f hours", seconds/3600)
	case seconds < 31536000:
		return fmt.Sprintf("%.1f days", seconds/86400)
	}

	years := seconds / 31536000
	if years < 1000 {
		return fmt.Sprintf("%.1f years", years)
	}
	if years >= ageOfUniverseYears {
		return fmt.Sprintf("%s years (%sx the age of the universe)",
			humanScale(years), humanScale(years/ageOfUniverseYears))
	}
	return fmt.Sprintf("%s years", humanScale(years))
}

// FormatHashRate renders hashes-per-second with a binary-free SI-style suffix
// (kH/s, MH/s, GH/s, TH/s), because "164100000000 H/s" is unreadable next to
// "164.1 GH/s".
func FormatHashRate(hps float64) string {
	switch {
	case hps <= 0:
		return "0 H/s"
	case hps < 1e3:
		return fmt.Sprintf("%.0f H/s", hps)
	case hps < 1e6:
		return fmt.Sprintf("%.1f kH/s", hps/1e3)
	case hps < 1e9:
		return fmt.Sprintf("%.1f MH/s", hps/1e6)
	case hps < 1e12:
		return fmt.Sprintf("%.1f GH/s", hps/1e9)
	default:
		return fmt.Sprintf("%.1f TH/s", hps/1e12)
	}
}

// FormatCost renders a USD amount so it reads as money at every scale: cents
// with two decimals, thousands with separators, and huge sums as named
// short-scale words ("$6.35 quadrillion") rather than a "$...0.00" digit wall.
// A tiny non-zero cost collapses to "< $0.01" so a trivially cheap crack does
// not masquerade as free.
func FormatCost(usd float64) string {
	if math.IsInf(usd, 1) {
		return "effectively unbounded"
	}
	switch {
	case math.IsNaN(usd) || usd <= 0:
		// NaN reaches here from owned hardware (Inf crack time * $0/hr); a free
		// rig genuinely costs nothing. Guarding it also keeps NaN from slipping
		// past every comparison into "$NaN".
		return "$0.00"
	case usd < 0.01:
		return "< $0.01"
	case usd < 1e6:
		// Thousands separator, two decimals below $10k, whole dollars above.
		if usd < 1e4 {
			return "$" + addThousands(fmt.Sprintf("%.2f", usd))
		}
		return "$" + addThousands(fmt.Sprintf("%.0f", usd))
	default:
		return "$" + humanScale(usd)
	}
}

// FormatCount renders an integer with thousands separators (e.g. "9,659,365").
// Breach counts are exact integers, so they use this rather than humanScale,
// which would print a lossy "9.7 million" or a stray "42.0" for small values.
func FormatCount(n int) string {
	return addThousands(fmt.Sprintf("%d", n))
}

// addThousands inserts commas into the integer part of a plain decimal string
// like "6345.60" -> "6,345.60". It expects a non-negative, already-formatted
// number and leaves any fractional part untouched.
func addThousands(s string) string {
	intPart, frac := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, frac = s[:dot], s[dot:]
	}
	n := len(intPart)
	if n <= 3 {
		return intPart + frac
	}
	var b []byte
	for i, c := range []byte(intPart) {
		if i > 0 && (n-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, c)
	}
	return string(b) + frac
}
