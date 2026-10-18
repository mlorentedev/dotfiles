---
tags: [spec, verification, templates]
created: "2026-10-17"
---

# Verification - PLAT-003-cross-os-verification

## Evidence

PR 1 (Tier 1), measured on the macmini (darwin-arm64) on 2026-10-17 with a branch build (`go build -o $TMPDIR/dotf-plat003 ./cmd/dotf`, `--repo` the worktree):

- [x] AC1 → `pwsh-modules` plan `to install from the PowerShell Gallery: PSScriptAnalyzer@1.25.0, Pester@6.2.0` → apply `1 changed` (8.3 s) → plan `[ OK ] 2 module(s) at their pin`, `0 to change`. The `tools` step installed `powershell` through brew (7.6.7) and no longer lists it. Test `TestPwshModules_InstallsWhatIsNotAtItsPin`. The combined form `--only tools,pwsh-modules` was not run as one command and cannot show 0 on this Mac. The `tools` step still plans `mise config to write` for state another branch converged (#2667, lesson 467), the same line the released `dotf` plans against `main`. The evidence for AC1 is therefore per step: `tools` installed and no longer lists `powershell`, and `pwsh-modules` went from plan 2 through apply to plan 0.
- [x] AC2 → `TestParseModules` (8 cases).
- [x] AC3 → `TestPwshModules_*` (6 tests).
- [x] AC4 → `TestCheckPowerShell` (4 cases) + `TestCheckPowerShell_AMalformedMarkerFails`; live: `[PowerShell (local lint-powershell)] (2 checks, all ok)`.
- [x] AC5 → `TestLintFiles_SeesWhatACommitWouldShip`, `TestAnalyzerPin`; live: `Analyzed 26 PowerShell files.` exit 0 in 3.57 s; with an untracked probe calling `Invoke-Expression`: `PSAvoidUsingInvokeExpression Warning`, exit 1.
- [x] AC6 → `TestLintMatchesCIsLintPowershellJob`; mutating `Severity` to `Error`, `SettingsFile` to drop `.psd1`, and `Extensions` to `.ps1` alone each fails it.
- [x] AC7 → `TestLinuxTemplate_SharesNothingWithTheHost`, `TestIsolated` (4 cases), `TestVerify_*` (4 tests). Mutations that each fail the suite: plain mode off, base adds `_default/mounts`, agent forwarded, virtiofs not treated as a host share, the guest never deleted, the token not on stdin, idempotence always passing. Live on the macmini (Lima 2.2.1, vz), `dotf vm linux --ref main`: `isolated: systemd is PID 1, no host directory mounted, HOME=/home/manu.guest`, then the clone. The guest was deleted after the failure below: Lima logged ``Deleted `dotf-linux-1792229031` ``, and `limactl list` afterwards printed `No instance found.`
- [ ] AC8 → **not met; blocked on #2688.** The live run's first converge failed: `[FAIL] records-harness … compile-harness.sh --deploy: exit status 2`, `instructions need dotf harness instructions; dotf is absent or predates it`, `2 changed, 0 ok, 0 opaque, 12 skipped, 1 failed`. `install.sh` execs converge by absolute path and never puts `~/.local/bin` on PATH, which `from-zero.yml` and `setup-linux.sh` both do before anything runs (lesson 473). `Converged` fired on a real failure. `Idempotent` is unit-tested only, because the run never reached a second converge. A probe in the same template showed that a later `limactl shell` session has `~/.local/bin` on PATH once the directory exists, so fixing `install.sh` (#2688 option A) should be enough. AC8 is re-run with `dotf vm linux --ref <fix>` once the fix is on a ref. The guest converges with the `dotf` release the ref pins, so a Go-side fix also needs a release.
  - #2688 fixed with option (B), the owner's pick (2026-10-17): `dotf converge` puts its own directory first on PATH when PATH does not resolve `dotf` to the running binary. Measured on msi (Ubuntu, linux/amd64) with msi's checkout, `env PATH=/usr/bin:/bin <dotf> converge --only records-harness`: the installed 0.69.0 -> `[FAIL] records-harness … dotf is absent or predates it`, exit 1; the branch tree without the fix -> the same failure, exit 1; the branch with it -> `[ OK ] records-harness`, `0 changed, 1 ok, … 0 failed`, exit 0, twice. The binaries sat in `/tmp/<dir>/dotf`, never over `~/.local/bin/dotf` (lesson 475). AC8 itself waits for the release that carries the fix, since the guest converges with the pinned release.
- [ ] AC9 → **not met; blocked on #2754 (BUG-156).** PR 3 on the Mac (QEMU 11.1.2, HVF), 2026-10-18. Unit side: `TestBuild_*`, `TestRun_*`, `TestStop_*`, `TestUnattend*`, `TestArchiveRef_TakesTheCommittedTreeOnly`; mutations of the run path (6 of 6) and of the build's running-guest refusal each fail the suite. Live:
  - **Build.** The first live build finished `exit=0`, but its base was finished by hand: the guest agent MSI rolled back on Arm64 (lesson 484) and was installed through the QMP keyboard channel, and UAC was set by hand. Both are now in `firstlogon.ps1`. REBUILD_PENDING
  - **Run, on that hand-finished base.** Agent, autologon, archive copy, the scheduled task as `dotf`, and `install.ps1` from zero all worked: it installed `dotf` 0.74.0 and converge wrote its report. The first converge failed in `tools`: Docker Desktop and the AWS CLI waited on UAC prompts nobody could answer, fixed with UAC auto-consent in the guest (ADR-053). The rerun installed both, then failed the `tools` post-condition: `installed by this machine's tools but not on PATH: uv (for poetry), uv (for pre-commit)`. On Windows uv comes only from `legacy-setup`, converge's last step, which a failure never reaches: #2754. 13 later steps were `not reached`. FINAL_RUN_PENDING
  - **BUG-155 (#2734)** measured: with no git on a clean Windows, the checkout step fails `git is not on PATH … cannot be cloned` (comment on #2734).
  - **#2672 and #2683 in the guest** (evidence on both PRs and #2356): unelevated `--plan` fails reading the firewall; elevated `--plan` fails on a lidless machine (#2764, `/q` vs `/qh`); the `Include` form dotf demands is one Win32-OpenSSH 9.5p2 ignores, which `sshd -t` cannot see (#2763, lesson 485); `sshd_config` is absent until sshd's first start (#2758); the apply was refused for want of a Tailscale adapter and rolled back cleanly, with a detail line that claimed unrun work (#2765). Fixtures: hand-finished base, a mesh-range address added next to the DHCP lease, no Tailscale adapter by design, OpenSSH installed and `sshd_config` edited by hand, and a measurement build of main `9f664d70` with `/q` changed to `/qh`.
  - The zip the run copies in is a `git archive`, not a clone: the guest has no `.git`, so `git-hooks` and `git-config` meet a deployed copy where the Linux tier meets a checkout, and converge says so (`DOTFILES_REPO_DIR … has no .git entry … reading the deployed copy instead`).
- [x] AC10 → on the Mac (2026-10-17, released `dotf` 0.74.0 against this worktree with `--repo`): plan `catalog to install: qemu, swtpm` → apply `catalog installed: qemu, swtpm`, `1 changed` (45.7 s) → plan `[ OK ] tools … 39 pinned CLI(s) at their pin`, `0 to change`. The deployed mise config was saved before and diffed after: unchanged (lesson 467). `qemu-system-aarch64` 11.1.2 lists `hvf` and starts a `virt` machine with `-accel hvf`; `share/qemu/edk2-aarch64-code.fd` present. `swtpm` 0.10.2 was declared and converged in the same run, then removed from the catalog: the builder's argv with `-device tpm-tis-device` exits `HV_BAD_ARGUMENT` under HVF, and the same argv without it boots (QMP `query-status`: `running`). A system entry is never uninstalled, so the Mac keeps the `swtpm` formula until someone removes it.

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...` → OK; `go test ./...` → 44 packages ok, exit 0.
- `golangci-lint run ./...` (2.12.2, the pin) → `0 issues.`
- bats reading `versions.conf` / `packages.json` (ci-pester-pin, ci-pssa-single-owner, packages-json, versions-conf, versions-no-hardcode, version-gte, bashrc-guards, zshrc-guards, setup-no-arch-downloads, check-doc-paths) → 98 ok, exit 0.
- bats that use pwsh, now that the Mac has one (bats-pwsh-gate-real, bats-pwsh-gate, ci-windows-doctor-gate, env-contract, obs-cli-ps1, powershell-profile, profile-heal-ps1, setup-windows, windows-defaults) → 207 ok, exit 0.

## Decisions made during implementation

- **pwsh through a `system` catalog entry, not mise.** mise's `powershell` resolves to `aqua:PowerShell/PowerShell`, but its binary is `pwsh`: the sync probes `mise which <name>`, so a marker would need a name→bin row in a released `dotf` first. CI's pwsh is the runner image's, unpinned too.
- **The module check warns, never fails.** CI's Windows doctor gate runs after `setup-windows.ps1`, which converges no module, and fails on any unlisted `[FAIL]`.
- **The lint reads the files a commit would ship** (`git ls-files --cached --others --exclude-standard`, present on disk) rather than CI's recursive walk, which on a working tree would also read ignored files.
- **The VM certifies the pinned release, not the tree.** The guest runs `./install.sh` from the clone, which installs the `DOTF_VERSION` release; `from-zero.yml` is the leg that tests the tree's binary. A clean guest is what a person's `curl | bash` meets, which is why it found #2688.
- **No guest-side script.** Every guest command is a `limactl shell` call from Go with plain arguments; the one shell string is the constant stdin-token wrapper.
- **The guest agent is extracted, not installed.** `msiexec /i` of virtio-win's `qemu-ga` exits 1603 on Arm64 because its VSS COM+ registration fails and rolls everything back; `msiexec /a` runs no custom action, and `New-Service` creates the service with the MSI's own command line (lesson 484).
- **A step reaches the user through a scheduled task** (`/ru dotf /it`), because the agent runs as LocalSystem. It runs in the autologon session and needs no password.
- **UAC auto-consents in the guest only** (ADR-053): the guest holds no credentials and every run is a throwaway overlay. A converge that needs elevation therefore passes in the guest where a real machine would stop at a prompt.
- **Adding an address to a DHCP adapter turns DHCP off** (`New-NetIPAddress`): the guest lost its lease and `Add-WindowsCapability` hung without a network. `netsh interface ipv4 set interface <alias> dhcpstaticipcoexistence=enabled` first keeps both.
- **Shared machine state.** The tools apply removed `zsh-autosuggestions` from the deployed mise config, state another session's unmerged branch (#2667) had converged; the file was restored from a copy taken before the apply. Lesson 467; the prune failure it exposed is #2677.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-466-a-substring-parity-assertion-passes-for-a-prefix-of-the-value.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-467-a-converge-from-one-worktree-undoes-what-another-branch-converged.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-473-a-leg-that-puts-the-binary-on-path-first-certifies-a-machine-the-installer-never-makes.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-481-a-vendor-gate-that-rejects-automation-is-an-input-boundary.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-484-an-installer-s-optional-part-can-roll-back-the-part-you-need.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-485-a-config-check-that-validates-syntax-cannot-see-an-ignored-include.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-053-the-windows-test-guest-runs-on-qemu-driven-by-dotf.md (Tier 3b's hypervisor and driver). The pwsh channel follows ADR-044 and PLAT-001c's system source
- [ ] New pattern candidate for `00_meta/patterns/`? no: both lessons are this repo's machinery

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-003-cross-os-verification/` -> `specs/archive/PLAT-003-cross-os-verification/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
