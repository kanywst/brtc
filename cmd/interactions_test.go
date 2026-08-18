package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// runBRTC executes rootCmd with the given args, capturing any error, and
// resets every flag-bound global afterwards. Cobra binds flags to package
// globals and only overwrites them when the flag is present, so without this
// reset a --all-hw or --hibp set by one case would leak into the next.
func runBRTC(t *testing.T, args ...string) error {
	t.Helper()
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SilenceUsage = false
		hwProfile, algo, workFactor, memoryStr = "rtx-4090", "bcrypt", 10, ""
		externalGuesses, useZxcvbn, useHIBP = "", false, false
		budget, outputFormat, failUnderTime, allHW = "", "tui", "", false
		failUnderEntropy, failOnBreach = 0, false
	})
	return rootCmd.Execute()
}

func TestFlagInteractions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string // substring; "" means expect success
	}{
		{"zxcvbn and guesses are mutually exclusive", []string{"pw", "--zxcvbn", "--guesses", "1e5"}, "mutually exclusive"},
		{"all-hw rejects hibp", []string{"pw", "--all-hw", "--hibp"}, "--hibp cannot be combined with --all-hw"},
		{"all-hw rejects budget", []string{"pw", "--all-hw", "--budget", "100usd"}, "--budget cannot be combined with --all-hw"},
		{"all-hw rejects sarif", []string{"pw", "--all-hw", "-o", "sarif"}, "sarif output is not supported"},
		{"all-hw rejects fail-under-entropy", []string{"pw", "--all-hw", "--fail-under-entropy", "60"}, "--fail-under-entropy cannot be combined with --all-hw"},
		{"all-hw rejects fail-on-breach", []string{"pw", "--all-hw", "--fail-on-breach"}, "--fail-on-breach cannot be combined with --all-hw"},
		{"negative fail-under-entropy is rejected", []string{"pw", "--fail-under-entropy", "-1"}, "must be a finite, non-negative number of bits"},
		// NaN would fail every comparison in checkGates, silently disabling the
		// gate; Inf would fail every password. Both are rejected up front.
		{"NaN fail-under-entropy is rejected", []string{"pw", "--fail-under-entropy", "NaN"}, "must be a finite, non-negative number of bits"},
		{"Inf fail-under-entropy is rejected", []string{"pw", "--fail-under-entropy", "Inf"}, "must be a finite, non-negative number of bits"},
		{"fail-on-breach needs a password", []string{"--guesses", "1e5", "--fail-on-breach"}, "--fail-on-breach needs a password"},
		{"fail-under-entropy fails a weak password", []string{"pw", "-o", "json", "--fail-under-entropy", "60"}, "estimated entropy"},
		{"fail-under-entropy passes a strong password", []string{"cX7#qLm2!vTr9$Wz", "-o", "json", "--fail-under-entropy", "60"}, ""},
		{"zxcvbn alone succeeds", []string{"pw", "--zxcvbn", "-o", "json"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runBRTC(t, tt.args...)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("args %v: got error %v, want substring %q", tt.args, err, tt.wantErr)
			}
		})
	}
}
