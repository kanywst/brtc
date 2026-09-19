package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// runBRTC executes rootCmd with the given args, capturing any error, and
// resets every flag-bound global afterwards. Cobra binds flags to package
// globals and only overwrites them when the flag is present, so without this
// reset a --all-hw or --hibp set by one case would leak into the next.
//
// The per-flag Changed bits are reset too. They are sticky across Execute
// calls on a shared rootCmd, which a real one-shot process never sees, so
// leaving them set makes a case that relies on Flags().Changed — the --cost
// and --hw checks — read a flag as user-supplied because an earlier case
// passed it.
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
		rootCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
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
		// A typo must not fall back to rtx-4090: slower hardware means a
		// longer crack time, which turns --fail-under-time into a false pass.
		{"unknown hw profile is rejected", []string{"pw", "-o", "json", "--hw", "rtx-4900"}, `unknown hardware profile "rtx-4900"`},
		{"unknown hw error lists the known profiles", []string{"pw", "-o", "json", "--hw", "h100"}, "known profiles: aws-p5.48xlarge,"},
		{"a known hw profile is accepted case-insensitively", []string{"pw", "-o", "json", "--hw", "RTX-5090"}, ""},
		// Same class of bug on --algo, and worse: an unknown algorithm used to
		// be modeled as bcrypt, the slowest one in the table, while the JSON
		// and SARIF reports echoed the typo back as the algorithm modeled.
		{"unknown algo is rejected", []string{"pw", "-o", "json", "--algo", "sha-256"}, `unknown hash algorithm "sha-256"`},
		{"unknown algo error lists the known algorithms", []string{"pw", "-o", "json", "--algo", "scrypt"}, "known algorithms: argon2id,"},
		{"a known algo is accepted case-insensitively", []string{"pw", "-o", "json", "--algo", "MD5"}, ""},
		// --all-hw feeds algo straight into buildMatrix, so it needs the same
		// guard as the single-profile path.
		{"unknown algo is rejected under --all-hw", []string{"pw", "--all-hw", "--algo", "scrypt"}, `unknown hash algorithm "scrypt"`},
		// --cost and --memory used to be accepted for algorithms that consume
		// neither, so the report named a work factor or a memory size the
		// calculation never applied.
		{"cost is rejected for a single-pass algo", []string{"pw", "-o", "json", "--algo", "md5", "--cost", "12"}, "--cost does not apply to md5"},
		{"cost error names the algos it applies to", []string{"pw", "-o", "json", "--algo", "ntlm", "--cost", "12"}, "applies to argon2id and bcrypt"},
		{"memory is rejected for a non-argon2id algo", []string{"pw", "-o", "json", "--algo", "bcrypt", "--memory", "1g"}, "--memory does not apply to bcrypt"},
		{"cost below bcrypt's minimum is rejected", []string{"pw", "-o", "json", "--algo", "bcrypt", "--cost", "3"}, "below bcrypt's minimum of 4"},
		{"cost above bcrypt's maximum is rejected", []string{"pw", "-o", "json", "--algo", "bcrypt", "--cost", "99"}, "above bcrypt's maximum of 31"},
		{"cost below argon2id's minimum is rejected", []string{"pw", "-o", "json", "--algo", "argon2id", "--cost", "0"}, "below argon2id's minimum of 1"},
		{"zero memory is rejected", []string{"pw", "-o", "json", "--algo", "argon2id", "--memory", "0m"}, "--memory must be at least 1MB"},
		// The default --cost is not "set", so it must not trip the check for an
		// algorithm that has no work factor.
		{"an unset cost is fine for a single-pass algo", []string{"pw", "-o", "json", "--algo", "md5"}, ""},
		{"bcrypt's boundary costs are accepted", []string{"pw", "-o", "json", "--algo", "bcrypt", "--cost", "31"}, ""},
		{"argon2id accepts memory", []string{"pw", "-o", "json", "--algo", "argon2id", "--memory", "128m"}, ""},
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
