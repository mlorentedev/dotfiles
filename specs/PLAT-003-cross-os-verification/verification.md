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
- [ ] AC9 → PR 3.
- [x] AC10 → on the Mac (2026-10-17, released `dotf` 0.74.0 against this worktree with `--repo`): plan `catalog to install: qemu, swtpm` → apply `catalog installed: qemu, swtpm`, `1 changed` (45.7 s) → plan `[ OK ] tools … 39 pinned CLI(s) at their pin`, `0 to change`. The deployed mise config was saved before and diffed after: unchanged (lesson 467). `qemu-system-aarch64` 11.1.2 lists `hvf` and starts a `virt` machine with `-accel hvf`; `swtpm` 0.10.2; `share/qemu/edk2-aarch64-code.fd` present.

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
- **Shared machine state.** The tools apply removed `zsh-autosuggestions` from the deployed mise config, state another session's unmerged branch (#2667) had converged; the file was restored from a copy taken before the apply. Lesson 467; the prune failure it exposed is #2677.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-466-a-substring-parity-assertion-passes-for-a-prefix-of-the-value.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-467-a-converge-from-one-worktree-undoes-what-another-branch-converged.md
- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-473-a-leg-that-puts-the-binary-on-path-first-certifies-a-machine-the-installer-never-makes.md
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the pwsh channel follows ADR-044 and PLAT-001c's system source; nothing structural is new
- [ ] New pattern candidate for `00_meta/patterns/`? no: both lessons are this repo's machinery

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/PLAT-003-cross-os-verification/` -> `specs/archive/PLAT-003-cross-os-verification/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
