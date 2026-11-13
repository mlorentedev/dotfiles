---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "CLI-133-update-self-install"
type: spec-verification
status: active
owner: manu
---

# Verification - CLI-133-update-self-install

## Evidence

- [x] AC1 (diverged, ahead, dirty and ff-failed checkouts install the pin, skip the converge, keep the status, exit 0; offline installs nothing): `TestRun_InstallsThePinOnADriftedCheckout` (four rows), `TestRun_OfflineInstallsNothing` (offline, dirty and offline, not a repo), and end to end `TestUpdate_InstallsThePinOnADivergedCheckout`, which builds a real diverged clone, serves a release over `httptest`, and asserts the running file was replaced, nothing converged, and the record says `diverged`.
- [x] AC2 (upstream's pin, local fallback said so): `TestRun_PinSource` (upstream wins over the checkout; unreadable upstream and no upstream fall back with "read from the checkout"). `TestUpdate_InstallsThePinAndConvergesWithIt` reads the pin through the real `git show @{u}:versions.conf` while the environment exports a stale `DOTF_VERSION=0.0.1`.
- [x] AC3 (checksum mismatch, missing entry or a staged binary that does not run installs nothing): `TestInstallSelf_RefusalsLeaveTheTargetUntouched` (five rows: mismatch, not listed, probe fails, probe reports another version, target not named dotf; each asserts the target byte-identical and nothing left beside it). End to end: `TestUpdate_ABadChecksumInstallsNothing` records `install-failed`, a non-zero exit, and an untouched running file.
- [x] AC4 (rename-aside replacement of a running binary, `.old` cleaned, restore on failure): `TestPlaceBinaryReplacesARunningExecutable` runs a copy of the test binary and replaces it while it runs, replaces it a second time while that copy still runs from `<dst>.old`, then checks a later placement leaves no `.old*`; `TestPlaceBinarySetsAsideAgainWhileAnOldCopyIsHeld` makes the held `.old` fail to delete on every OS (a `removeFile` seam) and requires the update to succeed; `TestPlaceBinaryRestoresTheOldFileWhenTheFinalRenameFails` injects a failed final rename and asserts the old bytes are back and no `.old` remains.
- [x] AC5 (the installed binary converges as a child; its failure is `converge-failed`): `TestRunPin` rows "ff, running behind the pin" (the converge is `ConvergeWith(<installed path>)`, the in-process one never runs) and "ff, the new binary's converge fails". End to end, `TestUpdate_InstallsThePinAndConvergesWithIt` asserts the installed fake dotf received `converge` with `DOTFILES_REPO_DIR` set to the checkout, and `TestUpdate_TheNewBinarysConvergeFailing` records `converge-failed`.
- [x] AC6 (no installer script; pin-path tests on every OS): `runInstaller` and `installerInvocation` are deleted; the command tests serve the test binary itself as the release (a `.zip` with `dotf.exe` on Windows) through `TestMain`, with no `runtime.GOOS` skip.

## Test status

- `cd cli && go build ./... && go vet ./... && go test ./...`: every package `ok`.
- `GOOS=windows go vet ./...`: ok. `golangci-lint run ./...` and `GOOS=windows golangci-lint run ./...`: `0 issues.`
- Mutation: disabling the drift branch in `installPin` (`if false && !deployable`) fails `TestRun_InstallsThePinOnADriftedCheckout` and `TestUpdate_InstallsThePinOnADivergedCheckout`; restored.
- Manual smoke: not run against a real release. The asset and checksum names are pinned by `TestDotfRelease` to what `install.sh` and `install.ps1` download; the first real exercise is the next release on each OS, and the Windows rename-aside is proved by the `test (windows-latest)` leg.
- No regressions: the full suite passes.

## Decisions made during implementation

- **One PR, not two.** The plan split the primitive from the rewire so the rewire could wait for #3428. #3428 landed while the first half was in review, so the second half went onto the same branch (#3441) by merging `origin/main`, never by rebasing a pushed branch.
- **Status tags unchanged.** A drifted checkout that got the pin keeps its own tag (`dirty`, `ahead`, `diverged`, `ff-failed`), because doctor's stuck-status warning is about the checkout and stays true; the message names the install. `install-failed` can now come from any fetched checkout. No new tag, so doctor and `record.go` are untouched.
- **Offline installs nothing.** A machine whose fetch failed would fail the download too; an offline night stays a benign skip instead of becoming `install-failed`. A dirty checkout is fetched by `Run` itself (Assess stops before fetching it), and a failed fetch there is treated the same way.
- **The BUG-172 report reader is gone.** With the install and the converge as two Go steps, their errors are distinct, so `InstallerConverge`, `failedConvergeSince` and their tests are deleted. This also retires the two CodeRabbit findings deferred from #3428 (a same-second stale report; the runbook overstating `install-failed`).
- **Release URL seam is a package variable, not `DOTF_RELEASE_BASE`.** The installers' variable means `<host>/releases/download`; `tools.Installer.BaseURL` means the host before `<owner>/<repo>`. Reusing the name with another shape would be a second contract; tests set the variable instead.
- **The pin is parsed in the command layer.** `update` cannot import `doctor` (doctor imports update), so `Deps.Pin(ref)` takes the ref and `cmd` reads `git show <ref>:versions.conf` through `doctor.ParseVersions`, split out of `ParseVersionsConf`.
- **A held `.old` is not an error** (advisor, before review). On Windows the previous copy is often still running (a `dotf secrets run` wrapper), so `<dst>.old` cannot be removed on the next update. The first draft returned an error there, which would have failed every second update on Windows while Unix, where unlinking a running binary is allowed, stayed green. Now the current binary goes aside under a unique `<dst>.old.<n>` and every `<dst>.old*` is swept best-effort. Mutation: restoring the error return fails `TestPlaceBinarySetsAsideAgainWhileAnOldCopyIsHeld` on macOS. Lesson 595.
- **`update.Run` is 23 lines** after the rework, closing #3427.

## Review findings (review.md, PASS at d738f8bb)

- **Minor, POSIX window between the two renames** — declined. One replacement path on every OS is the point (AC4, agnostic CLI): a Unix-only `os.Rename(staged, dst)` branch would leave the Windows path exercised by one CI leg only. The window is two consecutive renames in one directory; a concurrent `dotf` started inside it fails to start and is rerun, it never runs a partial binary.
- **Minor, a held `<dst>.old` aborts the next update on Windows** — applied in `86802fc6`, after the reviewed commit: unique aside name plus a best-effort sweep, `TestPlaceBinarySetsAsideAgainWhileAnOldCopyIsHeld` (decision above, lesson 595).
- **Minor, `convergeWith` does not pass stdin** — applied: the child converge now inherits `c.InOrStdin()`, so an interactive `dotf update` behaves like an interactive `dotf converge`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-595-a-binary-that-replaces-itself-must-survive-its-own-previous-copy.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-019 (fast-forward only) still governs the checkout; installing the pinned release is the existing DOTF_VERSION contract executed in Go, recorded in the spec and the runbook
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: self-replacement of a running binary occurs only in dotf

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-133-update-self-install/` -> `specs/archive/CLI-133-update-self-install/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018): #3433 closed by the merge of #3441 on 2026-11-13
- [x] Promotions above executed (if any)
