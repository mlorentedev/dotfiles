---
id: "lesson-609-a-failure-the-account-cannot-fix-is-a-report-not-an-error"
type: lesson
status: active
title: "A failure the account cannot fix is a report, not an error"
created: "2026-11-16"
---

# A failure the account cannot fix is a report, not an error

## Context

`system-upgrade` upgrades the catalog's outdated system packages on each converge. On Windows it ran one
`winget upgrade --id` per package and joined every failure into the step's error (TOOL-036 AC5). The converge
runner is fail-fast: the steps after a failed one are not reached.

## Problem

On a workstation whose account cannot elevate, three machine-wide packages failed on every run
(`0x8a15010c`, `0x8a15008e`, `0x8a150006`). Nothing the run could do would change that, so the step failed every
day, 18 unrelated steps were never reached, and `dotf update` recorded `install-failed`. The error was true and
useless: it reported a condition of the machine as a defect of the run, and paid for it with the rest of the
converge. Linux and macOS had already drawn this line (apt without `sudo -n` skips, a `needs_sudo` cask waits);
Windows had no notion of privilege at all, because `os.Geteuid` answers -1 there.

Tolerating the failure was not enough. The first fix left a failed id waiting and exited 0, and its run on the
real machine raised an administrator prompt for each machine-wide package: the attempt is itself the harm.

## Solution

Ask whether the process holds the privilege, not what the tool's exit code means, and ask before acting. Not
elevated: `winget list --scope machine`, which only reads, says which ids are installed machine-wide, and those
are never attempted; an id that is attempted and fails is left waiting with winget's status; the step exits 0.
Elevated: it attempts everything and fails as before. The rule is the token because the probe keeps no state between calls and must reach the answer the
reconcile reached. What keeps this from being a silent swallow is the record: a quiet day's `dotf update` reads
the marker and records `upgrade-skipped` with the ids (#3553).
