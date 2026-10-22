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
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — the Linux `[AGENT-DRAFT]` is resolved

## Implementation

### PR1 — loud-vs-quiet failure (AC1) — independent, ships first

- [x] [P] [AC1] Write failing test in `tests/hive-upgrade-timer.bats`: with no hive install resolvable, the script emits a non-empty message naming the condition (not a bare `exit 0`).
- [x] [P] [AC1] Write failing test: with the install present and already at the latest version, the script stays silent (the deliberate 15-minute no-op is preserved — this guards against "fix the silence by making it noisy always").
- [x] [AC1] `windows/hive-upgrade.ps1`: split the single `-not $installed -or ... -ge ...` guard into **three** branches, not two. The plan said two; the collapsed guard actually hid a third outcome — PyPI unreachable — which is transient rather than a fault. Final contract: already-current → silent, `exit 0`; PyPI unreachable → message, `exit 0`; no install → message, **`exit 1`**.
- [x] [AC1] Exit non-zero on the no-install branch, so the condition surfaces in Task Scheduler's `LastTaskResult`. Stdout is not captured by the scheduler, so the exit code is the only signal visible without running the script by hand — and it is precisely the one that read green throughout the outage.
- [x] [AC1] Add a test pinning that the branches stay separate (`! grep -qF '-not $installed -or'`), so a future tidy-up cannot silently re-collapse them.
- [x] [AC1] Refactor: step-0 fast-path contract intact (still no daemon restart on a no-op tick); `.DESCRIPTION` documents the three outcomes; script stays ASCII-only for PSScriptAnalyzer.

### PR2 — bootstrap + A3 trigger (AC2, AC3, AC4) — gated on hive#328

> **Dispositioned 2026-10-21 (re-scope).** Each task carries what is true on main today. The live remainder moved to PR5–PR7 below.

- [x] **GATE:** hive#328 shipped (PATH launcher installed by `self_upgrade`; `_resolve_exec()` no longer selects a dead binary) and published to PyPI.
      *Code shipped:* `_resolve_exec()` hardening in hive#330, and the PATH launcher in hive `61e1089` (#365, 2026-08-09), first released in hive v3.1.0; `packages.json` pins 4.4.1. HIVE-328 is still `verifying` (33/36): its AC7, a fresh-shell check on Windows, is what PR6's guest run proves.
