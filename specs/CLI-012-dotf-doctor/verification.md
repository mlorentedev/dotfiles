---
tags: [spec, verification]
created: "2026-06-14"
---

# Verification - CLI-012-dotf-doctor

## Evidence

| # | Acceptance criterion | Proof |
|---|---|---|
| 1 | `dotf doctor` reproduces the healthcheck + doctor checks; exit 0 on all-pass, non-zero on any failure with a per-failure report | `internal/doctor/*_test.go` (core tools, versioned paths, version match, symlinks, env vars, optional tools, env-contract, secrets, tmux, opencode, harness, antigravity); `TestRun_ExitContract` proves exit 1 + `Results:` line on a failing check; smoke run below |
| 2 | `dotf doctor --fix` applies the env-contract safe defaults doctor.sh applied (no heal-path regression) | `TestCheckContractEnvVars` (fix mode prints the `export VAR=` profile line); `fix.go` invokes `claude-mem-heal.sh`; smoke `--fix` ran the heal (10 `.mcp.json` patched) |
| 3 | `scripts/{healthcheck,doctor}.sh` and their bats no longer exist | `git rm` of `healthcheck.sh`, `doctor.sh`, `tests/healthcheck.bats`. **Scope note:** the `.ps1` twins + Pester are intentionally KEPT (deferred Windows follow-up — see Decisions) since `dotf` is not yet installed on Windows |
| 4 | No live reference to the retired scripts outside provenance | Guard-grep `grep -rEn '(healthcheck\|doctor)\.sh'` returns only: provenance (CHANGELOG/ADR/specs), inline migration-note comments, and one absence-assertion test (`[ ! -f doctor.sh ]`). Zero sourcing/exec of the deleted scripts |
| 5 | `setup-linux.sh` post-setup invokes `dotf doctor`; CLAUDE.md / AGENTS.md / docs point to it | `setup-linux.sh` folds the doctor+healthcheck blocks into one `dotf doctor` call; README, `guide-tmux.md`, `guide-antigravity-cli-migration.md`, `utils.sh` repointed. (AGENTS.md/CLAUDE.md had no live `.sh` doctor/healthcheck reference to repoint) |
| 6 | `go test ./...` covers the check logic; smoke-tested end-to-end | `go test ./...` green; `internal/doctor` 67.0% stmt coverage; smoke on the real worktree below |

## Test status

- **Go:** `go test ./...` → all packages `ok`; `internal/doctor` coverage **67.0%** of statements. `go vet ./...` clean; `gofmt -l` clean.
- **Shell:** `shellcheck setup-linux.sh scripts/claude-session-start.sh scripts/utils.sh` → exit 0 (only pre-existing SC1091/SC2015 info notes, none introduced). `bash -n` clean.
- **bats (full suite):** 1104 / 1107 pass. The 3 failures are all in `tests/shell-profile.bats` (the `profile-shell` timing harness) and are **pre-existing on clean `main`** (reproduced on the untouched checkout) — environment-related, unrelated to this change.
- **Smoke (real checkout):** `DOTFILES_DIR=<repo> dotf doctor` → `Results: 118 passed, 0 failed, 1 warned, 2 skipped`, exit 0. The one WARN is a genuine pinned-version drift (pi 0.79.2 vs 0.79.1, advisory). `--fix` ran `claude-mem-heal.sh` (patched 10 `.mcp.json`). `--verbose` lists passing checks.
- **No regressions:** confirmed against clean `main` (same 3 shell-profile failures, nothing else).

## Decisions made during implementation

- **Full 12-section parity** (user decision): `dotf doctor` reproduces all of healthcheck §1–12 + doctor's env-contract, MINUS the two explicitly out-of-scope items — `diff-check` (deferred to a future `dotf doctor` mode) and deep vault-health (the `vault-health.sh` invocation + linter check → `dotf vault`). Vault **presence** checks are kept.
- **`--fix` contract** (user decision): a Go subprocess cannot `export` into the parent shell, so doctor.sh's "apply default to session" was always ephemeral. `--fix` now prints the exact profile `export` line for each missing env default and runs the real heal (`claude-mem-heal.sh`). No behavioural regression in the durable heal path.
- **Defer the `.ps1` twins (Option B).** `dotf` is not installed on Windows (no Windows `install-dotf`), so deleting `healthcheck.ps1`/`doctor.ps1` would remove Windows post-setup diagnostics with no replacement and require editing `setup-windows.ps1` + the Windows CI blind. This PR is the **Linux port**: `.sh` deleted, `.ps1` kept and working. The cross-OS parity bats were rewritten to the asymmetric migration reality (Linux→`dotf doctor`, Windows→`.ps1` pending). Tracked follow-up: a Windows `install-dotf` + wire `dotf doctor` + delete the `.ps1`.
- **Session-start hook drift block removed, not repointed.** The full `dotf doctor` sweep is ~2.8s (dominated by the `compile-harness.sh --check` fork) — too heavy to run on every Claude session start, and repointing to a PATH command would break the hermetic isolation `session-start-false-positives.bats` relies on. Drift is still surfaced post-setup and on demand. A focused `dotf doctor --quick` belongs with the SessionStart hook port (ADR-021 roadmap step 6).
- **Consolidation, not transliteration:** contract-covered binaries (git, jq) are checked once (version) instead of twice; contract `optional_binaries` fold into the optional-tools section; `DOTFILES_DIR` is owned by the contract section, dropped from the tool-home list — removing the dual-maintenance the two scripts carried.

## Promotion candidates

- [x] Lesson for `docs/lessons.md`? **yes** — "Deleting an `.sh` twin while keeping its `.ps1` sibling forces asymmetric cross-OS parity tests; the cleaner unit is one-OS-per-PR with the parity tests rewritten to the migration reality, not symmetric." Also: "A consolidated diagnostic that shells out to `compile-harness --check` costs ~2.8s — fine on-demand, too heavy for a per-session hook."
- [ ] ADR-worthy decision? no (executes ADR-021; no new architectural decision).
- [ ] New cross-project pattern? no (dotfiles-specific).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` (on merge)
- [ ] Folder moved: `specs/CLI-012-dotf-doctor/` -> `specs/archive/CLI-012-dotf-doctor/` (on merge)
- [ ] Backlog: close #376 (the built-in workflow sets Done)
- [ ] Promotions above executed (lesson → `docs/lessons.md`)
