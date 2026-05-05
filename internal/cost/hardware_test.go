package cost

import (
	"math"
	"testing"
)

func TestCalculateHashRate_KnownProfile(t *testing.T) {
	got := CalculateHashRate("rtx-4090", "md5", 10)
	want := Profiles["rtx-4090"].BaseHashesMD5
	if got != want {
		t.Errorf("rtx-4090 md5 = %v, want %v", got, want)
	}
}

func TestCalculateHashRate_UnknownProfileFallsBackTo4090(t *testing.T) {
	got := CalculateHashRate("does-not-exist", "sha256", 10)
	want := Profiles["rtx-4090"].BaseHashesSHA256
	if got != want {
		t.Errorf("unknown hw fallback sha256 = %v, want %v", got, want)
	}
}

func TestCalculateHashRate_BcryptScalesByCost(t *testing.T) {
	base := Profiles["rtx-4090"].BaseHashesBcrypt // baseline at cost=5

	at10 := CalculateHashRate("rtx-4090", "bcrypt", 10)
	wantAt10 := base / math.Pow(2, 5)
	if at10 != wantAt10 {
		t.Errorf("bcrypt cost=10 = %v, want %v", at10, wantAt10)
	}

	atBaseline := CalculateHashRate("rtx-4090", "bcrypt", 5)
	if atBaseline != base {
		t.Errorf("bcrypt cost=5 = %v, want baseline %v", atBaseline, base)
	}

	// Below the baseline, the factor is clamped to 1 (never faster than baseline)
	atLow := CalculateHashRate("rtx-4090", "bcrypt", 1)
	if atLow != base {
		t.Errorf("bcrypt cost=1 = %v, want clamp to baseline %v", atLow, base)
	}
}

func TestCalculateHashRate_Argon2LinearScaling(t *testing.T) {
	base := Profiles["rtx-4090"].BaseHashesArgon2
	got := CalculateHashRate("rtx-4090", "argon2id", 4)
	want := base / 4
	if got != want {
		t.Errorf("argon2id workFactor=4 = %v, want %v", got, want)
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
