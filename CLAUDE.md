# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`brtc` (Brute-force Cost) is a Go CLI that takes a password and prints the entropy, the wall-clock time to brute-force it on a chosen hardware profile, and the USD cost to rent that hardware long enough to crack it. The tool models **offline** attacks against a stolen hash, not online login guessing.

## Common commands

```bash
make build            # go build -o brtc main.go
make test             # go test -v ./...
make lint             # golangci-lint run ./...   (requires v2; CI pins v2.10.1)
make format           # go fmt ./... && go mod tidy
make vuln             # govulncheck ./...

go test -v -run TestAnalyze ./internal/calc/   # run a single test
go test -race -cover ./...                     # what CI runs
go run . "P@ssw0rd!" --algo bcrypt --cost 12   # run without building
```

Go toolchain: `go.mod` declares **1.25.0** and `.golangci.yml` sets `go: "1.25"` for the linter — keep those two in sync if bumping versions. `.github/workflows/ci.yml` deliberately runs ahead on **1.26**: `govulncheck@latest` refuses to install on an older toolchain, and `go install` builds it against whatever the runner has. CI being newer than the language version the module targets is fine and is not drift to "fix"; `release.yml` uses `go-version-file: go.mod`, so released binaries are still built at the declared version.

**The module path carries a major-version suffix: `github.com/kanywst/brtc/v2`.** Any future major bump has to move it again (`/v3`, …) in `go.mod`, every internal import, the `-X .../cmd.version` ldflags in `.goreleaser.yml`, the `go install` line and the pkg.go.dev badge in `README.md`, the repo homepage, and the `go install` line in `kanywst/brtc-action`. Tagging `vN.0.0` without that makes `go install github.com/kanywst/brtc/vN@...` fail outright with *"module path must match major version"* — v2.0.0 and v2.0.1 shipped that way and had to be superseded.

## Architecture

Single-binary Cobra CLI. Data flows top-to-bottom through five steps in `cmd/root.go`'s `RunE`:

```text
input password
   │
   ▼
calc.Analyze(pw)              → EntropyResult{CharSpace, Length, Entropy, Combinations *big.Int}
   │
   ▼
cost.CalculateHashRate(hw, algo, workFactor)   → hashes/sec (float64)
   │
   ▼
calc.TimeToCrack(combinations, hashRate)       → seconds (float64), uses big.Float
   │
   ▼
cost.TotalCost(hw, ttc)                        → USD
   │
   ▼
ui.RunTUI / PrintJSON / PrintSARIF             → output
   │
   ▼
optional gatekeeper: cmd.checkGates — --fail-on-breach, --fail-under-entropy, --fail-under-time
```

Package responsibilities:

- `cmd/` — Cobra wiring, flag parsing, stdin fallback, `--fail-under-time` duration parser (`s|m|h|d|w|y`).
- `internal/calc/` — pure math. Entropy uses `len([]rune(pw))` (rune-aware) and `R^L` via `math/big.Int.Exp` so very long passwords don't overflow. `TimeToCrack` uses `big.Float` for the same reason and returns `+Inf` if `hashRate <= 0`.
- `internal/cost/` — `Profiles` map keyed by `--hw` string holds per-profile baseline hash rates for each algorithm plus `CostPerHourUSD`. `CalculateHashRate` applies algorithm scaling on top of the baseline: bcrypt is `2^(workFactor-5)` slower than the cost-5 baseline; argon2id divides by `workFactor` linearly. `MaxLengthForBudget` inverts the time formula to find the largest `L` an attacker could afford.
- `internal/ui/` — `tui.go` is a Bubble Tea / Lipgloss view. `output.go` has `PrintJSON`, `PrintSARIF`, and the shared `FormatDuration` helper used by both the TUI and the gatekeeper error message.

### Gotchas

