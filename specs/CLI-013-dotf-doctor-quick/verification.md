---
tags: [spec, verification]
created: "2026-06-14"
---

# Verification - CLI-013-dotf-doctor-quick

## Evidence

| # | Acceptance criterion | Proof |
|---|---|---|
| 1 | `--quick` runs only the contract sections; heavy sections do NOT run | `TestRun_QuickSkipsHeavySections` (quick output omits "Core tools in PATH", "Harness + skill drift", "Antigravity CLI health", "Knowledge vault"); smoke output shows only the 4 contract sections |
| 2 | `--quick` keeps the exit contract (0 pass / 1 any FAIL) | `TestRun_QuickSkipsHeavySections` asserts full mode exit 1 (missing tools), quick mode exit 0 (contract-only) on the same env |
| 3 | `--quick` does not fork `compile-harness.sh` (well under ~2.8s) | Smoke timing: `dotf doctor --quick` = **0.00s** vs ~2.8s for the full sweep; the harness-drift section is behind `if !opts.Quick` |
| 4 | SessionStart hook surfaces drift via `--quick`, hermetic test stays green | `claude-session-start.sh` runs `dotf doctor --quick` gated on `command -v dotf` + a deployed `env-contract.json`; `session-start-false-positives.bats` green (its isolated `$HOME` has no contract → block skips) |
| 5 | `go test ./...` covers quick-mode section selection | `go test ./...` green; `TestRun_QuickSkipsHeavySections` is the coverage |

## Test status

- **Go:** `go build ./...` + `go test ./...` green; `go vet` + `gofmt -l` clean; `golangci-lint v2.12.2 run ./...` → 0 issues.
- **Shell:** `shellcheck scripts/claude-session-start.sh` → exit 0.
- **bats:** `session-start-false-positives.bats` + `setup-linux.bats` green — hermetic isolation preserved (the `--quick` block skips with no deployed contract); the SessionStart parity test updated to the CLI-013 reality.
- **Smoke:** `DOTFILES_DIR=<repo> dotf doctor --quick` → 4 contract sections, `Results: 14 passed, 0 failed`, exit 0, **0.00s**.

## Decisions made during implementation

- `--quick` gates the entire healthcheck sweep + heals behind `if !opts.Quick`, leaving exactly the doctor.sh-era env-contract scope. Report-only (ignores `--fix`) — the hook never mutates.
- Hermetic-test safety comes from gating the hook block on a **deployed `env-contract.json`** (not on the `dotf` binary): the isolated test `$HOME` has neither `DOTFILES_DIR` set nor `~/.dotfiles/env-contract.json`, so the block skips — the same "sibling absent → skip" property the old file-based guard had.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no (the lesson "measure before wiring a heavy tool into a hot path" was already captured in CLI-012's archive).
- [ ] ADR-worthy? no.
- [ ] New pattern? no.

## Archive checklist

- [ ] `proposal.md` → `status: archived` (on merge)
- [ ] Folder → `specs/archive/CLI-013-dotf-doctor-quick/` (on merge)
- [ ] #380 item 2 ticked (the Windows item 1 remains open)
