---
tags: [spec, tasks]
created: "2026-06-14"
---

# Tasks - CLI-012-dotf-doctor

> TDD order. One task = one focused step. Frozen once `implementing` began.

## Setup

- [x] Branch `feat/dotf-doctor` from main (rebased onto main: ADR-021 #375 + install-dotf fix #373)
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Open questions resolved with the user (full 12-section parity; `--fix` = report defaults + run heals; defer `.ps1` to a Windows follow-up)

## Implementation (TDD)

Foundations:

- [x] `internal/doctor/system.go` — `System` injection seam (Getenv / LookPath / CommandOutput); real wiring + helpers
- [x] `internal/doctor/version.go` + `version_test.go` — `compareVersions` (`sort -V` semantics), `atLeast`
- [x] `internal/doctor/config.go` + test — `versions.conf` native parser, DOTFILES_DIR/repo-root resolution
- [x] `internal/doctor/contract.go` + test — `env-contract.json` native model (drops the `jq` shell-out)
- [x] `internal/doctor/report.go` + test — Status/Report; exit 1 iff any FAIL; non-verbose pass-suppression

Checks (each: write table test → implement → refactor):

- [x] Contract sweep (doctor.sh): env vars (+ `--fix` profile-line reporting), PATH entries, required binaries
- [x] Core tools §1 (dedupes contract-covered git/jq) + versioned paths §2 + version match §3 (+ yarn/dotf drift = WARN)
- [x] Symlinks §4 incl. claude-mem install-state (BUG-014) + hook-path cascade (BUG-015 / resolveClaudeMemHook)
- [x] Tool-home env vars §5 + optional tools §6 (folds contract optional_binaries)
- [x] Vault presence §7 (read-only; deep vault-health deferred to `dotf vault`)
- [x] Secrets integrity §8, tmux §9, opencode/pi §10
- [x] Harness + skill-symlink drift §11 (diff-check deferred to a future `dotf doctor` mode), Antigravity §12
- [x] `fix.go` — `--fix` heal tail (invokes claude-mem-heal.sh; faithful to doctor.sh)

Command:

- [x] `internal/cmd/doctor.go` — cobra `doctor` with `--fix` / `--verbose`; exit contract via main.go; registered in root.go
- [x] `internal/cmd/doctor_test.go` — help/flag wiring + NoArgs rejection

## Migration (delete twins + repoint)

- [x] `git rm scripts/{healthcheck,doctor}.sh tests/healthcheck.bats`
- [x] `setup-linux.sh` — fold the two post-setup blocks into one `dotf doctor` call; drop the doctor.sh chmod
- [x] `scripts/claude-session-start.sh` — retire the per-session drift block (full sweep ~2.8s too heavy to fork per session; tracked for the hook port)
- [x] Repoint live refs: README, `docs/runbooks/guide-tmux.md`, `guide-antigravity-cli-migration.md`, `utils.sh` comment
- [x] Update the cross-OS bats to the asymmetric migration reality (Linux→`dotf doctor`, Windows `.ps1` kept): setup-linux, opencode, pi-config, compile-harness, healthcheck-ps1, env-contract anchor
- [x] Guard-grep: no live `(healthcheck|doctor).sh` reference remains (only provenance + an absence-assertion)
- [ ] **Deferred (tracked follow-up):** delete `healthcheck.ps1` / `doctor.ps1` + Pester + wire `setup-windows.ps1` to `dotf doctor` — gated on a Windows `install-dotf`

## Closing

- [x] Every acceptance criterion covered by ≥1 Go test or observable check
- [x] `go test ./...` green (67% stmt coverage in internal/doctor); `go vet` clean; `gofmt` clean
- [x] `shellcheck` clean on changed scripts; full `bats` suite green
- [x] Smoke: `DOTFILES_DIR=<repo> dotf doctor` on a real checkout → 118 passed / 0 failed / exit 0; `--fix` runs the heal
- [x] `verification.md` filled
- [ ] PR opened referencing this spec folder, closing #376
