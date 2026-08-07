---
tags: [spec, tasks, templates]
created: "2026-08-07"
---

# Tasks - AI-028-hive-install-model-migration

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Decomposed into three PRs.** PR1 is independent and ships first — it converts the silent-failure class into a loud one, so the next recurrence surfaces in minutes instead of months. PR2 is gated on [hive#328](https://github.com/mlorentedev/hive/issues/328). PR3 is bookkeeping and can land any time after PR1.

## Setup

- [x] Worktree created from `origin/main`: `dotfiles-wt-hive-install-model`, branch `feat/hive-install-model-migration`
- [x] Gating issue open and assigned: dotfiles#791
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" — 1 `[AGENT-DRAFT]` remains (Linux path)

## Implementation

### PR1 — loud-vs-quiet failure (AC1) — independent, ships first

- [x] [P] [AC1] Write failing test in `tests/hive-upgrade-timer.bats`: with no hive install resolvable, the script emits a non-empty message naming the condition (not a bare `exit 0`).
- [x] [P] [AC1] Write failing test: with the install present and already at the latest version, the script stays silent (the deliberate 15-minute no-op is preserved — this guards against "fix the silence by making it noisy always").
- [x] [AC1] `windows/hive-upgrade.ps1`: split the single `-not $installed -or ... -ge ...` guard into **three** branches, not two. The plan said two; the collapsed guard actually hid a third outcome — PyPI unreachable — which is transient rather than a fault. Final contract: already-current → silent, `exit 0`; PyPI unreachable → message, `exit 0`; no install → message, **`exit 1`**.
- [x] [AC1] Exit non-zero on the no-install branch, so the condition surfaces in Task Scheduler's `LastTaskResult`. Stdout is not captured by the scheduler, so the exit code is the only signal visible without running the script by hand — and it is precisely the one that read green throughout the outage.
- [x] [AC1] Add a test pinning that the branches stay separate (`! grep -qF '-not $installed -or'`), so a future tidy-up cannot silently re-collapse them.
- [x] [AC1] Refactor: step-0 fast-path contract intact (still no daemon restart on a no-op tick); `.DESCRIPTION` documents the three outcomes; script stays ASCII-only for PSScriptAnalyzer.

### PR2 — bootstrap + A3 trigger (AC2, AC3, AC4) — gated on hive#328

- [ ] **GATE:** hive#328 shipped (PATH launcher installed by `self_upgrade`; `_resolve_exec()` no longer selects a dead binary) and published to PyPI.
- [ ] [P] [AC2] Write failing test: on a machine with no `hive` on PATH, the bootstrap command is the `uvx --from hive-vault hive self-upgrade` form (verified working 2026-08-07: `uvx --from hive-vault hive --version` → `hive-vault 1.43.0`, rc=0).
- [ ] [AC2] `mcp-servers.json`: replace `prerequisite_command` `uv tool install --upgrade hive-vault` with the bootstrap form. Keep `prerequisite_binary: uv` — it is the real and only prerequisite.
- [ ] [P] [AC3] Write failing test: the upgrade trigger invokes a bare `hive self-upgrade` with no `Stop-ScheduledTask` / `Start-ScheduledTask` around it.
- [ ] [AC3] `windows/hive-upgrade.ps1`: drop the A1 stop-before-upgrade orchestration (the A3 junction swap needs no daemon stop) and the defer-if-locked branch it existed to protect.
- [ ] [AC4] `setup-windows.ps1`: replace the `uv tool list` version gate before `hive service install` with a check against the resolved install, so a machine without a uv tool is bootstrapped rather than skipped.
- [ ] [P] [AC4] Write failing test: no Windows-path mechanism greps `uv tool list` to decide whether hive exists.
- [ ] [AC2] Real-hardware validation on the currently-broken box (the before/after table in `verification.md`).

### PR3 — decision reconciliation + docs (AC5, AC6) — independent of PR2

- [ ] [AC5] Amend `specs/AI-022-hive-daemon-activation/tasks.md`: the "A1, NOT A3" decision is superseded — hive spiked A3 on 2026-06-24 and shipped it in 1.43.0. Record the supersession with dates rather than deleting the original reasoning.
- [ ] [AC5] Add a forward pointer in `specs/archive/AI-023-hive-auto-upgrade-timer/` to this spec.
- [ ] [AC6] Record the Linux decision explicitly (resolve the `[AGENT-DRAFT]` in `proposal.md` first) in this spec and in the amended `AI-022`.
- [ ] Update `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`: either retire it (if the new model makes the failure unreachable) or point it at the new install model and at #574 for repair.
- [ ] Document the install model in one place — what owns the layout, what bootstraps it, what triggers upgrades, and which OS uses which mechanism.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Type checks pass (`pwsh` parse / shellcheck as applicable)
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` following [[pattern-feature-list-as-primitive]]. Authored once `tasks.md` freezes (after the Linux `[AGENT-DRAFT]` resolves), so verification commands name real test names rather than guesses.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.

**Verification must target the SSOT.** `~/.claude/scripts/hive-upgrade.ps1` is a *deployed copy*; the SSOT is `windows/hive-upgrade.ps1`. A verification command that runs the deployed copy without re-deploying first will pass against stale code.
