---
tags: [spec, verification]
created: "2026-10-17"
---

# Verification - OPS-053-self-deploy-trigger

## Evidence

PR A (darwin), PR B (linux, windows, twin deletion) and PR C (last-run record, doctor) are below.

- [x] AC1 (darwin) -> `TestLaunchd_InstallThenCurrentThenRemove`, `TestSelfDeploy_OptedInInstallsThenIsConverged`, `TestNativeScheduler_PicksTheBackendOfTheRunsOS`; live run below
- [x] AC2 (linux) -> `TestSystemd*` in `cli/internal/schedule` (fake systemctl); live `changed=0` not proved: the `dotf vm linux` run below stopped before the step (#2688), and msi is not exercised
- [x] AC3 (windows) -> `TestSchtasks*`, `TestTaskXML*` (fake schtasks), `GOOS=windows go vet ./...`; live `changed=0` not proved here, needs the work-lab Windows box
- [x] AC4 (step rule) -> `TestSelfDeploy_WithoutTheRole`, `TestSelfDeploy_WithoutTheRoleNeverFailsOnAnUnreachableScheduler`, `TestSelfDeploy_OptedInAdoptsAnUnmanagedTrigger`; darwin removal live below. Unmarked triggers exist only on linux/windows, so their detection lands with PR B; on darwin `Managed` is constant (AC4 restated, review round 1 F4)
- [x] AC5 -> `TestLaunchAgentPlist_BytesArePinned`, golden written from `cdf425af` before the renderer moved
- [x] AC6 -> `tests/guard-setup-no-self-deploy.bats` (no self-deploy block in either setup script, the unit files stay deleted); `systemd/dotfiles-selfupdate.*` and `tests/dotfiles-selfupdate-install.bats` deleted; runbook rewritten around the role
- [x] AC7 -> `TestRecord_*` in `cli/internal/update`, `TestUpdate_ConvergesAfterAFastForward` (a record after each run), `TestRecordUpdate_KeepsTheFailureAndNeverFailsTheRun`, `TestCheckSelfDeploy_ByState` (14 verdicts), `TestTrigger_CarriesTheXDGHomesOfTheInstallingRun`; live below

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...`: all green.
- `golangci-lint run` at the pinned 2.12.2: 0 issues.
- Manual smoke test on the macmini (darwin, 2026-10-17, dev build in the scratchpad):
  - role declared, `dotf converge --only self-deploy`:
    `[CHANGE] self-deploy  applied: write /Users/manu/Library/LaunchAgents/com.github.mlorentedev.dotf.self-deploy.plist; load gui/501/com.github.mlorentedev.dotf.self-deploy`
  - second run: `[ OK ] self-deploy  daily dotf update at 03:00 from gui/501/com.github.mlorentedev.dotf.self-deploy` (changed=0)
  - `launchctl print gui/501/com.github.mlorentedev.dotf.self-deploy`: program `/Users/manu/.local/bin/dotf`, calendar interval Hour 3 Minute 0, PATH from the plist; `plutil -lint` OK
  - role removed, next run: `[CHANGE] self-deploy  remove gui/501/com.github.mlorentedev.dotf.self-deploy (the machine does not declare the self-deploy role)`; `launchctl print` then exits 113 and the plist is gone; the run after that is a skip
- PR C on the macmini (2026-10-17, dev build, a scratch clone of the repo, scratch `XDG_CONFIG_HOME`
  and `XDG_STATE_HOME`; the real `machine.json` is not written, lesson 468):
  - `dotf update` on the level clone: exit 0, `<state>/dotfiles/update/last.json` holds
    `"status": "current"`; with an untracked file in the clone: exit 0, `"status": "dirty"`
  - role declared in the scratch config, `dotf doctor`:
    `[WARN] gui/501/com.github.mlorentedev.dotf.self-deploy differs from the declared trigger: write …plist; load …; run `dotf converge --only self-deploy``
    (no agent is loaded on this Mac); without the role: `[SKIP] this machine has not opted in (dotf env role add self-deploy)`
  - the PASS, FAIL and stale-record verdicts need a loaded trigger and are unit-tested only
  - mutations, each turning a test red: the failed-run branch, the stuck-skip branch, the 3-day
    threshold widened to 30, the unmarked-trigger branch, the `recordUpdate` call, `XDG_STATE_HOME`
    dropped from the trigger
- No regressions in existing test suite: yes.

## Decisions made during implementation

- Split into three PRs: the full change measured 565 executable lines against a 300-line cap.
- The step sits right after `env-persist`, not just before `legacy-setup`: `agent-extensions` must stay the last native step (#2387).
- The launchd backend reports every agent at its label as managed: no setup script ever wrote one, so there is nothing to adopt on darwin.
- The live test declared the role in the real `machine.json`, and the installed 0.73.0 refused it, failing every converge on the machine for about 25 minutes (lesson 468). Rolled back; the role is declared on the Mac only after a release that knows it is installed.

- PR C: a run skipped on a dirty, diverged or upstream-less checkout (and `not-a-repo`, `ff-failed`) is
  a WARN although AC7 names only failure and age: those skips recur every night until someone acts,
  and the dirty one kept a timer green while the deploy never ran (#694). `offline` and `ahead` pass.
- PR C: the trigger carries `XDG_STATE_HOME` as it carries `XDG_CONFIG_HOME`; without it a shell that
  exports one would have doctor read a record the job never writes. A machine that exports it sees a
  one-time `changed=1` on the next converge.
- PR C: the Windows task does not carry `Job.Env` (Task Scheduler has no per-task environment), so
  there a session-only `XDG_CONFIG_HOME` or `XDG_STATE_HOME` does not reach the run; a value
  persisted at user scope does. Found in review, filed as #2696 with three options.
- PR C: the native scheduler builder moved to `update.NativeBackend`, so the step and doctor address
  the same scheduler. It cannot live in `schedule`: `env` imports `schedule`, and the builder needs
  `env.ExecLaunchctl`.

## Review round 1 (FAIL at `9f596c94`): dispositions

| Finding | Disposition |
|---|---|
| F1 role read after the scheduler | **Applied.** `inspect` reads the role first. Without it, a scheduler that cannot be built or inspected is a skip naming the error, and nothing is installed or removed. With it, the error stays a failure. A reachable scheduler with no role keeps AC4's removal of a trigger dotf wrote. `TestSelfDeploy_WithoutTheRoleNeverFailsOnAnUnreachableScheduler` (both the Inspect and the builder path, plan and apply, probe) was red before the fix: `err = systemctl --user show: Failed to connect to bus`. |
| F2 no live Linux proof | **Run, blocked by #2688.** Output below. The guest's first converge fails at `records-harness` (BUG-154: `install.sh` leaves `~/.local/bin` off PATH), so `self-deploy` is "not reached" and the run says nothing about the step. The guest also converges the pinned release (0.74.0), not this tree, so this fix round is only certifiable there after a release. Windows live still needs the work-lab box. |
| F3 `%` in `ExecStart` and `Description` | **Applied.** Every `%` the job carries is doubled in the argv, the description and the environment; the comment now says where. `TestSystemdUnits_DoublesPercentEverywhereTheJobReaches` was red before the fix. |
| F4 darwin `Managed` constant | **Applied to the contract.** AC4 and the marker rule in `proposal.md`, and `features.json` f4, now say the unmarked case binds on linux and windows only. On darwin no setup script ever wrote an agent at the label, so none is adopted. |
| F5 the twins could return unremarked | **Applied.** `tests/guard-setup-no-self-deploy.bats`, mutation checked: a `DOTFILES_AUTODEPLOY` line appended to a copy of `setup-windows.ps1` turns case 1 red. Provenance of the deleted `tests/dotfiles-selfupdate-install.bats` (`git log --diff-filter=A/D`): added in `b29b781a` (#303, OPS-001), deleted in `ba750094` (#2693). |
| F6 two Windows registrars | **Ticketed #2715** (root cause and options in the body). The Env half is #2696. |
| F7a hardcoded 03:00 | **Applied.** The OK detail reads `job.Daily`. |
| F7b `ago()` of a future record | **Declined.** Only clock skew produces it; the age is under the threshold, so the verdict is PASS and only the wording is odd. |
| F7c "applied:" before the error | **Declined.** A failed install appends `; failed:` to the same line, as the review notes, so the line stays truthful. |
| F9 stale Closing boxes | **Applied** in `tasks.md`. The independent-review box stays open until round 2 passes. |

`dotf vm linux --ref main` from this branch's `cli/` (`go run ./cmd/dotf`), 2026-10-17, exit 1, 730 lines; the converge table:

```
==> boot dotf-linux-1792263536 (Ubuntu 24.04, nothing shared with the host)
==> isolated: systemd is PID 1, no host directory mounted, HOME=/home/manu.guest
==> clone https://github.com/mlorentedev/dotfiles.git at main
==> first converge: ./install.sh
[INFO] Creating directory: /home/manu.guest/.local/bin
[SUCCESS] Directory created: /home/manu.guest/.local/bin
[SUCCESS] dotf 0.74.0 installed to /home/manu.guest/.local/bin/dotf
converge on linux:
  [SKIP]   checkout            354ms  no upstream configured for the current branch of /home/manu.guest/Projects/dotfiles; not fast-forwarded
  [CHANGE] records-mirror       10ms  harness/ + 3 target(s) + the deploy-dir set → /home/manu.guest/.dotfiles (145 updated, 0 unchanged, 0 pruned)
  [CHANGE] env-generate          1ms  /home/manu.guest/.dotfiles/paths.sh written
  [FAIL]   records-harness     802ms  3 instruction file(s) deployed: .claude/CLAUDE.md, .config/opencode/AGENTS.md, .pi/agent/AGENTS.md; failed: compile-harness.sh --deploy: exit status 2
  [SKIP]   tools                      not reached
  [SKIP]   pwsh-modules               not reached
  [SKIP]   pi-settings                not reached
  [SKIP]   configs-deploy             not reached
  [SKIP]   shell-completions          not reached
  [SKIP]   records-bind               not reached
  [SKIP]   git-config                 not reached
  [SKIP]   git-lfs                    not reached
  [SKIP]   git-hooks                  not reached
  [SKIP]   env-persist                not reached
  [SKIP]   self-deploy                not reached
  [SKIP]   workstation-target         not reached
  [SKIP]   agent-extensions           not reached
  [SKIP]   legacy-setup               not reached
2 changed, 0 ok, 0 opaque, 15 skipped, 1 failed (after 3 changes)

failed:
  records-harness: 3 instruction file(s) deployed: .claude/CLAUDE.md, .config/opencode/AGENTS.md, .pi/agent/AGENTS.md; failed: compile-harness.sh --deploy: exit status 2
the first converge did not succeed (result failed; failed: records-harness)
exit status 1
```

## Archive decision

Archived on the round-2 verdict, **PASS-WITH-GAPS** at `12818876` (nan/mimo-v2.6-flash, `review.md`). The owner chose option A on #2502 (2026-10-17) and accepted the gaps, which stay open in #2727: no live Linux or Windows `changed=0` until a release carrying #2721 is pinned, no test for `selfDeployWhen`, and the stale F2 row above.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-468-a-machine-role-written-by-an-unreleased-dotf-breaks-the-released-one.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-019-self-deploy-fast-forward-only.md
- [x] New pattern candidate for `00_meta/patterns/`? no: the scheduling seam is specific to `dotf`; the cross-project rule it rests on (a strict reader makes a new value a breaking change for older binaries) is lesson 468, in this repo.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/OPS-053-self-deploy-trigger/` -> `specs/archive/OPS-053-self-deploy-trigger/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
