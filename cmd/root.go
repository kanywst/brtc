package cmd

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/kanywst/brtc/v2/internal/breach"
	"github.com/kanywst/brtc/v2/internal/calc"
	"github.com/kanywst/brtc/v2/internal/cost"
	"github.com/kanywst/brtc/v2/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// version is the binary version, overridden at release time via -ldflags
// (-X github.com/kanywst/brtc/v2/cmd.version=...), see .goreleaser.yml.
var version = "dev"

// getVersion returns the ldflags-injected version when set, otherwise the
// module version recorded in the build info. This makes `go install
// .../brtc@v1.2.3` report v1.2.3 even though it does not run goreleaser.
func getVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

// resolveOutputFormat downgrades the interactive TUI to JSON when stdout is
// not a terminal (a pipe, file, or CI log) and the user did not explicitly
// ask for a format. This keeps `brtc pw | jq` and CI runs from emitting the
// TUI's escape codes while still honouring an explicit `-o tui`.
func resolveOutputFormat(requested string, explicit, stdoutIsTTY bool) string {
	if !explicit && !stdoutIsTTY && strings.ToLower(requested) == "tui" {
		return "json"
	}
	return requested
}

var (
	hwProfile        string
	algo             string
	workFactor       int
	memoryStr        string
	externalGuesses  string
	useZxcvbn        bool
	useHIBP          bool
	budget           string
	outputFormat     string
	failUnderTime    string
	failUnderEntropy float64
	failOnBreach     bool
	allHW            bool
)

// buildMatrix runs the given combination count against every known hardware
// profile and returns the rows sorted fastest-attacker-first (then by name
// for a stable order). algo and workFactor are passed in (rather than read
// from globals) so the function stays pure and testable.
func buildMatrix(combinations *big.Int, algo string, workFactor, memoryMB int) []ui.MatrixRow {
	rows := make([]ui.MatrixRow, 0, len(cost.Profiles))
	for key, p := range cost.Profiles {
		hr := cost.CalculateHashRate(key, algo, workFactor, memoryMB)
		ttc := calc.TimeToCrack(combinations, hr)
		rows = append(rows, ui.MatrixRow{
			Profile:        key,
			Name:           p.Name,
			HashRate:       hr,
			TimeToCrackSec: ttc,
			CostUSD:        cost.TotalCost(key, ttc),
			CostPerHourUSD: p.CostPerHourUSD,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].HashRate != rows[j].HashRate {
			return rows[i].HashRate > rows[j].HashRate
		}
		return rows[i].Profile < rows[j].Profile
	})
	return rows
}

// gates is the input to the CI gatekeeper. A zero MinEntropy or ReqSecs means
// the corresponding flag was not set.
type gates struct {
	FailOnBreach bool
	BreachCount  int
	BreachErr    error // the HIBP lookup failed; count is unknown
	MinEntropy   float64
	EntropyBits  float64
	ReqSecs      float64
	TimeToCrack  float64
}

