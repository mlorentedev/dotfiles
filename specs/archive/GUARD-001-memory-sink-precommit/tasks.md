---
tags: [spec, tasks, templates]
created: "2026-06-16"
---

# Tasks - GUARD-001-memory-sink-precommit

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/guard-memory-sink`
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [x] **R1 settled** (2026-06-16): Option A — multi-hook chaining dispatcher under a global `core.hooksPath` (dotfiles `.gitconfig`); one dispatcher per hook type, each execs the literal `.git/hooks/<type>`; per-repo layer installed as `.git/hooks/<type>` calling `pre-commit run`

## Implementation (TDD)

> The guard is a single POSIX `pre-commit` dispatcher under a tracked `git-hooks/` dir; setup/`dotf doctor` point `core.hooksPath` at it. Keep each step one commit.

- [x] Failing bats: in a non-vault fixture repo, `git commit` staging `MEMORY.md` → expect non-zero exit + vault message (AC1) — **RED**
- [x] Implement the dispatcher core: vault detection (sentinel `.obsidian/` + `$VAULT_PATH` fallback, R2) + staged-path scan (`MEMORY.md`, `memory/`, session-record shape, R4) + reject → **GREEN** (AC1)
- [x] Failing bats: same artifact inside a vault fixture (sentinel present) → expect exit 0 (AC2) — **RED** → **GREEN**
- [x] Failing bats: chaining — fixture repo with a pre-existing local `pre-commit` hook; assert both the GUARD and the local hook execute, local hook not clobbered (AC3) — **RED**
- [x] Implement chaining: dispatcher execs the repo-local hook after the guard passes (R1 decision) → **GREEN** (AC3)
- [x] Failing Go test: `initrepo` scaffold `.gitignore` contains the `MEMORY.md` / `memory/` block (AC4) — `TestScaffoldGitignoreHasMemorySinkBlock`
- [x] Implement the gitignore block — added to the embedded `cli/internal/initrepo/templates/gitignore` (the scaffold copies it verbatim) → **GREEN** (AC4)
- [x] Failing Go test: `dotf doctor` install is idempotent — wires `core.hooksPath` only when unset (under `--fix`); re-run on an already-correct value is a no-op; an unrelated pre-existing `core.hooksPath` is preserved, not clobbered (AC5) — `TestCheckGuardHooks_*` (5 cases)
- [x] Implement the install/repair check `checkGuardHooks` in `cli/internal/doctor/checks_guard.go` (wire `git config --global core.hooksPath` at the deployed `git-hooks/`, fail-fast if the dispatcher is undeployed) → **GREEN** (AC5). *Chosen over a bats end-to-end so CI never mutates a real `~/.gitconfig`; the `System` injection covers the full contract.*
- [x] AGENTS.md: added the **MEMORY SINGLE-SINK (GUARD-001)** rule once (vault = only memory sink; Hive = the memory API over it) + `tests/agents-md.bats` grep assertion (AC6)
- [x] Cross-OS parity: AC4/5/6 add no new shell — AC4 is a template, AC5 is Go (`dotf doctor`, cross-compiled), AC6 is docs; the POSIX dispatcher from #409 is unchanged (R5 holds, no `.ps1` mirror needed)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Type checks pass (`go build ./...`)
- [ ] Lint passes (shellcheck on the hook + `golangci-lint`)
- [ ] No unrelated changes in the diff (no scope creep — symlink extraction #8 and per-agent hooks #7 are sibling tickets, NOT this PR)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder + issue #398

## Machine-readable features

`features.json` is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence`.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit 0, may set that terminal state. Reviewers reject PRs where `features.json` has `passing` entries with empty `evidence`.
