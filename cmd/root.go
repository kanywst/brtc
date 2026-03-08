package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/kanywst/brtc/internal/calc"
	"github.com/kanywst/brtc/internal/cost"
	"github.com/kanywst/brtc/internal/ui"
	"github.com/spf13/cobra"
)

var (
	hwProfile     string
	algo          string
	workFactor    int
	budget        string
	outputFormat  string
	failUnderTime string
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

		if password == "" {
			return fmt.Errorf("password is required. Please provide it as an argument or via stdin pipeline")
		}

		// 1. Analyze Entropy
		entropy := calc.Analyze(password)

		// 2. Hardware HashRate
		hr := cost.CalculateHashRate(hwProfile, algo, workFactor)

		// 3. Time to Crack
		ttc := calc.TimeToCrack(entropy.Combinations, hr)

		// 4. Cloud Cost
		costUSD := cost.TotalCost(hwProfile, ttc)

		// 5. Budget logic (optional)
		budgetVal, err := cost.ParseBudget(budget)
		if err != nil {
			return fmt.Errorf("invalid budget format: %v", err)
		}

		var budgetMaxChars int
		if budgetVal > 0 {
			budgetMaxChars = cost.MaxLengthForBudget(budgetVal, hwProfile, algo, workFactor, entropy.CharSpace)
		}

		// Compile output data
		outData := ui.OutputData{
			PasswordLength: entropy.Length,
			CharSpace:      entropy.CharSpace,
			Entropy:        entropy.Entropy,
			Combinations:   entropy.Combinations,
			Algorithm:      algo,
			WorkFactor:     workFactor,
			Hardware:       hwProfile,
			HashRate:       hr,
			TimeToCrackSec: ttc,
			CostUSD:        costUSD,
			BudgetUSD:      budgetVal,
			BudgetMaxChars: budgetMaxChars,
		}

		// Present output
		var errOut error
		switch strings.ToLower(outputFormat) {
		case "json":
			errOut = ui.PrintJSON(outData)
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

		// Gatekeeper (fail-under-time) logic
		if failUnderTime != "" {
			reqSecs, err := parseDurationToSeconds(failUnderTime)
			if err != nil {
				return fmt.Errorf("invalid fail-under-time format: %v", err)
			}
			if ttc < reqSecs {
				return fmt.Errorf("gatekeeper failed: estimated crack time (%s) is less than required (%s)",
					ui.FormatDuration(ttc), ui.FormatDuration(reqSecs))
			}
		}

		return nil
	},
}

func parseDurationToSeconds(d string) (float64, error) {
	d = strings.ToLower(strings.TrimSpace(d))
	re := regexp.MustCompile(`^(\d+)([smhdwy])$`)
	matches := re.FindStringSubmatch(d)
	if len(matches) != 3 {
		return 0, fmt.Errorf("expected format like 30d, 1y, 12h (units: s,m,h,d,w,y)")
	}
	val, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}
	switch matches[2] {
	case "s":
		return val, nil
	case "m":
		return val * 60, nil
	case "h":
		return val * 3600, nil
	case "d":
		return val * 86400, nil
	case "w":
		return val * 604800, nil
	case "y":
		return val * 31536000, nil
	}
	return 0, fmt.Errorf("unknown unit %s", matches[2])
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&hwProfile, "hw", "rtx-4090", "Attacker's hardware profile (rtx-4090, rtx-3060, gtx-1080ti, mac-m3-max, mac-m3, cpu-standard, aws-p5.48xlarge, raspberry-pi-4)")
	rootCmd.Flags().StringVar(&algo, "algo", "bcrypt", "Server-side hash algorithm (md5, sha256, bcrypt, argon2id)")
	rootCmd.Flags().IntVar(&workFactor, "cost", 10, "Work factor for algorithms like bcrypt")
	rootCmd.Flags().StringVar(&budget, "budget", "", "Attacker's budget in USD (e.g., 1000usd)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "tui", "Output format (tui, json, sarif)")
	rootCmd.Flags().StringVar(&failUnderTime, "fail-under-time", "", "Gatekeeper threshold for CI/CD (e.g., 1y, 30d)")
}
