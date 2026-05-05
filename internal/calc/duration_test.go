package calc

import "testing"

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{"30s", 30, false},
		{"1m", 60, false},
		{"12h", 12 * 3600, false},
		{"30d", 30 * 86400, false},
		{"2w", 2 * 604800, false},
		{"1y", 31536000, false},
		{"  5D  ", 5 * 86400, false}, // trim + case-insensitive

		{"", 0, true},
		{"30", 0, true},
		{"abc", 0, true},
		{"1h30m", 0, true}, // composite form rejected
		{"5x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseDuration(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got %v", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