- [ ] ~~[P] [AC2] Write failing test: on a machine with no `hive` on PATH, the bootstrap command is the `uvx --from hive-vault hive self-upgrade` form.~~ *Moved to PR6*, whose shape is the owner's decision.
- [ ] ~~[AC2] `mcp-servers.json`: replace `prerequisite_command` with the bootstrap form.~~ *Superseded 2026-10-21:* #1993 retires `prerequisite_command` instead, and `dotf deploy` already ignores it (`cli/internal/claude/mcp.go:18`). The install moves to whatever PR6 decides.
- ~~[P] [AC3] Write failing test: the upgrade trigger invokes a bare `hive self-upgrade`.~~ *Withdrawn with AC3* (hive ADR-020).
- ~~[AC3] `windows/hive-upgrade.ps1`: drop the A1 stop-before-upgrade orchestration.~~ *Withdrawn with AC3:* PR5 deletes the script.
- [x] [AC4] `setup-windows.ps1`: replace the `uv tool list` version gate before `hive service install` with a check against the resolved install.
      *Done unticked:* `016bf1a8` (#1311, 2026-08-27). The gate is `dotf tools version hive` on both OSes, held by the parity test in `tests/hive-upgrade-timer.bats` (moves to `tests/hive-daemon-gate.bats` in PR5).
- [ ] [P] [AC4] Write failing test: no Windows-path mechanism greps `uv tool list` to decide whether hive exists. *Still open:* `windows/hive-upgrade.ps1:72` does. PR5 deletes it, and PR5's guard asserts no setup installs an unattended upgrade.
- ~~[AC2] Real-hardware validation on the currently-broken box.~~ *Withdrawn with AC7:* the guest run in PR6 replaces it.

### PR3 — decision reconciliation + docs (AC5, AC6) — independent of PR2

- [x] [AC5] Amend `specs/AI-022-hive-daemon-activation/tasks.md`: the "A1, NOT A3" decision is superseded — hive spiked A3 on 2026-06-24 and shipped it in 1.43.0. Record the supersession with dates rather than deleting the original reasoning.
- [x] [AC5] Add a forward pointer in `specs/archive/AI-023-hive-auto-upgrade-timer/` to this spec.
- [x] [AC6] Record the Linux decision explicitly (resolved: Linux keeps `uv tool`) in this spec and in the amended `AI-022`.
- [x] Update `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`: either retire it (if the new model makes the failure unreachable) or point it at the new install model and at #574 for repair.
- [ ] **Deferred to PR2 (deliberately).** Document the install model in one place — what owns the layout, what bootstraps it, what triggers upgrades, and which OS uses which mechanism. Writing it now would document the A1 state that PR2 replaces; the doc lands with the change it describes.

### PR4 — POSIX install pinned in the catalog (AC8, #1993) — independent of PR2

Ships on its own. AI-028 still cannot archive until PR2's Windows check, but the Linux half is done once this lands.

- [x] [P] [AC8] Failing tests: a `uv-tool` source installs with `uv tool install <package>==<pin>`, upgrades a lower version, skips one at or above the pin, and names the argv on failure (`TestInstallUvTool*`)
- [x] [P] [AC8] Failing test: a tool whose `platforms` exclude this OS is a skip, not a failure, and plans as `unsupported` (`TestInstall_PlatformsGateIsASkipNotAFailure`, `TestInstallerPlan`)
- [x] [AC8] Failing test: the uv tool is probed on PATH, not in `Dest`; probing `Dest` would read it as absent and reinstall it on every run
- [x] [AC8] Failing test: an unknown platform in the catalog fails the load (`TestLoad_Platforms`), since a typo would silently make the tool unsupported everywhere
- [x] [AC8] Implement the `uv-tool` source and `Source.Platforms`; `dotf tools list` shows `uv:<package>` or `(not installed on this platform)`
- [x] [AC8] Declare `hive` in `packages.json`: `uv-tool`, `hive-vault` 4.2.2, `platforms: [linux, darwin]` (`TestTheRepoCatalogDeclaresHiveForPosixOnly`)
- [x] Update the `tests/packages-json.bats` source-type guard to admit `uv-tool`
- [x] [AC8] Review fix (PR-Agent on #2001): `setup-linux.sh` runs `dotf tools install` before it installs uv, so a fresh box's first run has no uv. That is now a named skip, and the plan reports `missing-manager`, as a missing `prerequisite_binary` is on the MCP path (`TestInstallUvTool_MissingUvIsANamedSkip`)
- [x] Measured: uv 0.9.29 replaces a different installed version with the pin in either direction and exits 0 when the pin is already there, so one argv covers install and upgrade. On msi, `dotf tools install --dry-run` plans `hive 4.2.2 4.2.2 skip`

### PR5 — retire the unattended upgrade timers (AC9, #2857) — added 2026-10-21

Two halves. The deletion needs no decision; the retire step on live machines does.

- [x] [AC9] Failing guard: no setup script installs `hive-upgrade`, `DotfilesHiveUpgrade` or a `uv tool upgrade hive-vault`, and the units and script are gone (`tests/hive-daemon-gate.bats`).
- [x] [AC9] Delete `systemd/hive-upgrade.{service,timer}`, `windows/hive-upgrade.ps1`, the two setup blocks that install them, `Register-HiveScheduledTask` (its only caller was the task) and `tests/hive-upgrade-timer.bats`; the daemon-gate parity test moves to `tests/hive-daemon-gate.bats`
- [x] [AC9] Failing tests, then the `hive-upgrade-retire` converge step (owner, 2026-10-21: converge, not `doctor --fix`). It removes `hive-upgrade.timer`, the `DotfilesHiveUpgrade` task and the script copy at `~/.claude/scripts/hive-upgrade.ps1` from a machine that has them, unmanaged as they are, and a second run changes nothing (`TestHiveUpgradeRetire_*`, `cli/internal/converge/hiveupgrade.go`)
- [ ] [AC9] Live: `dotf converge --only hive-upgrade-retire` twice on msi (1 change, then 0), after a release carrying it
- [ ] [AC9] Live on the work-lab box, elevated (the task an elevated setup registered is S4U): the same two runs remove `DotfilesHiveUpgrade` and the script
- [x] [AC9] Edit the stale mentions outside the deletion: `docs/architecture.md:27,29` and the comments in `systemd/hive.service.d/10-dotf-secrets.conf:8,58`, plus `docs/troubleshooting/hive-mcp-orphaned-trampoline.md`, which now calls the timer historical
- [x] State in the PR: until #1993 retires `prerequisite_command`, Windows setup still runs `uv tool install --upgrade hive-vault` on every run

### PR6 — Windows installs hive through hive's own layout (AC2, AC4) — added 2026-10-21, gated on an owner decision

**Decided (owner, 2026-10-21): (A).** The two shapes that were weighed:

- **(A) A `packages.json` source for hive's own installer** (for example `type: hive-self`, run as `uvx --from hive-vault hive self-upgrade <pin>`, probed with `hive --version` in a new process). It keeps one install channel per tool class (ADR-036) and gives `dotf tools` the install to report on, and it is a new public contract in the catalog schema. It is #1993's direction.
- **(B) A dedicated `dotf converge` step for hive on Windows**, outside the catalog. It is narrower, a second place a tool is installed, and the catalog would still report hive as unsupported on Windows.

Either way:

- [ ] [AC2] Failing test first: a clean Windows guest gets hive from the step and not from `prerequisite_command`
- [ ] [AC4] Remove the Windows run of `prerequisite_command` (with #1993), so no `uv tool` copy of hive-vault is made on Windows
- [ ] [AC4] A missing hive is reported by the step that owns it, so the converge is not green while the doctor fails (Baseline 2, V1 and V5)
- [ ] [AC2] Guest run: `dotf vm windows --ref <PR6 ref>` flips V2–V6, and the second converge changes nothing. The second converge runs in a new process, which is HIVE-328's AC7 fresh-shell check. **Gated on hive#515** (no hive install succeeds on Arm64 today, V6); without it, run it on a clean x64 Windows machine
- [ ] Document the install model in one place, as deferred in PR3: what owns the layout, what installs it, that nothing upgrades it unattended (ADR-020), and which OS uses which mechanism

### PR7 — doctor names the right fix on Windows (#2856) — added 2026-10-21

- [x] Failing test: the not-installed fix follows the OS; Windows never gets `dotf tools install hive` (`TestHiveDaemon_NotInstalledFixFollowsTheOS`, landed in #2869, `c213ae90`)
- [x] `hiveInstallRemedy`: the catalog pin on Linux and macOS, hive's own bootstrap on Windows (#2869, `c213ae90`, closes #2856)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [ ] Type checks pass (`pwsh` parse / shellcheck as applicable)
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` following [[pattern-feature-list-as-primitive]]. Authored when PR2 lands, so verification commands name real test names rather than guesses. PR1's criterion (AC1) is already pinned by four named bats cases.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.

**Verification must target the SSOT.** `~/.claude/scripts/hive-upgrade.ps1` is a *deployed copy*; the SSOT is `windows/hive-upgrade.ps1`. A verification command that runs the deployed copy without re-deploying first will pass against stale code.
