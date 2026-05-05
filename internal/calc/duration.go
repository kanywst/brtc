package calc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var durationRe = regexp.MustCompile(`^(\d+)([smhdwy])$`)

// ParseDuration converts a CLI duration string into seconds.
//
// Accepted units: s (second), m (minute), h (hour), d (day),
// w (week), y (year, 365 days). Single integer + single unit only;
// composite forms like "1h30m" are intentionally not supported because
// the gatekeeper threshold reads more clearly as a single magnitude.
func ParseDuration(d string) (float64, error) {
	d = strings.ToLower(strings.TrimSpace(d))
	matches := durationRe.FindStringSubmatch(d)
	if len(matches) != 3 {
		return 0, fmt.Errorf("expected format like 30d, 1y, 12h (units: s,m,h,d,w,y)")
	}
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}
	switch matches[2] {
	case "s":
		return val, nil
	case "m":
		return val * 60, nil
	case "h":
		return val * 3600, nil
	case "d":
		return val * 86400, nil
	case "w":
		return val * 604800, nil
	case "y":
		return val * 31536000, nil
	}
	return 0, fmt.Errorf("unknown unit %s", matches[2])
}
