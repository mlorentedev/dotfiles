---
tags: [spec, tasks]
created: "2026-11-12"
---

# Tasks - CLI-133-update-self-install

> TDD order. Run every command from `cli/`. Each behaviour is observed failing
> before production code changes. `[AC<n>]` maps to `proposal.md`. Two PRs:
> PR 1 is the Go self-install primitive in `internal/tools`; PR 2 rewires
> `dotf update` onto it, after #3428 (BUG-172) has landed.

## Setup

- [x] Branch `feat/3433-update-self-install` from `origin/main` (worktree)
- [x] Baseline: `go test ./internal/tools/ ./internal/update/ ./internal/cmd/ -count=1` passes

## Implementation — PR 1: the primitive

- [x] [P] [AC4] In `internal/tools/place_test.go`, add
  `TestPlaceBinaryReplacesARunningExecutable`: copy the test binary to a temp
  `dotf` (`dotf.exe`), start it as a helper child that blocks on stdin, call
  `placeBinary(new, thatPath)` while it runs, then assert the path holds the new
  bytes and no `.old` is left after the child exits and a second placement runs.
  Add `TestPlaceBinaryRestoresTheOldFileWhenTheFinalRenameFails` through a
  `rename` seam. Run `go test ./internal/tools/ -run TestPlaceBinary -count=1`.
  Expected: FAIL on Windows for the first (access denied on rename-over), and
  everywhere for the second (`undefined: renameFile`).
- [x] [AC4] In `internal/tools/install.go`, make `placeBinary` replace through
  rename-aside: remove a stale `<dst>.old`, rename `dst` to `<dst>.old`, rename
  the staged file in, undo the first rename when the second fails, and remove
  `<dst>.old` best-effort. No `GOOS` branch. Rerun. Expected: PASS.
- [x] [P] [AC3] In `internal/tools/self_test.go`, add `TestDotfRelease` (asset,
  member and checksums names per OS and arch, matching `install.sh` and
  `install.ps1`) and `TestInstallSelf` against an `httptest` release server:
  installs into the given path and returns it; a checksum mismatch, an asset
  missing from `checksums.txt`, and a staged binary whose `--version` fails each
  return an error and leave the target byte-identical; a target whose file name
  is not `dotf` (`dotf.exe`) is refused. Run
  `go test ./internal/tools/ -run 'TestDotfRelease|TestInstallSelf' -count=1`.
  Expected: FAIL, `undefined: DotfRelease`.
- [x] [AC3] Add `internal/tools/self.go` with `DotfRelease(version) Tool` and
  `(*Installer).InstallSelf(t Tool, target string) error`, which sets `Dest` to
  the target's directory and runs `fetchVerifyPlace` unconditionally (a pin below
  the running version installs: the pin is the contract). Rerun. Expected: PASS.

## Implementation — PR 2: `dotf update` onto it

- [x] Merge `origin/main` once #3428 has landed (merged, not rebased: no force-push). #3428 landed while PR 1 was in review, so both halves ship as one PR, #3441.
- [x] [P] [AC2] In `internal/update/update_test.go`, add
  `TestRun_PinComesFromTheUpstream` (Git fake answers `show @{u}:versions.conf`)
  and `TestRun_PinFallsBackToTheCheckout` (show fails: local pin used, message
  says so). Expected: FAIL.
- [x] [AC2] Split `doctor.ParseVersionsConf` into `ParseVersions(io.Reader)` and
  the path wrapper. In `update`, read the pin from `@{u}:versions.conf` (fetching
  first on a dirty checkout, which `Assess` leaves unfetched), else `Deps.Pin`.
  Rerun. Expected: PASS.
- [x] [P] [AC1] Add `TestRun_InstallsThePinOnADriftedCheckout` (table: dirty,
  ahead, diverged): install called with the upstream pin, converge not called,
  status unchanged, message names the install and the skipped converge, nil
  error. Add `TestRun_OfflineInstallsNothing`. Expected: FAIL.
- [x] [P] [AC5] Change `Deps.Install` to return the installed path and add
  `Deps.ConvergeWith(exe string) error`; add
  `TestRun_ConvergesWithTheInstalledBinary` and
  `TestRun_ConvergeFailureAfterTheInstall` (status `converge-failed`, non-nil
  error). Expected: FAIL.
- [x] [AC1] [AC5] Rework `update.Run`: one deployable/drift classification, the
  install for every checkout state but `offline` and `not-a-repo`, the converge
  only when deployable. Remove `skipDrift`'s hint and the BUG-172
  `InstallerConverge` seam. Keep `Run` under 40 lines (closes #3427). Rerun.
  Expected: PASS.
- [x] [P] [AC6] Rewrite `internal/cmd/update_pin_test.go` on a fake release:
  an `httptest` server serving the test binary as `dotf` in a `.tar.gz` (`.zip`
  on Windows), the test binary acting as the fake dotf through a helper env var
  (`--version` prints the pin, `converge` records its args and exits with a
  given code). Cover: installs and converges on a current checkout; installs and
  skips the converge on a diverged one; a failed converge is `converge-failed`;
  a bad checksum is `install-failed` and leaves the target untouched. No
  `runtime.GOOS` skip. Expected: FAIL.
- [x] [AC6] In `internal/cmd/update.go`, wire `Install` to
  `tools.Installer.InstallSelf(tools.DotfRelease(pin), <resolved executable>)`
  and `ConvergeWith` to `exec.Command(exe, "converge")` with
  `DOTFILES_REPO_DIR` set; delete `runInstaller`, `installerInvocation` and
  `failedConvergeSince`. Package-level seams for the executable path and the
  release base URL. Rerun. Expected: PASS on Linux, macOS and the Windows leg.

## Closing

- [ ] `go build ./... && go vet ./... && go test ./...`; `GOOS=windows go vet ./...`; `golangci-lint run` and `GOOS=windows golangci-lint run ./...`
- [ ] Runbook `docs/runbooks/guide-self-deploy-timer.md`: the flow, the binary installed on every checkout state, `install-failed` vs `converge-failed`; `dotf update --help` text
- [ ] `verification.md` filled; independent review (`dotf spec review`); archive
