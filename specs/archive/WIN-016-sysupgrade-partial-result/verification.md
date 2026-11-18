---
tags: [spec, verification, templates]
created: "2026-11-17"
id: "WIN-016-sysupgrade-partial-result"
type: spec-verification
status: active
owner: manu
---

# Verification - WIN-016-sysupgrade-partial-result

## Evidence

- [x] AC1 -> `TestSystemUpgradeWindowsKeepsWhatItDidWhenTheConfirmingListingFails`. Before the change: `err = winget upgrade: exit status 1: boom, want the failed confirmation named`, with an empty result.
- [x] AC2 -> `TestSystemUpgradeWindowsProbeDoesNotListWithoutElevation`. Before: `Probe = winget upgrade: exit status 1: transient, want nil`, and three listings in the log.
- [x] AC3 -> `TestSystemUpgradeWindowsDoesNotCallTheCatalogUpToDateWhenEveryIdWaits`. Before: `catalog entries up to date; left, needs elevation: GitHub.cli (machine-wide install)`, for plan and apply.
- [x] AC4 -> `TestSystemUpgradeWindowsWithoutElevationReadsNoMachineWideInstallAsNone`. It passed before the change; the behaviour had no test. With the fake answering `0x8A15002B` instead it fails: `left, needs elevation: Git.Git (install scope unknown: winget list: exit status 2316632107)`.

## Test status

- `cd cli && go test ./internal/converge/ -count=1` -> `ok ... 232.818s` (Windows 11 x64, 2026-11-17).
- `go test ./internal/cmd/ -count=1 -run 'Update|SystemUpgrade|Converge'` -> `ok ... 65.615s`: the commands that read the step's detail.
- `go vet ./internal/converge/` for Windows and for Linux: clean.
- Not run on a real winget: the cases need a listing that fails between two runs. The test fake answers winget's commands as the existing Windows tests of this step do.

## Decisions made during implementation

- The ids of a failed confirmation count as changes. Counting them as zero would drop the step's line (`runStep`, BUG-136), which is the loss the finding is about.
- The probe returns before the listing instead of ignoring its error: the answer was never used, so there was nothing to tolerate.
- The wording is fixed in the shared summary, so a macOS run that leaves a cask waiting for sudo stops saying `up to date` too. No test asserted the old text with something waiting (`grep -rn "catalog entries up to date"`: one fixture in `update_skip_test.go`, with nothing waiting); the darwin case is now in the AC3 test.

## Review dispositions

Independent review: `review.md`, PASS WITH GAPS, `agy/gemini-4-hydrogen-high` on `382ad32a`.

- Minor, REAL, archive-gate (the promotion lines were a plain bullet): applied below.
- Minor, THEORETICAL, test-coverage (no dedicated case for a macOS dry run with a cask waiting for sudo, nor for a failed id in `waiting` when the confirmation fails too): declined. Both go through `upgradeSummary` and `unconfirmed`, whose branches the four tests cover and the reviewer's mutation probes confirmed; `dryRun` only changes the verb, and a failed id reaches `waiting` on a path the confirmation does not touch.
- Minor, SPECULATIVE, defensive-parsing (`splitMachineWide` parses the output on `0x8A150014` instead of returning early): declined. winget's "no installed package found" text holds no package id, so the parse returns none, which `TestSystemUpgradeWindowsWithoutElevationReadsNoMachineWideInstallAsNone` holds.
- Not run by the reviewer either: a real winget failing between two listings.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`.

- [x] Lesson for the repo's `docs/lessons/`? no: the finding is specific to this probe, and the spec and its four tests record it.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no decision changes; the step keeps its contract.
- [x] New pattern candidate for `00_meta/patterns/`? no: "a post-condition does not take a reading it will ignore" has one instance.

## Archive checklist

- [x] Independent adversarial review recorded in `review.md`
- [x] Issue #3573 closed by the archiving change
