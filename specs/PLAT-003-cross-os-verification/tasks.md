---
tags: [spec, tasks, templates]
created: "2026-10-17"
---

# Tasks - PLAT-003-cross-os-verification

> TDD order. `[AC<n>]` maps a task to the acceptance criterion in `proposal.md`.

## Setup

- [x] Branch `feat/cross-os-verification` from main
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Open questions resolved (Windows doctor gate → module check warns; pwsh channel → `system` entry, not mise)

## PR 1 — Tier 1, local PowerShell shift-left

- [x] [AC2] Failing tests for `pwsh.ParseModules` (marker, near-miss, injection-shaped name/version), then the parser
- [x] [AC3] Failing tests for `pwsh.Missing` / `pwsh.Install` (one pwsh process, CI's flags, a failed pwsh is an error), then the package
- [x] [AC3] Failing tests for the `pwsh-modules` reconciler (plan installs nothing, apply installs only what is missing, probe names a module still missing, skip without pwsh), then the reconciler, wired after `tools`
- [x] [AC4] Failing table test for `checkPowerShell` (pass, module warn, pwsh fail where the catalog installs it, skip on linux), then the check
- [x] [AC5] Failing tests for `lintFiles` and `analyzerPin`, then `dotf lint powershell`
- [x] [AC6] Parity test against `ci.yml`'s `lint-powershell` job, parsed as YAML, with a token boundary; three mutations fail it
- [x] [AC1] `packages.json` `powershell` entry; `versions.conf` marks `PSSCRIPTANALYZER_VERSION` and `PESTER_VERSION`
- [x] [AC1] On the Mac: plan → apply → plan `0 to change`; doctor section green; lint clean in < 5 s and exit 1 on a probe finding
- [x] Lessons 466 and 467; follow-ups #2678 (hook + CI wiring) and #2677 (BUG-153, prune defect found on the way)

## PR 2 — Tier 3a, Lima Linux VM

- [x] [AC7] Declarative Lima template embedded in `dotf` (plain mode, `mounts: []`, base the image alone, no forwarded agent or host keys); a test pins each and seven mutations fail the suite
- [x] [AC7] `dotf vm linux`: boot, check systemd is PID 1 and no host directory is mounted, clone a pushed ref, converge twice, delete the guest (even on failure) unless `--keep`; GITHUB_TOKEN only on stdin
- [x] [AC7] Live run on the Mac: isolation checked in the guest
- [ ] [AC8] Live run green: blocked on #2688 (BUG-154, `install.sh` leaves `~/.local/bin` off PATH, found by this run); re-run `dotf vm linux --ref <fix>`
- [x] Lesson 473; `lima` catalog entry

## PR 3 — Tier 3b, Windows 11 ARM VM

- [x] Hypervisor chosen by the owner: QEMU driven by `dotf` (ADR-053)
- [x] [AC10] `qemu` and `swtpm` catalog entries; on the Mac: plan → apply → plan `0 to change`
- [ ] Owner: the Spanish Windows 11 Arm64 ISO at `~/.local/state/dotfiles/vm/windows/win11-arm64.iso`
- [ ] [AC9] Failing tests with a fake QEMU and guest agent, then `dotf vm windows`: build `clean-base` unattended, revert, copy a `git archive` of the ref in, run `install.ps1`, converge again, assert `changed` 0
- [ ] [AC9] Live run on the Mac; then #2683 (powercfg) and #2672 (sshd) verified in the guest, evidence on those PRs and #2356

## Closing

- [x] Every PR 1 acceptance criterion is covered by a test
- [x] `go build`, `go vet` (linux + windows), `go test ./...`, pinned golangci-lint, related bats
- [ ] `verification.md` complete for every tier
- [ ] Independent adversarial review before archive (`dotf spec review PLAT-003-cross-os-verification`)
