# GitHub Action

brtc ships a composite GitHub Action at the root of this repository. It builds brtc on the runner, passes the secret over stdin, fails the job when a gate trips, and exposes the JSON fields (or a SARIF file) as step outputs.

> **brtc is a cost calculator, not a strength meter.** For accurate strength evaluation, run [zxcvbn](https://github.com/dropbox/zxcvbn) first and feed its `guesses` value via the `guesses:` input.

## Quick start

Gate a deployment on a secret being expensive enough to crack:

```yaml
- name: Gate on password strength
  uses: kanywst/brtc@v2
  with:
    password: ${{ secrets.SERVICE_PASSWORD }}
    algorithm: bcrypt
    cost: "12"
    hardware: aws-p5.48xlarge
    fail-under-time: 1y
```

If `aws-p5.48xlarge` could crack the secret in less than a year, the step exits non-zero and the workflow fails.

## Migrating from `kanywst/brtc-action`

The action used to live in [kanywst/brtc-action](https://github.com/kanywst/brtc-action), which is retired. Inputs and outputs are unchanged, so only the `uses:` line moves:

```yaml
# before
- uses: kanywst/brtc-action@v2
# after
- uses: kanywst/brtc@v2
```

One behaviour differs. `brtc-version` used to default to a pinned release that each action release bumped by hand. It now defaults to empty, which builds the brtc source at the same ref as the action, so the action and the CLI can no longer drift apart. Set `brtc-version` explicitly to keep the old install-a-tag behaviour.

## Pair with zxcvbn (recommended)

Use a real strength estimator for entropy, brtc for the cost translation:

```yaml
- name: Estimate guesses with zxcvbn
  id: zxcvbn
  env:
    # Pass the secret via env, not string interpolation into the script.
    PASSWORD: ${{ secrets.SERVICE_PASSWORD }}
  run: |
    npx --yes zxcvbn-cli "$PASSWORD" --json \
      | jq -r '.guesses' \
      | xargs -I{} echo "guesses={}" >> "$GITHUB_OUTPUT"

- name: Convert to USD cost
  uses: kanywst/brtc@v2
  with:
    guesses: ${{ steps.zxcvbn.outputs.guesses }}
    algorithm: bcrypt
    cost: "12"
    fail-under-time: 1y
```

## Gates

Three independent gates decide whether the step fails. Use one, or combine them — the step fails on the first one that trips.

```yaml
- uses: kanywst/brtc@v2
  with:
    password: ${{ secrets.SERVICE_PASSWORD }}
    algorithm: bcrypt
    cost: "12"
    fail-under-time: 1y        # too cheap to crack
    fail-under-entropy: 60     # too few bits
    fail-on-breach: true       # already in a public corpus
```

`fail-on-breach` looks the secret up in [Have I Been Pwned](https://haveibeenpwned.com/) using the k-anonymity range API: only the first 5 hex characters of the secret's SHA-1 hash leave the runner, and the suffix is matched locally. It needs network access, and a lookup that **cannot be completed** fails the step too — a gate that was never evaluated must not report a pass. Set it only when your runners can reach `api.pwnedpasswords.com`.

`fail-under-entropy` reads brtc's entropy estimate, which is a naive `R^L` upper bound unless you feed a real guess count through `guesses`. Pair the two for a threshold that reflects actual guessability.

Both gates need brtc **v1.4.0 or later**. With an older `brtc-version` the action stops with a clear error instead of passing the unknown flag through.

## Inputs

| Name              | Default     | Description                                                                              |
| ----------------- | ----------- | ---------------------------------------------------------------------------------------- |
| `password`        | _none_      | Secret to evaluate. Either `password` or `guesses` must be set.                          |
| `guesses`         | _none_      | External guess count (e.g. `zxcvbn` output, `1e10`). Skips brtc's built-in entropy.      |
| `algorithm`       | `bcrypt`    | `md5`, `sha1`, `sha256`, `ntlm`, `bcrypt`, `argon2id`, or `pbkdf2-sha256` (brtc v2.4.0+). From brtc v2.1.0 an unrecognized name fails the job rather than being modeled as `bcrypt`. |
| `cost`            | _none_      | Work factor — bcrypt `4`–`31`, argon2id time iterations `1`+, pbkdf2-sha256 iteration count `1`+. Unset, the action sends `10` for bcrypt and argon2id. **Required for pbkdf2-sha256**: brtc rejects it unset, and the action does not fill it in. From brtc v2.2.0 setting it for a single-pass algorithm (`md5`, `sha1`, `sha256`, `ntlm`) fails the step, so it is withheld for those unless you set it. |
| `memory`          | _none_      | Argon2id memory (e.g. `64m`, `128m`, `1g`). From brtc v2.2.0 setting it for any other algorithm fails the step rather than being discarded. |
| `hardware`        | `rtx-4090`  | Attacker hardware profile — `rtx-5090`, `rtx-4090`, `rx-7900xtx`, `rtx-3060`, `gtx-1080ti`, `mac-m3-max`, `mac-m3`, `cpu-standard`, `aws-p5.48xlarge`, `raspberry-pi-4`. An unrecognized name fails the step. |
| `all-hw`          | `false`     | Compare across every hardware profile. Ignores the `fail-*` gates and `budget`, no `sarif`. |
| `fail-under-time` | _none_      | Fail the step if estimated crack time is shorter (e.g. `1y`, `30d`, `12h`).              |
| `fail-under-entropy` | _none_   | Fail the step if estimated entropy is below this many bits (e.g. `60`, `80`).            |
| `fail-on-breach`  | `false`     | Fail the step if the secret is in Have I Been Pwned. Needs runner network access.        |
| `budget`          | _none_      | Attacker budget in USD (e.g. `1000usd`).                                                 |
| `output`          | `json`      | `json` or `sarif`. `json` populates the outputs; `sarif` populates `sarif-file`.        |
| `brtc-version`    | _empty_     | Leave empty to build the brtc source at the ref you reference the action at (`@v2.4.1` runs brtc v2.4.1, `@v2` the latest v2 release). Set a release tag (e.g. `v2.4.0`, `latest`, `main`) to `go install` that brtc instead. Must be `v2.0.2` or later: brtc's module path became `github.com/kanywst/brtc/v2` in v2.0.2, so v1 tags and the earlier v2.0.0/v2.0.1 will not install. |
| `go-version`      | _empty_     | Go toolchain used to build brtc. Empty uses the toolchain brtc's `go.mod` pins for that ref, the Go its released binaries are built with. |

## Outputs

| Name                    | Description                                            |
| ----------------------- | ------------------------------------------------------ |
| `cost-usd`              | Estimated USD cost to crack.                           |
| `time-to-crack-seconds` | Estimated seconds to crack.                            |
| `entropy-bits`          | Estimated entropy in bits.                             |
| `breach-count`          | Times the secret appears in HIBP. `0` unless `fail-on-breach` is set. |
| `raw-json`              | Full brtc JSON output. Only set when `output: json`.   |
| `sarif-file`            | Path to the written SARIF file. Only set when `output: sarif`. |

## SARIF output for Code Scanning

`output: sarif` writes a SARIF file into the workspace and exposes its path via the `sarif-file` output. `upload-sarif` needs a checked-out repo and the `security-events: write` permission:

```yaml
permissions:
  security-events: write

steps:
  - uses: actions/checkout@v7
  - name: brtc → SARIF
    id: brtc
    uses: kanywst/brtc@v2
    with:
      password: ${{ secrets.SERVICE_PASSWORD }}
      output: sarif
  - name: Upload SARIF
    uses: github/codeql-action/upload-sarif@v3
    with:
      sarif_file: ${{ steps.brtc.outputs.sarif-file }}
```

## Notes

- The `password` is piped to brtc over stdin so it never appears in the runner's process list. brtc trims surrounding whitespace from a stdin password, so leading/trailing spaces are not counted.
- `fail-on-breach` is the only input that sends anything off the runner, and it sends 5 hex characters of a SHA-1 hash, never the secret. Every other gate is computed locally.
- The action compiles brtc on the runner, so the Go toolchain it installs is the standard library your run links. By default that is the toolchain in brtc's `go.mod` for the ref you use.

## Versioning

The action is versioned with brtc itself: every brtc release from v2.4.1 on is also an action release. Earlier brtc tags do not contain the action.

- `@v2`: floating major tag, moved to each new `v2.x.y` release by the release workflow. This is what the examples above use.
- `@vX.Y.Z`: pin to a specific brtc release, CLI and action together.
- `@main`: bleeding edge, may break.

The runner fetches the action as a source tarball with no git history, so only a `@vX.Y.Z` ref tells the action which version it is building. Under `@v2`, `@main` or a commit SHA, `brtc --version` reports `dev`, though the code is exactly the release or commit the ref points at.

**A new brtc release can change your numbers** when you track `@v2`: hardware baselines, new validation and new defaults all flow through. Pin `@vX.Y.Z` if a run has to keep producing the same numbers over time.

brtc v1 is not available from this repository. Workflows that need it can stay on `kanywst/brtc-action@v1`, which keeps working after that repository is archived.
