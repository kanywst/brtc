package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kanywst/brtc/internal/calc"
	"github.com/kanywst/brtc/internal/cost"
	"github.com/kanywst/brtc/internal/ui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

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
	hwProfile       string
	algo            string
	workFactor      int
	memoryStr       string
	externalGuesses string
	budget          string
	outputFormat    string
	failUnderTime   string
)

var rootCmd = &cobra.Command{
	Use:   "brtc [password]",
	Short: "brtc visualizes password cracking cost",
	Long:  `brtc (Brute-force Cost) takes a password and calculates its entropy, the time to crack using specific hardware, and the estimated cloud cost.`,
	Args:  cobra.MaximumNArgs(1),
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

		// 1. Combinations: from --guesses (e.g. zxcvbn output) or computed entropy.
		var entropy calc.EntropyResult
		if externalGuesses != "" {
			g, err := calc.ParseGuesses(externalGuesses)
			if err != nil {
				return fmt.Errorf("invalid guesses format: %v", err)
			}
			entropy = calc.FromGuesses(g, password)
		} else {
			entropy = calc.Analyze(password)
		}

		// Resolve --hw to a known profile up front. An unknown value silently
		// falls back to rtx-4090, so warn and report the resolved name rather
		// than echoing the bogus input back in the results.
		resolvedHW, known := cost.ResolveProfileName(hwProfile)
		if !known {
			cmd.PrintErrf("warning: unknown hardware profile %q, falling back to %s\n", hwProfile, resolvedHW)
		}
		hwProfile = resolvedHW

		// 2. Hardware HashRate
		hr := cost.CalculateHashRate(hwProfile, algo, workFactor, memoryMB)

		// 3. Time to Crack
		ttc := calc.TimeToCrack(entropy.Combinations, hr)

		// 4. Cloud Cost
		costUSD := cost.TotalCost(hwProfile, ttc)

		// 5. Budget logic (optional). Skipped when --guesses is in use because
		// MaxLengthForBudget needs a charset assumption that external estimates
		// do not provide.
		budgetVal, err := cost.ParseBudget(budget)
		if err != nil {
			return fmt.Errorf("invalid budget format: %v", err)
		}

		var budgetMaxChars int
		var budgetUnlimited bool
		if budgetVal > 0 && externalGuesses == "" {
			budgetMaxChars = cost.MaxLengthForBudget(budgetVal, hwProfile, algo, workFactor, memoryMB, entropy.CharSpace)
			// Owned hardware has no rental cost, so no budget bounds the
			// attacker. Surface that as a flag rather than leaking the 999
			// sentinel into the output as a literal length.
			if budgetMaxChars == cost.UnlimitedBudgetChars {
				budgetUnlimited = true
				budgetMaxChars = 0
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
		if reqSecs > 0 && externalGuesses == "" {
			recommendedChars = cost.MinLengthForTime(reqSecs, hwProfile, algo, workFactor, memoryMB, entropy.CharSpace)
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

		// Gatekeeper (fail-under-time) logic. reqSecs was parsed and validated
		// above; a zero value means the flag was not set.
		if reqSecs > 0 && ttc < reqSecs {
			return fmt.Errorf("gatekeeper failed: estimated crack time (%s) is less than required (%s)",
				ui.FormatDuration(ttc), ui.FormatDuration(reqSecs))
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&hwProfile, "hw", "rtx-4090", "Attacker's hardware profile (rtx-4090, rtx-3060, gtx-1080ti, mac-m3-max, mac-m3, cpu-standard, aws-p5.48xlarge, raspberry-pi-4)")
	rootCmd.Flags().StringVar(&algo, "algo", "bcrypt", "Server-side hash algorithm (md5, sha256, bcrypt, argon2id)")
	rootCmd.Flags().IntVar(&workFactor, "cost", 10, "Work factor (bcrypt) or time iterations (argon2id)")
	rootCmd.Flags().StringVar(&memoryStr, "memory", "", "Argon2id memory parameter (e.g. 64m, 128m, 1g). Defaults to the profile baseline (64MB)")
	rootCmd.Flags().StringVar(&externalGuesses, "guesses", "", "Override entropy with an external guess count from zxcvbn or similar (e.g. 1e10, 12345)")
	rootCmd.Flags().StringVar(&budget, "budget", "", "Attacker's budget in USD (e.g., 1000usd)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "tui", "Output format (tui, table, json, sarif)")
	rootCmd.Flags().StringVar(&failUnderTime, "fail-under-time", "", "Gatekeeper threshold for CI/CD (e.g., 1y, 30d)")
}
