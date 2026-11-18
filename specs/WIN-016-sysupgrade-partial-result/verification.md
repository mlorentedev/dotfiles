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
- [x] AC4 -> `TestSystemUpgradeWindowsWithoutElevationReadsNoMachineWideInstallAsNone`. It passed before the change; the behaviour had no test.

## Test status

- `cd cli && go test ./internal/converge/ -count=1` -> `ok ... 232.818s` (Windows 11 x64, 2026-11-17).
- `go test ./internal/cmd/ -count=1 -run 'Update|SystemUpgrade|Converge'` -> `ok ... 65.615s`: the commands that read the step's detail.
- `go vet ./internal/converge/` for Windows and for Linux: clean.
- Not run on a real winget: the cases need a listing that fails between two runs. The test fake answers winget's commands as the existing Windows tests of this step do.

## Decisions made during implementation

- The ids of a failed confirmation count as changes. Counting them as zero would drop the step's line (`runStep`, BUG-136), which is the loss the finding is about.
- The probe returns before the listing instead of ignoring its error: the answer was never used, so there was nothing to tolerate.
- The wording is fixed in the shared summary, so a macOS run that leaves a cask waiting for sudo stops saying `up to date` too. No darwin test asserted the old text.

## Promotion candidates

- None. The rule "a post-condition does not take a reading it will ignore" is specific to this probe.

## Archive checklist

- [ ] Independent adversarial review recorded in `review.md`
- [ ] Issue #3573 closed by the archiving change
