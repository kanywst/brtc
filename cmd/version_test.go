package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// --version should print the build version and exit cleanly, without running
// the analysis. The release pipeline injects the real value via -ldflags.
func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--version"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("--version returned an error: %v", err)
	}
	if !strings.Contains(out.String(), version) {
		t.Errorf("--version output %q does not contain version %q", out.String(), version)
	}
}
