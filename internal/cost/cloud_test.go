package cost

import "testing"

func TestParseBudget(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{"", 0, false},
		{"1000", 1000, false},
		{"1000usd", 1000, false},
		{"$2500", 2500, false},
		{"  500 USD  ", 500, false},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseBudget(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseBudget(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaxLengthForBudget_OwnedHardwareReturnsSentinel(t *testing.T) {
	// mac-m3 has CostPerHourUSD = 0, so any budget covers infinite time.
	got := MaxLengthForBudget(100, "mac-m3", "md5", 10, 94)
	if got != 999 {
		t.Errorf("owned hardware should return 999 sentinel, got %d", got)
	}
}

func TestMaxLengthForBudget_ScalesWithBudget(t *testing.T) {
	small := MaxLengthForBudget(1, "rtx-4090", "md5", 10, 94)
	large := MaxLengthForBudget(1_000_000, "rtx-4090", "md5", 10, 94)
	if !(large > small) {
		t.Errorf("larger budget should crack longer passwords: small=%d large=%d", small, large)
	}
}

func TestMaxLengthForBudget_ZeroBudget(t *testing.T) {
	if got := MaxLengthForBudget(0, "rtx-4090", "md5", 10, 94); got != 0 {
		t.Errorf("zero budget should return 0, got %d", got)
	}
}
