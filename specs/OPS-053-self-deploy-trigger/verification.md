---
tags: [spec, verification]
created: "2026-10-17"
---

# Verification - OPS-053-self-deploy-trigger

## Evidence

PR A (darwin), PR B (linux, windows, twin deletion) and PR C (last-run record, doctor) are below.

- [x] AC1 (darwin) -> `TestLaunchd_InstallThenCurrentThenRemove`, `TestSelfDeploy_OptedInInstallsThenIsConverged`, `TestNativeScheduler_PicksTheBackendOfTheRunsOS`; live run below
- [x] AC2 (linux) -> `TestSystemd*` in `cli/internal/schedule` (fake systemctl); live `changed=0` not proved here, needs msi
- [x] AC3 (windows) -> `TestSchtasks*`, `TestTaskXML*` (fake schtasks), `GOOS=windows go vet ./...`; live `changed=0` not proved here, needs the work-lab Windows box
- [x] AC4 (step rule) -> `TestSelfDeploy_WithoutTheRole`, `TestSelfDeploy_OptedInAdoptsAnUnmanagedTrigger`; darwin removal live below. Unmarked triggers exist only on linux/windows, so their detection lands with PR B
- [x] AC5 -> `TestLaunchAgentPlist_BytesArePinned`, golden written from `cdf425af` before the renderer moved
- [x] AC6 -> no `DOTFILES_AUTODEPLOY` in either setup script; `systemd/dotfiles-selfupdate.*` and `tests/dotfiles-selfupdate-install.bats` deleted; runbook rewritten around the role
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
- PR C: the native scheduler builder moved to `update.NativeBackend`, so the step and doctor address
  the same scheduler. It cannot live in `schedule`: `env` imports `schedule`, and the builder needs
  `env.ExecLaunchctl`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-468-a-machine-role-written-by-an-unreleased-dotf-breaks-the-released-one.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-019-self-deploy-fast-forward-only.md
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/OPS-053-self-deploy-trigger/` -> `specs/archive/OPS-053-self-deploy-trigger/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