- **`--hw` and `--algo` both reject unknown strings, for the same reason.** An unrecognized value used to fall back silently — `--hw` to `rtx-4090`, `--algo` to `bcrypt` — and because both fallbacks are *slower* than what the user typed, the crack time came out longer and `--fail-under-time` reported a pass for a password that should have failed the gate. `--algo` was the worse of the two: the JSON and SARIF reports echo the input string back as the algorithm modeled, so a CI log never showed the substitution. Both are now resolved and rejected in `cmd/root.go` before the `--all-hw` branch, and both resolve to a canonical lowercase key, so `--algo MD5` reports `md5`. `lookupProfile` and the fallback inside `CalculateHashRate` stay for direct callers of `internal/cost`.
- **`--hw` rejects unknown strings.** `internal/cost/hashrates.yaml` currently defines `rtx-5090`, `rtx-4090`, `rx-7900xtx`, `rtx-3060`, `gtx-1080ti`, `mac-m3`, `mac-m3-max`, `cpu-standard`, `raspberry-pi-4`, and `aws-p5.48xlarge`. `cmd/root.go` errors out on anything else rather than falling back, because a typo used to land on slower hardware and turn `--fail-under-time` into a false pass. The `--hw` help string and the error message are both generated from `cost.ProfileNames()`, so only the README table is hand-written — `TestProfilesContainAdvertisedNames` guards the YAML against it in both directions. `lookupProfile` still falls back to `rtx-4090` for direct callers of `internal/cost`.
- **Hardware baselines are 2026 hashcat numbers** (`last_reviewed: 2026-07-19`). Note the modeling insight baked into the YAML comments: password cracking is integer/bitwise work, so consumer GPUs (5090/4090) beat datacenter AI GPUs (H100/H200) on both raw speed and hashes-per-dollar. `aws-p5.48xlarge` is intentionally cost-inefficient, not the scary top attacker it looks like.
- **`MaxLengthForBudget` returns `999` as a sentinel** when `CostPerHourUSD == 0` (owned hardware), meaning "effectively unlimited." Callers that present this number should treat it as an infinity flag, not a literal length.
- **SARIF output is intentionally a stretch.** `PrintSARIF` emits a fixed `BRTC-001` rule with severity bucketed from entropy bits — it's a CI-pipeline convenience, not a real static-analysis report.
- **bcrypt baseline is at cost = 5**, not the cobra default of `cost = 10`. The factor in `CalculateHashRate` is `2^(workFactor - 5)`. If you change the baseline you must update both the comment and the math. Note that `cost = 4`, bcrypt's minimum and the one valid cost below the baseline, extrapolates to twice the baseline rate rather than clamping to it.
- **`--cost` and `--memory` are rejected where the algorithm does not consume them**, and `--cost` is range-checked (`cost.TuningFor` holds the table: bcrypt 4–31, argon2id 1+ with memory, the single-pass algorithms neither). Same principle as the `--hw`/`--algo` rejections, but for a weaker reason: the clamps all landed on the safe side of a gate, so nothing false-passed. What was wrong is that the report named parameters the calculation had discarded — `--algo md5 --cost 12` printed `work_factor: 12` over the raw md5 baseline. `work_factor` is now `omitempty` and set to 0 for single-pass algorithms, so JSON consumers see the field disappear rather than read a default that was never applied. The floors inside `CalculateHashRate` remain as backstops for direct callers of `internal/cost`.

## Conventions for changes here

- Keep linter rules minimal — `.golangci.yml` enables only `govet`, `staticcheck`, `ineffassign`, `misspell`, `errcheck`. Don't add lint suppressions to fix the linter; fix the code.
- New hardware profiles go in `internal/cost/hashrates.yaml` (the `Profiles` map is unmarshaled from it at init, not written by hand), **and** must be added to the table in `README.md` plus the `advertised` list in `TestProfilesContainAdvertisedNames`. The `--hw` help string and the unknown-profile error are generated from the YAML and need no edit. Every profile must define a positive rate for **every** algorithm — `init` panics otherwise, because `AlgoNames` reads the advertised `--algo` set off the fallback profile, and a profile missing a key would route a *valid* `--algo` down the bcrypt fallback and reintroduce the false pass through valid input.
- New algorithms go in every profile's `hashrates` block, the `requiredAlgos` list in `internal/cost/hardware_test.go`, and the `--algo` row of the README table. The `--algo` help string and the unknown-algorithm error are generated from the YAML.
- New output formats branch in the `switch strings.ToLower(outputFormat)` in `cmd/root.go` and live in `internal/ui/`. Add a corresponding `Print<Format>` and reuse `FormatDuration`.
