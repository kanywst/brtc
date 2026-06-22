package cmd

import "testing"

func TestResolveOutputFormat(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		explicit  bool
		tty       bool
		want      string
	}{
		{"tui on a terminal stays tui", "tui", false, true, "tui"},
		{"tui piped downgrades to json", "tui", false, false, "json"},
		{"explicit tui is honoured even when piped", "tui", true, false, "tui"},
		{"json piped stays json", "json", false, false, "json"},
		{"sarif piped stays sarif", "sarif", false, false, "sarif"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveOutputFormat(tt.requested, tt.explicit, tt.tty); got != tt.want {
				t.Errorf("resolveOutputFormat(%q, %v, %v) = %q, want %q",
					tt.requested, tt.explicit, tt.tty, got, tt.want)
			}
		})
	}
}
