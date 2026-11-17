---
id: "CLI-133-update-self-install"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3433"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CLI-133-update-self-install

> **Naming**: file lives at `<repo>/specs/CLI-133-update-self-install/proposal.md`. `CLI-133-update-self-install` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3433: dotf update cannot update dotf itself unless the checkout fast-forwards -->

`dotf update` is the command that updates the tool, and it installs the pinned `dotf` only when the dotfiles checkout is clean and fast-forwards: one stray local commit (diverged), an unpushed one (ahead) or an edited file (dirty) and it installs nothing, forever, and tells the user to run `./install.sh`. The pin it reads is the local checkout's, and the install shells out to `install.sh` or `pwsh install.ps1`, two per-OS scripts, so the CLI depends on setup scripts to replace itself. Measured on the owner's Windows box on 2026-11-12: `dotf update` answered "local branch has diverged from origin/main — skipping self-update" and the machine stayed on an old release.

## What

`dotf update` updates the binary from any checkout state, with one Go code path on every OS:

1. **The pin is upstream's.** After the fetch, the pin is `DOTF_VERSION` in `git show @{u}:versions.conf`. A checkout with no upstream, or whose upstream `versions.conf` cannot be read, falls back to its local `versions.conf`, today's behaviour, and the run says so.
2. **The install is Go.** When the running release is not the pin, `dotf` downloads `dotf_<pin>_<os>_<arch>.tar.gz` (`.zip` on Windows) from the release, verifies its sha256 against the release `checksums.txt`, extracts it, runs the staged binary's `--version`, and replaces the file it is running from (`os.Executable()`). The replacement renames the old file aside before renaming the new one in, which is the one sequence Windows allows for a running executable; every OS runs it. This reuses `tools.Installer`'s fetch, verify, extract, probe and place pipeline.
3. **The converge is the new binary's, and only from a deployable checkout.** On a checkout that fast-forwarded or was current, the installed binary converges as a child process (`<exe> converge`); Windows has no `exec(2)`, so no OS uses it. On dirty, ahead, diverged, no-upstream and ff-failed, the binary is still installed and the converge is skipped with the reason. An offline run installs nothing (the download needs the network the fetch just lacked) and stays a benign skip.
4. **`install.sh` and `install.ps1` are bootstrap-only.** `dotf update` stops calling them.

Status tags do not change. A drift skip keeps its tag (`dirty`, `ahead`, `diverged`), so doctor still warns that the checkout is stuck, and its message names the install. A failed install is `install-failed`, now on every checkout state. A converge failure after the install is `converge-failed`, which makes the BUG-172 report reader (`InstallerConverge`, reading `converge/last.json`) unnecessary: install and converge are two Go steps whose errors are already distinct.

## Out of scope

- A machine with no checkout (`not-a-repo`): there is no pin to read. It keeps skipping; `install.sh` / `install.ps1` remain the bootstrap.
- Adding `dotf` to the `packages.json` catalog so `dotf converge` reconciles it. The self-install is a Go-built `tools.Tool`, not a catalog entry.
- Release signing beyond the sha256 check the installers do today.

## Risks / open questions

- **Replacing the running binary on Windows.** Resolved: rename `dotf.exe` to `dotf.exe.old`, rename the staged file in, and remove `.old` best-effort; a leftover `.old` is removed before the next replacement. If the second rename fails, the first is undone, so the old binary is never lost.
- **Where the binary lives.** Resolved: the directory of `os.Executable()` after `filepath.EvalSymlinks`. A release binary whose file name is not `dotf` (`dotf.exe`) is refused rather than installing beside it.
- **A source build** (`dev`, a pseudo-version) is never replaced, as today.
- **A pin below the running version** (a rollback) is installed: the pin is the contract, as in the installers.
- **Sequencing with #3428** (BUG-172). It lands first as-is; the second PR of this spec removes the `InstallerConverge` seam it adds.

## Acceptance criteria

- [ ] AC1: On a diverged, ahead or dirty checkout whose running release is not upstream's pin, `dotf update` installs the pin, skips the converge with the reason, keeps the drift status, and exits 0. An offline run installs nothing.
- [ ] AC2: The pin is read from `@{u}:versions.conf` after the fetch; when the upstream cannot be read, the local `versions.conf` is used and the message says so.
- [ ] AC3: A checksum mismatch, a missing checksum entry or a staged binary that does not run installs nothing and leaves the running binary byte-identical; the run is `install-failed` with a non-zero exit.
- [ ] AC4: Replacing a binary whose path is held open succeeds through rename-aside on every OS, leaves no `.old` behind when it can be removed, and restores the old binary when the final rename fails.
- [ ] AC5: On a fast-forwarded or current checkout, the installed binary converges as a child process; its failure is `converge-failed` with a non-zero exit.
- [ ] AC6: `dotf update` no longer invokes `install.sh` or `install.ps1`, and its pin-path tests run on the Windows CI leg instead of being skipped.

## References

- Issue: mlorentedev/dotfiles#3433. Prior: #3395 / BUG-172 (PR #3428), BUG-144 (the pin as part of "up to date").
- `cli/internal/tools/install.go` (`fetchVerifyPlace`, `placeBinary`), `cli/internal/update/update.go`, `cli/internal/cmd/update.go`.
- Runbook: `docs/runbooks/guide-self-deploy-timer.md`. ADR-019 (self-deploy is fast-forward only: unchanged, it governs the checkout, not the binary).
