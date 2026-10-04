package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/kanywst/brtc/v2/internal/cost"
)

// setNow pins the clock staleness is judged against for one test.
func setNow(t *testing.T, at time.Time) {
	t.Helper()
	orig := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = orig })
}

// latestReviewed is the newest last_reviewed across every profile, so a
// clock past it plus a year makes all of them stale however they are bumped.
func latestReviewed() time.Time {
	var latest time.Time
	for _, n := range cost.ProfileNames() {
		if r := cost.BaselineFor(n).Reviewed; r.After(latest) {
			latest = r
		}
	}
	return latest
}

func TestStaleBaselineWarning(t *testing.T) {
	reviewed := cost.BaselineFor("rtx-4090").Reviewed

	t.Run("fresh", func(t *testing.T) {
		setNow(t, reviewed.AddDate(0, 6, 0))
		stderr, err := runBRTCStderr(t, "pw", "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if strings.Contains(stderr, "warning") {
			t.Errorf("unexpected warning %q", stderr)
		}
	})

	t.Run("stale", func(t *testing.T) {
		setNow(t, reviewed.AddDate(1, 0, 1))
		stderr, err := runBRTCStderr(t, "pw", "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		want := "baseline for rtx-4090 was last reviewed " + reviewed.Format(time.DateOnly)
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr %q does not contain %q", stderr, want)
		}
	})

	// The comparison view uses every profile's baseline, so it names each
	// stale one, grouped into one line per review date.
	t.Run("all-hw", func(t *testing.T) {
		setNow(t, latestReviewed().AddDate(1, 0, 1))
		stderr, err := runBRTCStderr(t, "pw", "--all-hw", "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		for _, name := range cost.ProfileNames() {
			if !strings.Contains(stderr, name) {
				t.Errorf("no warning for %s in %q", name, stderr)
			}
		}
		dates := map[time.Time]bool{}
		for _, n := range cost.ProfileNames() {
			dates[cost.BaselineFor(n).Reviewed] = true
		}
		if got := strings.Count(stderr, "warning:"); got != len(dates) {
			t.Errorf("got %d warning lines, want one per review date (%d)", got, len(dates))
		}
	})
}

func TestStaleBaselineDoesNotFailTheGate(t *testing.T) {
	setNow(t, cost.BaselineFor("rtx-4090").Reviewed.AddDate(5, 0, 0))
	if err := runBRTC(t, "cX7#qLm2!vTr9$Wz", "-o", "json", "--fail-under-entropy", "60"); err != nil {
		t.Errorf("a stale baseline must warn, not fail a passing gate: %v", err)
	}
}
