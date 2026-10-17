---
tags: [spec, tasks]
created: "2026-10-17"
---

# Tasks - OPS-053-self-deploy-trigger

> TDD order. Three PRs: the executable diff of all seven criteria exceeds the 300-line cap.
> PR A is the shared renderer, the launchd backend and the step (darwin, proved live); PR B adds
> the systemd and Scheduled Task backends and deletes the setup-script twins; PR C is the
> last-run record and doctor.

## Setup

- [x] Branch `feat/self-deploy-trigger` from origin/main (`cdf425af`)
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Risks resolved or stated as a verification boundary

## PR A — the `self-deploy` step on darwin (AC1, AC4, AC5)

- [x] [AC5] Golden test: `env.LaunchAgentPlist` bytes as of `cdf425af`
- [x] [AC5] `schedule.LaunchdPlist` renders a job's plist; `env.LaunchAgentPlist` calls it; golden stays green
- [x] [AC1] `schedule` launchd backend: inspect (plist bytes + `launchctl print`), install (write, bootout if loaded and stale, bootstrap), remove (bootout, delete); fake-runner tests
- [x] [AC1] `update.Trigger`: the one job declaration (daily 03:00, `dotf update`, per-OS names)
- [x] [AC1] `env.RoleSelfDeploy` added to the known roles; role comment and `dotf env role` help no longer say every role is about the network
- [x] [AC1, AC4] `converge` step `self-deploy` (every OS) after `env-persist`, gated on the role, backend from `Options.Scheduler`; unset or unsupported → skip
- [x] [AC4] Marker rule in the step: role absent + managed → remove; unmanaged → reported, untouched
- [x] [AC1] Live on the Mac: role add → plan → apply → `changed=0` → `launchctl print`; then role rm → removal → skip
- [x] Runbook macOS subsection, ADR-019 amendment, lesson 468
- [x] Gates: go build/vet/test, `GOOS=windows go vet`, pinned golangci-lint

## PR B — systemd and Scheduled Task backends, twins deleted (AC2, AC3, AC4, AC6)

- [x] [AC2] `schedule` systemd backend: render service + timer with the marker, inspect (bytes + `is-enabled`/`is-active`), install (write, daemon-reload, enable --now), remove; fake-runner tests
- [x] [AC3] `schedule` schtasks backend: task XML with a digest marker in Description, inspect (`/Query /XML`), install (`/Create /XML /F`), remove (`/Delete /F`); logon type from an elevation probe; fake-runner tests
- [x] [AC4] Unmarked units/task (the setup scripts' install) read as unmanaged
- [x] [AC6] Delete the setup-linux.sh and setup-windows.ps1 blocks, `systemd/dotfiles-selfupdate.*`, `tests/dotfiles-selfupdate-install.bats`; fix the `systemd/hive.service.d/10-dotf-secrets.conf` mention
- [x] [AC6] `dotf update --help`, runbook rewritten around the role on every OS
- [x] Gates as above, plus bats for every file that references a changed script
- [x] Verification boundary in the PR: live `changed=0` on Linux (msi) and Windows (work-lab box) is not proved here

## PR C — last-run record and doctor (AC7)

- [x] [AC7] `update` writes `<StateDir>/update/last.json` {at, status, message, error} on every outcome; test with a temp StateDir
- [x] [AC7] doctor `Self-deploy` section: role, trigger state (the step's own inspect, read-only), last run; table-driven tests
- [x] Gates as above

## Closing

- [ ] Every acceptance criterion covered by a test, mapped in `features.json`
- [ ] `verification.md` filled in
- [ ] Independent review: `dotf spec review OPS-053-self-deploy-trigger` (not this session)
