---
id: "PLAT-003-cross-os-verification"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-10-17"
issue: "mlorentedev/dotfiles#2344"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner-authorised 2026-10-17 for PLAT-003 only: Tier 1 local pwsh shift-left unblocks #2013 T5/X3 (13 active, limit 10, 2026-10-17)"
---

# PLAT-003: Local cross-OS verification fabric

## Why

<!-- from issue #2344: PLAT-003: Local cross-OS verification fabric — PowerShell shift-left + Lima/VMware snapshot-resettable VM harness -->

Linux and Windows behaviour can only be verified today by pushing to CI or by sitting at a physical Linux or Windows machine. `lint-powershell` is a required check on `main`, yet `versions.conf` says of its pin "a CI pin only: no machine provisions it", so a `.ps1` lint error costs a push and a 9–16 minute `test-windows` round trip. Clean-install and daemon tests wait on hardware (#2013 T5, X3). The owner works from a Mac (Apple Silicon); this spec makes that Mac the place where Linux and Windows behaviour is checked first.

## What

Three tiers, one PR each, in this order. Tier 2 is deferred.

- **Tier 1 — local PowerShell shift-left (PR 1).**
  - `packages.json` gains a `powershell` system entry (`brew: powershell`, `winget: Microsoft.PowerShell`, `command: pwsh`). That is how PLAT-001c provisions an unpinned OS package; mise is for pinned single-binary CLIs, and its `powershell` tool would need a `powershell`→`pwsh` bin mapping in a released `dotf` before a marker could land. CI's own `pwsh` is the runner image's and is unpinned too, so presence is the parity that matters. Linux has no entry: Ubuntu's archive does not carry PowerShell, and CI's lint job stays the Linux gate.
  - `versions.conf` marks `PSSCRIPTANALYZER_VERSION` and `PESTER_VERSION` with `# pwsh-module: <Name>`. A new `dotf converge` reconciler, `pwsh-modules`, installs each marked module at its exact pin into the CurrentUser scope, with the same `Install-Module` flags CI uses, and its probe imports each at that version. Skipped where `pwsh` is not on PATH.
  - `dotf doctor` gains a PowerShell section: `pwsh` absent where the catalog names a manager for this OS fails; a marked module not importable at its pin warns, naming `dotf converge --only pwsh-modules`.
  - `dotf lint powershell` runs PSScriptAnalyzer at the pin over the checkout's `.ps1`/`.psm1` files with CI's settings file and severities, and exits 1 on any finding. A Go test reads `ci.yml` and fails when the job's parameters drift from the command's.
- **Tier 3a — Lima Linux VM (PR 2).** A declarative Lima template in the repo: Ubuntu 24.04 under systemd, a guest `$HOME` that is not the host's, no host mount of credential directories, no credentials in the image. The guest clones the repo and converges `setup-linux.sh`. Started on demand, deleted after.
- **Tier 3b — Windows 11 ARM VM (PR 3).** A `clean-base` snapshot, reverted before each run, exercising `setup-windows.ps1` and asserting `changed=0` on the second run. It needs a hypervisor (VMware Fusion or UTM); installing one is the owner's step, and this PR does not start before it exists.
- **Tier 2 — ARM runners in CI (deferred, no PR in this spec).** `ubuntu-24.04-arm` in dotfiles and `windows-11-arm` in ts-bridge. Deferred: it changes `.github/workflows/` (owner-lands) and the CI duration ceilings of #2246.

## Out of scope

- Installing a hypervisor, and the Tailscale three-node mesh (ts-bridge QA-013), which rides on Tier 3b.
- Wiring `dotf lint powershell` into the pre-commit hook, and switching CI's `lint-powershell` job to call it. The hook runs the installed `dotf` (lesson 463) and the job is a workflow edit; both wait for the release that carries the command. Ticketed from PR 1.
- Provisioning `pwsh` on Linux machines.

## Risks / open questions

- **Windows CI runs `dotf doctor` after `setup-windows.ps1`, which does not converge modules** (`.github/scripts/doctor-gate.ps1` fails on any unlisted `[FAIL]`). A module FAIL would turn every Windows run red, so the module check warns. *Resolved* by the WARN; making setup converge them is a setup-script change the owner decides.
- **PowerShell Gallery availability.** A Gallery outage fails the `pwsh-modules` apply; the next run retries. Converge never reports success without the import probe.
- **Module name casing.** The marker carries the Gallery name, so the casing is explicit rather than derived from the variable.
- **Credentials in VM images (Tier 3).** Guardrail: no credential file is baked into a template, image or snapshot; secrets reach a guest only through `dotf secrets run`. The Windows VM never mounts the host working tree (CRLF contamination, audit-010); it runs a clone or `git archive` bundle.

## Acceptance criteria

Tier 1 (PR 1):

- [ ] AC1: on the Mac, `dotf converge --only tools,pwsh-modules` plans `pwsh` and the two modules, applies them, and a second plan reports 0 changes.
- [ ] AC2: `versions.conf` pins marked `# pwsh-module: <Name>` parse into (name, version); an unmarked pin is ignored, and a marker not followed by a `NAME_VERSION=` line, or a near-miss marker, is an error naming the line.
- [ ] AC3: the `pwsh-modules` reconciler is skipped where `pwsh` is not on PATH, installs only the modules not importable at their pin, and its probe fails when one is still missing.
- [ ] AC4: `dotf doctor` reports `pwsh` and each marked module at its pin; `pwsh` missing on darwin or windows fails, a module missing warns with the converge remedy, and on linux without `pwsh` the section skips.
- [ ] AC5: `dotf lint powershell` analyses every tracked or untracked-unignored `.ps1`/`.psm1` with `.PSScriptAnalyzerSettings.psd1` at `-Severity Error,Warning`, exits 1 on a finding and 0 on a clean tree, in under 5 s on the Mac.
- [ ] AC6: a test fails when `ci.yml`'s `lint-powershell` job stops using the settings file, severities or file types the command uses.

Tier 3a (PR 2):

- [ ] AC7: the Lima template boots Ubuntu 24.04 with systemd as PID 1, the guest `$HOME` is not a host mount, and no host credential directory is mounted.
- [ ] AC8: inside the guest, a clone of the repo converges `setup-linux.sh`, and a second run reports nothing to change.

Tier 3b (PR 3, after the owner installs a hypervisor):

- [ ] AC9: one command reverts the Windows VM to `clean-base`, runs `setup-windows.ps1` from a clone, and asserts `changed=0` on a second run; no credential file persists in the snapshot.

## References

- Issue #2344; parent epic #2013 (T5, X3); downstream ts-bridge#183, hive#479, hive#484.
- ADR-044 (pinned CLIs through mise), ADR-045 (converge registry), ADR-028 (secrets), ADR-020 (Go layer).
- PLAT-001c (system source type, P5a/P5b); lesson 463 (a hook runs the installed `dotf`).
- Research: `10_projects/knowledge/research/2026-10-11-cross-platform-vm-testing-synthesis.md` (vault).
