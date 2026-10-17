---
id: "OPS-053-self-deploy-trigger"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-17"
issue: "mlorentedev/dotfiles#2502"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, self-deploy, converge, launchd, systemd, scheduled-task]
template_version: "1.0"
wip_override: "owner authorised the override for this spec only (2026-10-17): the Mac never self-deploys and was 21 commits behind on 2026-10-14 (13 active, limit 10, 2026-10-17)"
---

# OPS-053: one converge step owns the self-deploy trigger on every OS

## Why

<!-- from issue #2502: OPS-053: macOS has no self-deploy scheduler, so dotf update never runs unattended on the Mac -->

`dotf update` (fast-forward the checkout, then converge) runs unattended only where a setup
script installed a trigger for it: a systemd `--user` timer from `setup-linux.sh` and a
Scheduled Task from `setup-windows.ps1`, both behind `DOTFILES_AUTODEPLOY`. macOS has no setup
script any more, so nothing ever schedules it: on 2026-10-14 the Mac's main checkout was 21
commits behind and nothing had tried to move it. The opt-in is also an environment variable read
at setup time, so no later command can tell whether a machine opted in, and nothing reports
whether the last unattended run worked.

## What

- A machine opts in by declaring the role `self-deploy` in `machine.json`
  (`dotf env role add self-deploy`). The role replaces `DOTFILES_AUTODEPLOY` (ADR-019 decision 5
  amended): it persists, and doctor can read it.
- A new `dotf converge` step, `self-deploy`, owns the trigger on all three OSes from one job
  declaration (`~/.local/bin/dotf update`, daily):
  - **darwin:** a LaunchAgent in `~/Library/LaunchAgents`, `StartCalendarInterval`, loaded into
    `gui/<uid>`;
  - **linux:** `dotfiles-selfupdate.{service,timer}` in the systemd user dir, enabled `--now`;
  - **windows:** the `DotfilesSelfUpdate` Scheduled Task, registered from task XML through
    `schtasks /Create /XML`, S4U when the caller is elevated and InteractiveToken otherwise (the
    principal rule of `Register-HiveScheduledTask`).
  Plan reports what would change; apply makes it; a second apply reports `changed=0`; the probe
  re-inspects.
- Everything the step writes carries an ownership marker. With the role absent, the step removes
  a marked trigger (the opt-out after `dotf env role rm self-deploy`) and never touches an unmarked
  one: a trigger a setup script installed before this change is reported, with the command that
  adopts it, instead of being removed.
- The per-OS rendering lives in one stdlib-only package (`internal/schedule`): a `Job` plus three
  renderers and three backends behind injected runners. It is the seed of the scheduling table of
  #2013 D5 / #1843 B8; `dotf schedule` itself is not built here.
- The self-deploy blocks of `setup-linux.sh` and `setup-windows.ps1` are deleted, with the
  checked-in systemd units they copied and the bats file that grepped them.
- `dotf update` records each run's outcome (`<StateDir>/update/last.json`), and `dotf doctor`
  reports, on every OS: whether the machine opted in, whether the trigger is current, and the last
  run's outcome and age.

## Out of scope

- `dotf schedule install/status/remove` and the scheduling table as data (#1843 B8, #2013 D5,
  still proposed).
- The hive-upgrade timer and task (D5 decides port or retire against the ADR-036 floor).
- Bumping pins (#1267, OPS-035).
- Making `dotf update` itself work under launchd's bare PATH beyond what the job sets (measured,
  not assumed: see Risks).

## Risks / open questions

- **Silent opt-out on upgrade.** Machines that set `DOTFILES_AUTODEPLOY=1` (msi, the work-lab box)
  have an unmarked trigger and no role at their first converge after this lands. Resolved by the
  marker rule above: unmarked is never removed; the step and doctor name
  `dotf env role add self-deploy`. The PR and the runbook state the migration.
- **env-persist's plist bytes.** `LaunchAgentPlist` is compared byte-for-byte by env-persist; a
  generalised renderer must reproduce today's bytes exactly, or every Mac rewrites and reloads its
  login agent on the next converge. Pinned by a golden test.
- **launchd catch-up differs from `Persistent=true`.** A `StartCalendarInterval` slot missed while
  asleep runs on wake; one missed while powered off does not run. Stated in the runbook.
- **launchd PATH.** The agent sets `PATH` explicitly (`~/.local/bin`, Homebrew, system dirs).
  Whether a full converge under launchd reaches every tool is not proved tonight: kickstarting the
  job would run `dotf update` on the shared main checkout under peer sessions. The first scheduled
  run's record is the evidence; doctor reports it.
- **Windows and Linux are proved by unit tests only** on this Mac (fake `systemctl`/`schtasks`
  runners, `GOOS=windows go vet`). Live `changed=0` needs msi and the work-lab box.
- **`schtasks /Query /XML` normalises** the stored XML, so drift is read from the marker the
  Description carries (a digest of the rendered task), never by comparing XML.

## Acceptance criteria

- [ ] **AC1** On darwin, with the role declared, `dotf converge --only self-deploy` installs and
  loads the LaunchAgent; a second run reports `changed=0`; `launchctl print gui/<uid>/<label>`
  shows it. Proved live on the Mac.
- [ ] **AC2** On linux, the step writes both units, reloads, and enables the timer; a second run
  is `changed=0`; the probe fails when the timer is not enabled. Unit tests with a fake runner.
- [ ] **AC3** On windows, the step registers the task from XML with the elevation-dependent logon
  type; a second run is `changed=0` (marker digest equal). Unit tests with a fake runner;
  `GOOS=windows go vet ./...` clean.
- [ ] **AC4** Role absent: a marked trigger is removed, an unmarked one is left in place and
  reported with the adopt command, and no trigger at all is a skip naming the opt-in. Unit tests
  per OS.
- [ ] **AC5** env-persist's plist is byte-identical to the pre-change renderer (golden test).
- [ ] **AC6** No self-deploy block remains in `setup-linux.sh` or `setup-windows.ps1`
  (`git grep -n 'DOTFILES_AUTODEPLOY\|DotfilesSelfUpdate\|dotfiles-selfupdate' -- setup-linux.sh setup-windows.ps1`
  is empty); the systemd unit files and their bats are gone; the runbook and `dotf update --help`
  describe the role.
- [ ] **AC7** `dotf update` writes `<StateDir>/update/last.json` on every outcome; doctor's
  self-deploy section reports opt-in, trigger state, and the last run (FAIL on a failed run, WARN
  when absent or older than 3 days, WARN on an unmarked trigger). Unit tests.

## References

- Issue: #2502. Epic context: #1843 (B8), #2013 (D5).
- ADR-019 (amended here), ADR-045 (one entrypoint converges every OS), ADR-041.
- Prior art: `cli/internal/converge/envpersist.go` (LaunchAgent + marker), `workstation.go` and
  `doctor/checks_workstation.go` (role-gated step + read-only doctor twin).
