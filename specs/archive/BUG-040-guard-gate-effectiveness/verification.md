---
tags: [spec, verification, templates]
created: "2026-08-07"
---

# Verification - BUG-040-guard-gate-effectiveness

## Evidence

- [x] **AC1 — Equivalent dispatcher passes** -> Go `TestCheckGuardHooks_EquivalentDispatcherElsewherePasses`; bats `an equivalent dispatcher at another path is reported active, not INACTIVE`
- [x] **AC2 — Foreign hooksPath still warns** -> Go `TestCheckGuardHooks_ForeignPreCommitStillWarns`; bats `a pre-commit without the memory-sink guard still warns` (no-clobber asserted in both)
- [x] **AC3 — Separator normalization** -> Go `TestCheckGuardHooks_SeparatorAndTrailingSlashNormalize`; bats `a trailing-slash variant of the target counts as already wired`
- [x] **AC4 — No behaviour change on settled paths** -> the 5 pre-existing Go tests and 9 pre-existing bats tests pass unmodified
- [x] **AC5 — Regression asserted** -> red-first confirmed: before the fix, AC1 and AC3 failed and the other six Go tests passed

## Test status

- Go: `go vet ./internal/doctor/` clean; `go test ./internal/doctor/ -run TestCheckGuardHooks` -> **8/8 PASS**
- bats: `bats tests/install-git-hooks.bats` in `ubuntu:26.04` with bats `1.13.0` (the
  `versions.conf` pin that `tests/Dockerfile.integration` consumes) -> **12/12 PASS**
  (9 pre-existing + 3 new)
- PowerShell: parse errors 0; non-ASCII bytes 0 (`pattern-powershell-ascii-only`);
  PSScriptAnalyzer reports only pre-existing `PSAvoidUsingWriteHost` /
  `PSUseShouldProcessForStateChangingFunctions`
- Manual smoke: this machine is the reproducer — `core.hooksPath` =
  `C:/Users/mlorente/Projects/Workspace/dotfiles/git-hooks`, target =
  `C:\Users\mlorente\.dotfiles\git-hooks`, `pre-commit` identical in both
  (SHA256 `CD3A25D3…37F3`), and setup printed `the memory-sink guard is INACTIVE`
- No regressions: yes — AC4 above

## Decisions made during implementation

- **Effectiveness is structural, not a marker grep.** "Is this a GUARD dispatcher"
  tests for `pre-commit` **and** `lib/memory-sink-guard.sh`, because `pre-commit`
  execs that script. A comment marker could be edited without changing behaviour.
- **Tier 2 is a PASS, not a softened WARN.** The check's contract is "is the guard
  active", and it is. The active path is printed so a divergent setup stays visible
  instead of being silently blessed.
- **Fixed in three places rather than collapsed.** The deploy half must stay in shell
  (ADR-020 C7: the `dotf` release carries no source tree, so it cannot place
  `git-hooks/`). Collapsing the *wiring* half is real and filed as **#793**, not done
  here — it needs a setup-ordering change (`install-git-hooks` runs at
  `setup-windows.ps1:346`, `dotf` installs at `:554`).

## Environment findings surfaced while verifying

Neither is caused by this change; both reproduced against untouched `main`.

1. **`tests/install-git-hooks.bats` cannot pass under Git Bash on Windows** — tests
   1-4 fail identically on a clean `main` checkout (path translation of
   `GIT_CONFIG_GLOBAL` values; `-x` bits on NTFS). They look like regressions and are not.
2. **A linked worktree cannot be bind-mounted into a Linux container for git work** —
   a worktree's `.git` is a *file* holding `gitdir: C:/Users/...`; inside the container
   that resolves nowhere, so every `git` call with CWD under the mount aborts with
   `fatal: not a git repository`, including `git config --global`. Verification ran
   against a copy with the `.git` pointer removed. Same family as #776.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — a guard check must test
      effectiveness, not path identity; and the two environment findings above are
      exactly the kind of false signal that wastes a debugging hour.
- [ ] ADR-worthy decision? no — ADR-020 already governs the layering; #793 carries the follow-through.
- [ ] New pattern candidate for `00_meta/patterns/`? no — repo-specific, not yet seen in >1 project.

## Remaining work

- Pester case for `Test-GuardDispatcher` / `Test-SameHooksPath` in
  `tests/install-git-hooks.Tests.ps1`. Tracked rather than written blind, for the
  same reason as #781: it needs a real Windows pwsh run to mean anything.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-040-guard-gate-effectiveness/` -> `specs/archive/BUG-040-guard-gate-effectiveness/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