// checkGates returns the first gate failure, or nil when every enabled gate
// passes. Gates are independent and evaluated most-damning-first.
func checkGates(g gates) error {
	// A gate that could not be evaluated must not report a pass.
	if g.FailOnBreach && g.BreachErr != nil {
		return fmt.Errorf("gatekeeper failed: HIBP check could not be completed: %w", g.BreachErr)
	}
	// A breach also fails a bare --fail-under-time, as it did before
	// --fail-on-breach existed.
	if (g.FailOnBreach || g.ReqSecs > 0) && g.BreachCount > 0 {
		return fmt.Errorf("gatekeeper failed: password found in %d known breaches", g.BreachCount)
	}
	// NaN fails rather than passes, matching how the verdict treats a
	// non-finite crack time as unsafe.
	if g.MinEntropy > 0 && (math.IsNaN(g.EntropyBits) || g.EntropyBits < g.MinEntropy) {
		return fmt.Errorf("gatekeeper failed: estimated entropy (%.1f bits) is less than required (%.1f bits)",
			g.EntropyBits, g.MinEntropy)
	}
	if g.ReqSecs > 0 && g.TimeToCrack < g.ReqSecs {
		return fmt.Errorf("gatekeeper failed: estimated crack time (%s) is less than required (%s)",
			ui.FormatDuration(g.TimeToCrack), ui.FormatDuration(g.ReqSecs))
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:     "brtc [password]",
	Short:   "brtc visualizes password cracking cost",
	Long:    `brtc (Brute-force Cost) takes a password and calculates its entropy, the time to crack using specific hardware, and the estimated cloud cost.`,
	Args:    cobra.MaximumNArgs(1),
	Version: getVersion(),
	// SilenceErrors hands the single error to main(), which prints it once to
	// stderr; without it cobra prints the error too and the user sees it twice.
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Past argument parsing, any failure is a runtime error rather than a
		// misuse of the command line, so suppress the usage dump from here on.
		// (Argument-parsing errors still show usage because this has not run
		// yet for them.)
		cmd.SilenceUsage = true

		password := ""
		if len(args) > 0 {
			password = args[0]
		} else {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				b, err := io.ReadAll(os.Stdin)
				if err == nil && len(b) > 0 {
					password = strings.TrimSpace(string(b))
				}
			}
		}

		if password == "" && externalGuesses == "" {
			return fmt.Errorf("password is required. Please provide it as an argument, via stdin pipeline, or pass --guesses")
		}

		// Parse --memory (Argon2id only; ignored by other algorithms).
		memoryMB := 0
		if memoryStr != "" {
			var err error
			memoryMB, err = calc.ParseMemory(memoryStr)
			if err != nil {
				return fmt.Errorf("invalid memory format: %v", err)
			}
		}

		if useZxcvbn && externalGuesses != "" {
			return fmt.Errorf("--zxcvbn and --guesses are mutually exclusive (both set the guess count)")
		}
		if useZxcvbn && password == "" {
			return fmt.Errorf("--zxcvbn needs a password to analyze; pass it as an argument or via stdin")
		}
		// NaN and ±Inf parse fine as a float64 but make a nonsense threshold:
		// NaN in particular would fail every comparison in checkGates and turn
		// the gate into a silent no-op.
		if math.IsNaN(failUnderEntropy) || math.IsInf(failUnderEntropy, 0) || failUnderEntropy < 0 {
			return fmt.Errorf("--fail-under-entropy must be a finite, non-negative number of bits, got %g", failUnderEntropy)
		}

		// 1. Combinations: from --zxcvbn (built-in pattern estimate), --guesses
		// (external estimate), or the naive R^L entropy computed from the charset.
		var entropy calc.EntropyResult
		switch {
		case useZxcvbn:
			g, _ := calc.Zxcvbn(password)
			entropy = calc.FromGuesses(g, password)
		case externalGuesses != "":
			g, err := calc.ParseGuesses(externalGuesses)
			if err != nil {
				return fmt.Errorf("invalid guesses format: %v", err)
			}
			entropy = calc.FromGuesses(g, password)
		default:
			entropy = calc.Analyze(password)
		}

		// Resolve --hw to a known profile up front. A typo used to warn and
		// fall back to rtx-4090, which is the same silent-downgrade failure
		// the --all-hw checks below reject: a misspelled profile lands on
		// slower hardware, the crack time comes out longer, and
		// --fail-under-time reports a pass for a password that should have
		// failed the gate. Reject it instead.
		resolvedHW, known := cost.ResolveProfileName(hwProfile)
		if !known {
			return fmt.Errorf("unknown hardware profile %q (known profiles: %s)",
				hwProfile, strings.Join(cost.ProfileNames(), ", "))
		}
		hwProfile = resolvedHW

		// --all-hw is a standalone comparison view across every profile. The
		// single-profile concepts (budget, the CI gatekeeper, the SARIF report)
		// are rejected rather than silently ignored — silently dropping
		// --fail-under-time in particular would bypass a security gate.
		if allHW {
			switch {
			case cmd.Flags().Changed("hw"):
				return fmt.Errorf("--hw cannot be combined with --all-hw (it compares every profile)")
			case budget != "":
				return fmt.Errorf("--budget cannot be combined with --all-hw")
			case failUnderTime != "":
				return fmt.Errorf("--fail-under-time cannot be combined with --all-hw")
			case failUnderEntropy > 0:
				return fmt.Errorf("--fail-under-entropy cannot be combined with --all-hw")
			case failOnBreach:
				return fmt.Errorf("--fail-on-breach cannot be combined with --all-hw")
			case strings.ToLower(outputFormat) == "sarif":
				return fmt.Errorf("sarif output is not supported with --all-hw")
			case useHIBP:
				return fmt.Errorf("--hibp cannot be combined with --all-hw")
			}
			rows := buildMatrix(entropy.Combinations, algo, workFactor, memoryMB)
			if strings.ToLower(outputFormat) == "json" {
				return ui.PrintMatrixJSON(rows)
			}
			return ui.PrintMatrixTable(rows)
		}

		// 2. Hardware HashRate
		hr := cost.CalculateHashRate(hwProfile, algo, workFactor, memoryMB)

		// 3. Time to Crack
		ttc := calc.TimeToCrack(entropy.Combinations, hr)

		// 4. Cloud Cost
		costUSD := cost.TotalCost(hwProfile, ttc)

		// haveCharset is false for external estimates (--guesses, --zxcvbn),
		// which report a guess count with no character-class assumption. The
		// budget and recommendation math both need a charset, so they are
		// skipped in that case rather than emitting a misleading "0 chars".
		haveCharset := entropy.CharSpace > 1

		// 5. Budget logic (optional).
		budgetVal, err := cost.ParseBudget(budget)
		if err != nil {
			return fmt.Errorf("invalid budget format: %v", err)
		}

		var budgetMaxChars *int
		var budgetUnlimited bool
		if budgetVal > 0 && haveCharset {
			maxChars := cost.MaxLengthForBudget(budgetVal, hwProfile, algo, workFactor, memoryMB, entropy.CharSpace)
			// Owned hardware has no rental cost, so no budget bounds the
			// attacker. Surface that as a flag rather than leaking the 999
			// sentinel into the output as a literal length.
			if maxChars == cost.UnlimitedBudgetChars {
				budgetUnlimited = true
			} else {
				budgetMaxChars = &maxChars
			}
		}

		// 6. Gatekeeper threshold (optional). Parsed up front so we can both
		// recommend a safe length and enforce the gate after output. Like the
		// budget logic, the recommendation needs a charset and is skipped for
		// external --guesses estimates.
		var reqSecs float64
		if failUnderTime != "" {
			reqSecs, err = calc.ParseDuration(failUnderTime)
			if err != nil {
				return fmt.Errorf("invalid fail-under-time format: %v", err)
			}
		}
		var recommendedChars int
		if reqSecs > 0 && haveCharset {
			recommendedChars = cost.MinLengthForTime(reqSecs, hwProfile, algo, workFactor, memoryMB, entropy.CharSpace)
		}

		// Optional Have I Been Pwned check. Network-dependent and opt-in;
		// --fail-on-breach turns the lookup on itself rather than silently
		// passing when --hibp was forgotten. A lookup failure is not fatal
		// here — the offline analysis still prints and checkGates decides
		// whether it sinks the run.
		var breachChecked bool
		var breachCount int
		var breachErr error
		if useHIBP || failOnBreach {
			if password == "" {
				flag := "--hibp"
				if failOnBreach {
					flag = "--fail-on-breach"
				}
				return fmt.Errorf("%s needs a password to check; pass it as an argument or via stdin", flag)
			}
			res, herr := breach.Check(cmd.Context(), password, nil)
			if herr != nil {
				breachErr = herr
				// Under --fail-on-breach the gate error carries this, so
				// warning here too would print the same thing twice.
				if !failOnBreach {
					cmd.PrintErrf("warning: HIBP check failed: %v\n", herr)
				}
			} else {
				breachChecked = true
				breachCount = res.Count
			}
		}

		// Compile output data
		outData := ui.OutputData{
			PasswordLength:   entropy.Length,
			CharSpace:        entropy.CharSpace,
			Entropy:          entropy.Entropy,
			Combinations:     entropy.Combinations,
			Algorithm:        algo,
			WorkFactor:       workFactor,
			MemoryMB:         memoryMB,
			Hardware:         hwProfile,
			HashRate:         hr,
			TimeToCrackSec:   ttc,
			CostUSD:          costUSD,
			BudgetUSD:        budgetVal,
			BudgetMaxChars:   budgetMaxChars,
			BudgetUnlimited:  budgetUnlimited,
			RecommendedChars: recommendedChars,
			BreachChecked:    breachChecked,
			BreachCount:      breachCount,
		}

		// Present output. IsCygwinTerminal covers MSYS/Git Bash on Windows,
		// where IsTerminal alone reports false for a real interactive terminal.
		fd := os.Stdout.Fd()
		stdoutIsTTY := isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
		format := resolveOutputFormat(outputFormat, cmd.Flags().Changed("output"), stdoutIsTTY)
		var errOut error
		switch strings.ToLower(format) {
		case "json":
			errOut = ui.PrintJSON(outData)
		case "table":
			errOut = ui.PrintTable(outData)
		case "sarif":
			errOut = ui.PrintSARIF(outData)
		case "tui":
			errOut = ui.RunTUI(outData)
		default:
			// A misspelled format previously fell through to the TUI, which is
			// confusing in scripts; fail loudly instead.
			return fmt.Errorf("unknown output format %q (want tui, table, json, or sarif)", outputFormat)
		}
		if errOut != nil {
			return errOut
		}

		return checkGates(gates{
			FailOnBreach: failOnBreach,
			BreachCount:  breachCount,
			BreachErr:    breachErr,
			MinEntropy:   failUnderEntropy,
			EntropyBits:  entropy.Entropy,
			ReqSecs:      reqSecs,
			TimeToCrack:  ttc,
		})
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Built from hashrates.yaml rather than hand-listed, so adding a profile
	// cannot leave the help advertising a stale set.
	rootCmd.Flags().StringVar(&hwProfile, "hw", "rtx-4090",
		"Attacker's hardware profile ("+strings.Join(cost.ProfileNames(), ", ")+")")
	rootCmd.Flags().StringVar(&algo, "algo", "bcrypt", "Server-side hash algorithm (md5, sha1, sha256, ntlm, bcrypt, argon2id)")
	rootCmd.Flags().IntVar(&workFactor, "cost", 10, "Work factor (bcrypt) or time iterations (argon2id)")
	rootCmd.Flags().StringVar(&memoryStr, "memory", "", "Argon2id memory parameter (e.g. 64m, 128m, 1g). Defaults to the profile baseline (64MB)")
	rootCmd.Flags().StringVar(&externalGuesses, "guesses", "", "Override entropy with an external guess count from zxcvbn or similar (e.g. 1e10, 12345)")
	rootCmd.Flags().BoolVar(&useZxcvbn, "zxcvbn", false, "Estimate strength with the built-in zxcvbn pattern analyzer instead of naive R^L entropy (catches dictionary words, keyboard walks, l33t)")
	rootCmd.Flags().BoolVar(&useHIBP, "hibp", false, "Check the password against Have I Been Pwned via k-anonymity (only the SHA-1 prefix is sent; requires network)")
	rootCmd.Flags().StringVar(&budget, "budget", "", "Attacker's budget in USD (e.g., 1000usd)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "tui", "Output format (tui, table, json, sarif)")
	rootCmd.Flags().StringVar(&failUnderTime, "fail-under-time", "", "Gatekeeper threshold for CI/CD (e.g., 1y, 30d)")
	rootCmd.Flags().Float64Var(&failUnderEntropy, "fail-under-entropy", 0, "Gatekeeper threshold in bits of entropy (e.g. 60, 80)")
	rootCmd.Flags().BoolVar(&failOnBreach, "fail-on-breach", false, "Gatekeeper: fail if the password appears in Have I Been Pwned (implies --hibp; a failed lookup also fails the gate)")
	rootCmd.Flags().BoolVar(&allHW, "all-hw", false, "Compare the password across every hardware profile (not combinable with --budget, the --fail-* gates, or -o sarif)")
}
