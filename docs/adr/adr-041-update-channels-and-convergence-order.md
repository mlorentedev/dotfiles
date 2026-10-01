---
id: "ADR-041-update-channels-and-convergence-order"
type: adr
status: accepted
owner: manu
date: "2026-09-30"
supersedes: []
extends: [adr-019-self-deploy-fast-forward-only, adr-036-install-channels, adr-038-harness-data-read-from-deployed-records]
issue: mlorentedev/dotfiles#1843
tags: [architecture, decision, release, install, self-deploy, channels]
created: "2026-09-30"
---

# ADR-041: `dotf` has a stable and an edge channel, and a convergence run moves the records before the binary

## Context

A fix merged to `main` reaches a machine only through a release. Measured for #1843:

- The release PR stayed open a median of 44 h in September, or 11.5 h over the last 30 release PRs.
- There were nine releases in September.
- After a release, each machine still needs a manual `./setup-linux.sh`, because the ADR-019 timer is opt-in and was not installed on the machine measured.

Meanwhile, code on `main` and the installed binary disagree, and that has caused real bugs. In #1814, setup called flags that the pinned binary lacked. #1675 and #734 are the same class.

The first draft of #1843 proposed an edge channel and a timer that pulled the binary on its own. An independent review failed it on three grounds:

1. **The binary would move ahead of the records it reads.** ADR-038 D3 requires the opposite.
2. **The Go installer would replace the shell bootstrap.** That contradicts ADR-020 C7 and CLI-090 (#1803).
3. **ADR-019 D3 blocks a new binary.** "Setup runs only when HEAD moved", so a new binary never reaches a machine whose checkout is current.

This ADR sets the rules those three findings require before any of track A is built.

## Decision

1. **Two channels.**
   - `stable` is the default. Its target is the pinned release (`DOTF_VERSION`).
   - `edge` targets the newest build of `main` whose tests and lint passed for that commit.
   - The channel is per-machine state, kept with the other per-machine settings (ADR-025), never in the repository.
   - A machine opts into `edge` explicitly. A raw installer stream with no channel always gets `stable`.
2. **An edge build is never the latest release.** It is published as a GitHub prerelease with `--latest=false`. The raw installer resolves "latest" through GitHub's latest-release API (CLI-090), so it can never return an edge build.
3. **Edge versions stay readable by today's parsers.** Two constraints bind whatever format row A1 of #1843 picks:
   - The version contains an `X.Y.Z` that the installers' `\d+\.\d+\.\d+` regex reads, and that is at or above the stable pin.
   - `dotf version --commit` prints the full commit, so the provenance check in doctor can place the binary against the checkout.
4. **A convergence run has a fixed order:**
   1. Fast-forward the checkout, under ADR-019 D2's rules, which are unchanged.
   2. Mirror the records into the deploy directory.
   3. Move the binary to the channel's target.

   Step 3 never runs unless step 2 succeeded in the same run. A timer or command that moves only the binary is rejected. This is ADR-038 D3 applied to every machine, not only to the release runbook.
5. **ADR-019 D3 is amended.** "Setup runs only when `HEAD` moved" becomes: a convergence run acts when `HEAD` moved, or when the channel's target is newer than the installed binary. When neither holds, the run is a no-op. That includes a `stable` machine whose binary is already above the pin, for example after it leaves `edge`: under decision 6 its target is the installed binary itself.
6. **Pins are floors on both channels** (ADR-036, `decideAction` in `cli/internal/tools/install.go`). An installed build at or above the pin is never downgraded by convergence. The only way to go down is an explicit rollback (decision 8). Doctor is aligned to the same rule (#1262).
7. **The shell bootstrap stays** (ADR-020 C7, CLI-090). `install-dotf.sh` and `install-dotf.ps1` remain the install and recovery path on a machine with no working `dotf`. `dotf self-update` is added on top of them and never replaces them. `dotf update` keeps its name and its behaviour, because the systemd unit, the Windows task and the tests call it.
8. **Nothing automatic before signing and rollback.** Row A5 (the scheduled convergence) ships only after two other rows:
   - A3: the installers verify a signature, not only a checksum.
   - A4: `dotf self-update` verifies the new binary after the swap and restores the previous one if the check fails, and `--version` rolls back on request.

   Until both land, a machine moves to edge only by explicit command.

## Consequences

- A fix merged to `main` can reach an opted-in machine without a release PR and without a manual setup. That is the first acceptance line of #1843.
- The release PR keeps its merge policy (owner decision Q7 on #1843). With an edge channel it becomes a public release train rather than the only way a fix travels.
- `setup-linux.sh` installs `dotf` (the `install_dotf` call) before it mirrors the harness records. A single setup run keeps that window to seconds, but it is the order this ADR rejects. Row B6 of #1843 (`dotf converge`) owns the order from then on, and `setup-linux.sh` is not reordered in place: logic is not added to setup scripts, it moves into `dotf`.
- A machine on `edge` can run a binary newer than its checkout. The doctor provenance check reports that case, and decision 4 keeps the records at least as new as the binary.

## Alternatives rejected

- **A timer that updates only the binary.** It is the inverse of ADR-038 D3: records older than the binary that reads them.
- **Porting the installer to Go.** It needs a working `dotf` to repair a broken `dotf` (ADR-020 C7).
- **goreleaser `--nightly`.** It is a Pro feature. Edge uses `--snapshot` and `gh release upload` instead.
- **Renaming `dotf update` to `dotf sync`.** ADR-021 reserves `dotf sync` for the port of `dotfiles-sync.sh`, which moves data the other way, and the rename would break the scheduled units.

## References

- EPIC #1843 (track A, rows A0 to A5), CLI-090 (#1803), #1262, #1814, #1675, #734.
- ADR-019, ADR-020 (C7), ADR-021, ADR-025, ADR-036, ADR-038.
