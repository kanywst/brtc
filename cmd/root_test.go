package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// A runtime error from RunE should surface the error itself, but not the
// full flag usage block. Regression guard for the previous behaviour where
// cobra printed the usage dump (and the error a second time) on any failure.
func TestRunE_RuntimeErrorDoesNotPrintUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"abc", "--budget", "not-a-number"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error for an invalid --budget value")
	}

	combined := out.String() + errOut.String()
	if strings.Contains(combined, "Usage:") {
		t.Errorf("runtime error should not print the usage block, got:\n%s", combined)
	}
}
