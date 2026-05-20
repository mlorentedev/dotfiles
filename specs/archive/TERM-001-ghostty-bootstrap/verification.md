---
tags: [spec, verification, terminal, ghostty]
created: "2026-05-19"
---

# Verification - TERM-001-ghostty-bootstrap

Retrospective close-out: implementation landed on main in commits `11270f3` (proposal scaffold), `b00353e` (Ghostty bootstrap for Linux), and `7424731` (config comments translation). The 10 acceptance criteria are covered by existing artefacts already shipped; this PR formalises the audit trail.

## Evidence

- [x] AC1 `command -v ghostty` resolves on a deployed install → covered by `setup-linux.sh` block (lines 480-510) + healthcheck.sh Section 11/12 first check. Commit `b00353e`.
- [x] AC2 `ghostty --version` matches `GHOSTTY_VERSION=1.3.0` from versions.conf → healthcheck.sh Section 11/12 second check + `tests/ghostty.bats` test "GHOSTTY_VERSION matches semver pattern". Commit `b00353e`.
- [x] AC3 `~/.config/ghostty/config` contains font-family, theme, confirm-close-surface → `tests/ghostty.bats` tests 2-4. Commits `b00353e` + `7424731`.
- [x] AC4 `ghostty +validate-config` exits 0 → `tests/ghostty.bats` test 6 (skip if absent). Commit `b00353e`.
- [x] AC5 Two-tier sync (cmp $repo == $deploy) → setup-linux.sh `cmp -s` block + diff-check.sh integration in healthcheck Section 12/12. Commit `b00353e`.
- [x] AC6 Healthcheck section "Ghostty" with 3 OK lines (binary, version, config) → `scripts/healthcheck.sh` lines 345-372 (Section 11/12). Commit `b00353e`.
- [x] AC7 `tests/ghostty.bats` with ≥6 assertions → **10 assertions** shipped (see `tests/ghostty.bats`). Commit `b00353e`.
- [x] AC8 Existing bats suite green → confirmed via full regression: 673/673 pass post-BUG-007. Ghostty additions cause no regression.
- [x] AC9 CI green → spec-gate, lint, lint-powershell, integration, test all pass on current main.
- [x] AC10 Vault runbook `40-runbooks/guide-ghostty-setup.md` exists → verified via `ls ~/Projects/knowledge/10_projects/dotfiles/40-runbooks/`. Three sections present (GUI one-time steps, SSH terminfo workaround, recommended workflow).

## Test status

- `bats tests/ghostty.bats` → **10/10 pass** (verified just now on this branch).
- `bats tests/*.bats` → 673/673 pass (full regression remains green; close-out introduces zero code changes, only spec-folder edits).
- `shellcheck --severity=error setup-linux.sh scripts/healthcheck.sh` → clean.
- Vault runbook present at expected path.

## Decisions made during implementation (retrospective)

The bulk of implementation decisions were captured at commit-message time in `b00353e`. Three worth reiterating because they shape the ongoing maintenance contract:

- **Warn-not-fail on missing `ghostty` binary.** `apt install ghostty` requires sudo, which the rest of `setup-linux.sh` minimizes. Detect-and-act (log a warning, deploy the config anyway) keeps `setup-linux.sh` runnable without sudo on a fresh box; the user installs ghostty as a separate conscious step.
- **Theme name format gotcha guard.** Ghostty theme names are literal capitalized with spaces (`Catppuccin Mocha`), NOT kebab-case. `tests/ghostty.bats` test 5 catches the typo class with a negative regex. Captured empirically during AI-011-validation.
- **Strip `-dev+HASH` suffix when comparing versions.** Ubuntu universe ships `Ghostty 1.3.0-dev+0000000` as the "1.3.0" build. Both setup-linux.sh and healthcheck.sh use `sed 's/-.*//'` before comparison against `versions.conf`. Locked in via the bats `^[0-9]+\.[0-9]+\.[0-9]+$` semver pin assertion.

## Promotion candidates

- [x] Lesson for `90-lessons.md`? **Yes — already captured** during the implementation cycle (theme-name-format gotcha + version suffix stripping are both in lessons + the proposal "Risks" section). Promotion already executed.
- [ ] ADR-worthy? No — operational install, not architecture. ADR-009 (multi-agent runtime) covers the Ghostty + opencode + tmux split workflow rationale already.
- [ ] New pattern candidate for `00_meta/patterns/`? Yes potentially — the "detect-and-act + reconcile-not-skip + warn-not-fail" trio is the established pattern for sudo-requiring optional tools in setup-linux.sh (precedent: tmux, xclip, ghostty all share it). Defer formal pattern promotion until a second project adopts the same shape.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`.
- [x] Folder moved: `specs/TERM-001-ghostty-bootstrap/` → `specs/archive/TERM-001-ghostty-bootstrap/`.
- [x] Backlog entry in vault `11-tasks.md` ticked with PR link.
- [x] Promotions above executed (lesson already in vault).
