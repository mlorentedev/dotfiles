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
  - #2688 is fixed in Go (option B, owner); the re-run waits for the `dotf` release that carries it
- [x] Lesson 473; `lima` catalog entry

## PR 3 — Tier 3b, Windows 11 ARM VM

- [x] Hypervisor chosen by the owner: QEMU driven by `dotf` (ADR-053)
- [x] [AC10] `qemu` catalog entry; on the Mac: plan → apply → plan `0 to change`. `swtpm` was declared, then removed: HVF cannot attach a TPM (ADR-053)
- [ ] Owner: the English (United States) Windows 11 Arm64 ISO, downloaded in a browser; the build takes it with `--iso`
- [x] [AC9] QEMU argv builder (shares nothing with the host, install media read-only, socket paths within the macOS limit) and guest-agent client (sync, exec, file read and write), tested against a fake agent; argv validated against QEMU 11.1.2
- [ ] PR 3a — `dotf vm windows build`: acquire and verify the ISO (`--iso`, a file or a download link) and virtio-win against SHA256 pins in the `vm` package (`versions.conf` holds semver values only); embedded `autounattend.xml` (LabConfig bypass, ARM64 virtio drivers, Pro selected by its published KMS client key, passwordless local account, guest agent, no device encryption, shut down at the end); QMP client; unattended install, answering the ISO's boot prompt with an arrow key (Enter reaches Setup's focused Cancel button, measured); a check boot (agent answers, no sshd); `clean-base` sealed read-only, disk and UEFI variables
- [ ] PR 3b — `dotf vm windows`: boot a throwaway qcow2 overlay of `clean-base` (refused while a guest runs, or without a sealed base), wait for the agent and the autologon, copy in a `git archive` of `--ref` (committed content only) and `guestrun.ps1` through the guest agent, run `install.ps1` then `dotf converge` as the guest user through a scheduled task, judge each on its own `last.json` (`Converged`, `Idempotent`), discard the overlay unless `--keep`; `dotf vm windows stop` ends a kept guest. No `GITHUB_TOKEN` reaches the guest in this version
- [ ] [AC9] Live run on the Mac; then #2683 (powercfg) and #2672 (sshd) verified in the guest, evidence on those PRs and #2356; BUG-155 (#2734) measured

## Closing

- [x] Every PR 1 acceptance criterion is covered by a test
- [x] `go build`, `go vet` (linux + windows), `go test ./...`, pinned golangci-lint, related bats
- [ ] `verification.md` complete for every tier
- [ ] Independent adversarial review before archive (`dotf spec review PLAT-003-cross-os-verification`)
