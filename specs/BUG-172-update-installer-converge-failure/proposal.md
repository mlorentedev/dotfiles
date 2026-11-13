---
id: "BUG-172-update-installer-converge-failure"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3395"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-172: a converge failure inside install.sh reads as an install failure

## Why

<!-- from issue #3395: update: a converge failure inside install.sh is reported as an install failure -->

When the running `dotf` is behind the checkout's `DOTF_VERSION` pin, `dotf update` runs the checkout's installer, which installs the pinned binary and then converges with it. `update.Run` reports any non-zero exit of the installer as `install-failed`. On 2026-11-12 the binary installed fine and a Homebrew cask download failed inside the converge, yet the report said "installing it failed". That sends the operator to the wrong step, and doctor's Self-deploy line repeats it.

## What

- When the installer exits non-zero, `dotf update` reads the full converge report (`<state>/converge/last.json`). If that report finished no earlier than the installer started and records a failed run, the installer's own converge produced it. The installer converges only after the new binary is in place, so this proves the pin installed.
- In that case the status is `converge-failed` (an existing tag), and the message says the pin is installed and names the failed steps. The error names them too.
- Otherwise the status stays `install-failed`: no fresh report, a passing one, or one that cannot be read.

## Out of scope

- A new exit code or marker in `install.sh`/`install.ps1` (option A in the issue). The report the converge already writes answers the question with no installer contract change.
- A new status tag. `converge-failed` already means "the deploy converged and a step failed", and doctor already treats it as a failure.
- Parsing the console output of the converge.

## Risks / open questions

- A converge from another process could finish between the installer's start and its exit and leave a failed report. The window is one installer run, and the only other converge on the machine is this same scheduled update, so this is accepted. Resolved.
- `finished_at` has one-second resolution. The comparison truncates the installer's start to the second, so a converge that finishes in the start second still counts. Resolved, and tested.

## Acceptance criteria

- [ ] AC1: An installer that exits non-zero after its converge failed a step gives `converge-failed`, and both the message and the error name the pin as installed and the failed step.
- [ ] AC2: An installer that exits non-zero with no converge report newer than its start (or with a passing one, or an unreadable one) still gives `install-failed`.
- [ ] AC3: A failed report with no failed step still counts as a converge failure, named as the run with its error.

## References

- #3395, BUG-144 (#2506, the pin install)
- `cli/internal/update/update.go`, `cli/internal/cmd/update.go`, `cli/internal/converge/report.go`
