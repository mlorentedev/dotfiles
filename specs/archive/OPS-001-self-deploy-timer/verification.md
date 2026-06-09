---
tags: [spec, verification, templates]
created: "2026-06-09"
---

# Verification - OPS-001-self-deploy-timer

## Evidence

All acceptance criteria are covered by the bats suites
`tests/dotfiles-selfupdate.bats` (behavior) and `tests/dotfiles-selfupdate-install.bats`
(units + wiring). Full run: **24/24 passing**.

- [x] AC1 dirty worktree skip -> test `skips on a dirty worktree without running setup`
- [x] AC2 diverged/non-ff skip -> test `skips when local has diverged (non fast-forward) without running setup`
- [x] AC3 already current no-op -> test `is a no-op when already current (no setup run)`
- [x] AC4 clean fast-forward runs setup once -> test `fast-forwards and runs setup exactly once when the remote is ahead`
- [x] AC5 setup failure -> non-zero -> test `exits non-zero when the setup command fails`
- [x] AC6 opt-in gate (1/0/unset) -> tests `setup-linux.sh gates the timer on DOTFILES_AUTODEPLOY`,
      `enables the timer with --now when opted in`, `disables + removes the timer on opt-out (=0)`,
      `self-deploy block is non-fatal`
- [x] AC7 units valid -> tests `systemd unit files exist`, `fires daily with catch-up`,
      `installs to timers.target`, `oneshot running the repo selfupdate script`, `ExecStart is absolute`
- [x] AC8 cross-shell / ASCII -> tests `has valid bash syntax`, `has valid zsh syntax`,
      `scripts/dotfiles-selfupdate.ps1 exists and is ASCII-only`; shellcheck clean

## Test status

- New suites: `bats tests/dotfiles-selfupdate.bats tests/dotfiles-selfupdate-install.bats` -> **24 tests, 0 failures**.
- Lint: `shellcheck scripts/dotfiles-selfupdate.sh` -> clean; `shellcheck --severity=error setup-linux.sh`
  (the CI invocation) -> clean. `.ps1` files ASCII-only (`grep -P '[^\x00-\x7F]'` finds nothing new).
- Full suite regression gate: `bats tests/*.bats` -> the only non-passing entries are **pre-existing and
  unrelated**: 6 PowerShell tests skipped (`# skip pwsh not available` on this Linux box) and 3
  `shell-profile.bats` env-dependent failures that fail **identically on the untouched `main` checkout**
  (verified by running `bats /home/manu/Projects/dotfiles/tests/shell-profile.bats`). Zero regressions from this change.
- Live smoke (deferred to the user, by preference): enabling the real timer and observing a real
  fast-forward + setup run. The `fast-forwards and runs setup exactly once` test exercises the real
  `git fetch` + `--ff-only` path against a fixture remote, so the mechanism itself is proven.
- Windows runtime: authored (ps1 + Scheduled Task), runtime verification deferred to a Windows-box
  session (batch-windows-work).

## Decisions made during implementation

- The issue assumed `dotfiles-sync.sh` was the base; it actually pushes + rsyncs (opposite direction).
  Pivoted to a dedicated `dotfiles-selfupdate.sh` (verify-before-act).
- Made the setup command injectable via `DOTFILES_SELFUPDATE_SETUP_CMD` purely for testability
  (the suite injects a stub that records runs) — default stays `<repo>/setup-linux.sh`.
- No healthcheck assertion added: the timer is opt-in/default-OFF, so a healthcheck line would
  false-positive on every machine that opted out.
- `.claude/CLAUDE.md` left untouched: it is gitignored / local-only, not a tracked repo artifact.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **no** — the "verify-before-act on a stale issue premise"
      lesson already exists in the vault; nothing new generalizes beyond ADR-019.
- [x] ADR-worthy decision? **yes — done in this PR**: `docs/adr/adr-019-self-deploy-fast-forward-only.md`.
- [x] New pattern candidate for `00_meta/patterns/`? **no** — single-project ops automation; revisit only
      if a second repo grows the same self-deploy need.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-001-self-deploy-timer/` -> `specs/archive/OPS-001-self-deploy-timer/`
- [ ] Backlog entry (bitácora #295) closed with PR link
- [ ] Promotions above executed (ADR-019 shipped in-PR)
