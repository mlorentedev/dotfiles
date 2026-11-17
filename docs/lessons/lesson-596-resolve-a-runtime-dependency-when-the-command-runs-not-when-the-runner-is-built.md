---
id: "lesson-596-resolve-a-runtime-dependency-when-the-command-runs-not-when-the-runner-is-built"
type: lesson
status: active
title: "Resolve a runtime dependency when the command runs, not when the runner is built"
created: "2026-11-12"
---

# Resolve a runtime dependency when the command runs, not when the runner is built

## Context

On a clean Windows 11 Arm64 guest, `dotf converge` installed bats through mise's npm backend and then
reported it "not running at its pin" with a bare error (#3328, #2346). bats on Windows is a `bats.cmd`
launcher that runs `bash.exe` from the PATH.

## Problem

Without Git Bash first on the PATH, `bats.cmd` picks `C:\Windows\System32\bash.exe`, WSL's launcher, and
fails. Run by hand in the guest, `bats.cmd` printed `Bats 1.13.0` once `C:\Program Files\Git\bin` was first.
The runner already prepended Git Bash for bats, but it looked Git Bash up once, when the runner was built.
A converge builds its runners before its catalog installs git, so on a clean box the lookup answered "not
found" and kept answering it for the whole tools step. The failure note ran later, found Git Bash, and said
nothing, which is why the symptom had no explanation. A box that already had git never showed it.

## Solution

Ask for the dependency at each command that needs it (`homeCommand` calls `locateBash` per bats command),
and test the ordering that bites: a locator that answers "absent" and later "present" must change what the
next command sees. Any runner built before a step that installs what it depends on has the same shape.

## References

- #3328, #2346, `cli/internal/tools/mise.go` (`homeRunners`).
