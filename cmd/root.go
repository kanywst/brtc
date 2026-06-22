package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kanywst/brtc/internal/calc"
	"github.com/kanywst/brtc/internal/cost"
	"github.com/kanywst/brtc/internal/ui"
	"github.com/spf13/cobra"
)

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
	RunE: func(cmd *cobra.Command, args []string) error {
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
		if budgetVal > 0 && externalGuesses == "" {
			budgetMaxChars = cost.MaxLengthForBudget(budgetVal, hwProfile, algo, workFactor, memoryMB, entropy.CharSpace)
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
			RecommendedChars: recommendedChars,
		}

		// Present output
		var errOut error
		switch strings.ToLower(outputFormat) {
		case "json":
			errOut = ui.PrintJSON(outData)
		case "table":
			errOut = ui.PrintTable(outData)
		case "sarif":
			errOut = ui.PrintSARIF(outData)
		case "tui":
			fallthrough
		default:
			errOut = ui.RunTUI(outData)
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
