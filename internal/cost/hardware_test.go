package cost

import (
	"math"
	"testing"
)

func TestCalculateHashRate_KnownProfile(t *testing.T) {
	got := CalculateHashRate("rtx-4090", "md5", 10, 0)
	want := Profiles["rtx-4090"].Hashrates["md5"]
	if got != want {
		t.Errorf("rtx-4090 md5 = %v, want %v", got, want)
	}
}

func TestCalculateHashRate_UnknownProfileFallsBackTo4090(t *testing.T) {
	got := CalculateHashRate("does-not-exist", "sha256", 10, 0)
	want := Profiles["rtx-4090"].Hashrates["sha256"]
	if got != want {
		t.Errorf("unknown hw fallback sha256 = %v, want %v", got, want)
	}
}

func TestCalculateHashRate_BcryptScalesByCost(t *testing.T) {
	base := Profiles["rtx-4090"].Hashrates["bcrypt"] // baseline at cost=5

	at10 := CalculateHashRate("rtx-4090", "bcrypt", 10, 0)
	wantAt10 := base / math.Pow(2, 5)
	if at10 != wantAt10 {
		t.Errorf("bcrypt cost=10 = %v, want %v", at10, wantAt10)
	}

	atBaseline := CalculateHashRate("rtx-4090", "bcrypt", 5, 0)
	if atBaseline != base {
		t.Errorf("bcrypt cost=5 = %v, want baseline %v", atBaseline, base)
	}

	// Below the baseline, the factor is clamped to 1 (never faster than baseline)
	atLow := CalculateHashRate("rtx-4090", "bcrypt", 1, 0)
	if atLow != base {
		t.Errorf("bcrypt cost=1 = %v, want clamp to baseline %v", atLow, base)
	}
}

func TestCalculateHashRate_Argon2LinearScaling(t *testing.T) {
	base := Profiles["rtx-4090"].Hashrates["argon2id"]
	got := CalculateHashRate("rtx-4090", "argon2id", 4, 0)
	want := base / 4
	if got != want {
		t.Errorf("argon2id workFactor=4 = %v, want %v", got, want)
	}
}

func TestCalculateHashRate_Argon2MemoryScaling(t *testing.T) {
	base := Profiles["rtx-4090"].Hashrates["argon2id"]

	// Default (0) keeps the YAML baseline at 64MB.
	if got := CalculateHashRate("rtx-4090", "argon2id", 1, 0); got != base {
		t.Errorf("argon2id memory=0 = %v, want baseline %v", got, base)
	}

	// 64MB matches the baseline, so no extra memory penalty.
	if got := CalculateHashRate("rtx-4090", "argon2id", 1, 64); got != base {
		t.Errorf("argon2id memory=64MB = %v, want baseline %v", got, base)
	}

	// 128MB doubles memory pressure, halving attacker throughput.
	if got := CalculateHashRate("rtx-4090", "argon2id", 1, 128); got != base/2 {
		t.Errorf("argon2id memory=128MB = %v, want %v", got, base/2)
	}

	// Below baseline does not speed the attacker up beyond the baseline.
	if got := CalculateHashRate("rtx-4090", "argon2id", 1, 32); got != base {
		t.Errorf("argon2id memory=32MB = %v, want clamp to baseline %v", got, base)
	}

	// Memory and time factors compose multiplicatively.
	if got := CalculateHashRate("rtx-4090", "argon2id", 4, 128); got != base/(4*2) {
		t.Errorf("argon2id t=4 m=128MB = %v, want %v", got, base/(4*2))
	}
}

func TestResolveProfileName(t *testing.T) {
	// A known profile resolves to its canonical key and reports known=true.
	if name, known := ResolveProfileName("RTX-4090"); name != "rtx-4090" || !known {
		t.Errorf("ResolveProfileName(RTX-4090) = (%q, %v), want (rtx-4090, true)", name, known)
	}

	// An unknown profile falls back to rtx-4090 and reports known=false so
	// the caller can warn the user.
	if name, known := ResolveProfileName("nope"); name != "rtx-4090" || known {
		t.Errorf("ResolveProfileName(nope) = (%q, %v), want (rtx-4090, false)", name, known)
	}
}

func TestTotalCost(t *testing.T) {
	// rtx-4090 is $0.30/hour. 1 hour = $0.30.
	got := TotalCost("rtx-4090", 3600)
	if math.Abs(got-0.30) > 1e-9 {
		t.Errorf("TotalCost(rtx-4090, 1h) = %v, want 0.30", got)
	}

	// Owned hardware costs nothing regardless of duration.
	if TotalCost("mac-m3", 86400) != 0 {
		t.Error("mac-m3 should report $0 cost")
	}
}

func TestProfilesContainAdvertisedNames(t *testing.T) {
	// Names exposed via the --hw flag and README must all be in the map;
	// otherwise users get a silent fallback to rtx-4090 (a much faster GPU)
	// and their cost numbers become a lie.
	advertised := []string{
		"rtx-4090", "rtx-3060", "gtx-1080ti",
		"mac-m3", "mac-m3-max",
		"cpu-standard", "aws-p5.48xlarge", "raspberry-pi-4",
	}
	for _, name := range advertised {
		if _, ok := Profiles[name]; !ok {
			t.Errorf("hardware profile %q is advertised but missing from Profiles", name)
		}
	}
}

// Each profile in hashrates.yaml must carry hashrates for every algorithm
// the CLI exposes via --algo, plus a citation. Catches data drift where a
// new algorithm or profile is added but the YAML row is left half-filled.
func TestProfilesAreComplete(t *testing.T) {
	requiredAlgos := []string{"md5", "sha256", "bcrypt", "argon2id"}

	for name, p := range Profiles {
		t.Run(name, func(t *testing.T) {
			if p.Name == "" {
				t.Errorf("profile %q has empty display name", name)
			}
			if p.Source == "" {
				t.Errorf("profile %q must cite a source URL or note", name)
			}
			if p.LastReviewed == "" {
				t.Errorf("profile %q must set last_reviewed", name)
			}
			for _, algo := range requiredAlgos {
				if rate, ok := p.Hashrates[algo]; !ok || rate <= 0 {
					t.Errorf("profile %q is missing or has non-positive hashrate for %s", name, algo)
				}
			}
		})
	}
}
