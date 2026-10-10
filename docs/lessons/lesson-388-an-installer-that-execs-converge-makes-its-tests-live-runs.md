---
id: "lesson-388-an-installer-that-execs-converge-makes-its-tests-live-runs"
type: lesson
status: active
title: "An installer that execs converge makes its tests live runs"
created: "2026-10-10"
---

# An installer that execs converge makes its tests live runs

## Context
PLAT-001b PR 4b made `install.sh` the one entrypoint: executed, it installs `dotf` and then
`exec`s `dotf converge "$@"`. Before that, the same script only installed the binary, and its bats
suite ran it standalone with the developer's own environment.

## The Trap
One existing test executed the script with no version argument and an unreachable release base,
to prove it read the pin from `versions.conf`. It inherited the developer's `PATH`. On a machine
whose installed `dotf` already matched the pin, the installer took its "already installed" branch,
which succeeds, and the new hand-off then ran the real `dotf converge` against the real `HOME`: an
apply, from a test. Nothing in the test changed. What changed is that a success path of the script
under test now ends in a command that mutates the machine.

The hand-off has a second trap of the same kind. A fresh machine has no `~/.local/bin` on `PATH`,
so `exec dotf converge` would find nothing, or an older `dotf` elsewhere on `PATH`, rather than
the binary the installer had just verified and placed.

## The Solution
- Every test that executes the entrypoint confines `HOME` and `PATH` (`PATH=/usr/bin:/bin` plus
  a stub directory), so the only `dotf` it can reach is a fixture that records its arguments.
- `install_dotf` records the binary it vetted in `DOTF_BIN` (`$script:DotfBin` in PowerShell): the
  `dotf` on `PATH` when it kept it, the file it placed when it installed one. The hand-off execs
  that path, never a `PATH` lookup.
- The hand-off tests assert the full command line the stub received, including the binary's path,
  so a regression to a `PATH` lookup fails them.

## Takeaways
- When a script gains a step that mutates the machine, re-read every test that executes it, not
  only the ones about the new step: each success path now ends in that mutation.
- A test that runs an entrypoint with the caller's `PATH` is testing the caller's machine.
- Hand off to the artifact you verified, by path; a name lookup can resolve to a different one.
