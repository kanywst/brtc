# Security Policy

## Reporting a vulnerability

Report vulnerabilities through [GitHub private vulnerability reporting](https://github.com/kanywst/brtc/security/advisories/new). Please do not open a public issue for anything exploitable.

Expect an acknowledgement within 7 days. If a report is confirmed, the fix ships in the next patch release and the advisory is published once that release is out.

## Supported versions

Only the latest release receives fixes. There are no maintained release branches — upgrade to the newest tag before reporting.

## Handling passwords safely

`brtc` analyzes a password that you supply, so how you supply it matters more than anything inside the tool.

Passing a password as an argument puts it in your shell history and, on most systems, in the process list where any local user can read it while the command runs:

```bash
brtc 'hunter2'          # visible in `ps` and in ~/.zsh_history
```

Prefer stdin, which avoids both:

```bash
printf '%s' "$PASSWORD" | brtc
```

In CI, keep the password in a secret and pipe it in the same way rather than interpolating it into the command line, where it may be echoed into build logs.

## What leaves your machine

`brtc` performs no network I/O unless you pass `--hibp` or `--fail-on-breach`.

Those flags query [Have I Been Pwned](https://haveibeenpwned.com/API/v3#SearchingPwnedPasswordsByRange) using its k-anonymity range API: the password is hashed with SHA-1 locally and **only the first five hex characters of that hash** are sent. The full password, and the full hash, never leave the machine. The response is a bucket of suffixes that is matched locally.

## Scope

`brtc` estimates the cost of an **offline** attack against a stolen hash. Its numbers are a planning aid built on published hashcat benchmarks, not a guarantee — an attacker with better hardware, a targeted wordlist, or knowledge of your password policy will do better than the naive `R^L` model.

A high entropy score is not a safety certificate. Treat the following as in scope for a report:

- A path where a supplied password is written to disk, a log, or the network.
- A crash or hang reachable from an ordinary password or flag combination.
- A gate (`--fail-under-time`, `--fail-under-entropy`, `--fail-on-breach`) that reports success when it should fail.

Inaccurate hashrate baselines are a data-quality issue rather than a vulnerability — open a regular issue with a source, and see `internal/cost/hashrates.yaml` for the current citations.
