---
tags: [spec, verification, templates]
created: "2026-11-17"
id: "WIN-015-doctor-batch-launcher"
type: spec-verification
status: active
owner: manu
---

# Verification - WIN-015-doctor-batch-launcher

## Evidence

- [x] AC1, AC2, AC3, AC4 -> `TestCheckAgentLaunchers_ReportsWhatALaunchStarts`: a native launcher, a batch file with a real `.exe` behind it, a batch file with npm's extensionless stub behind it, a `.bat`, and an agent not on PATH.
- [x] AC4 -> `TestCheckAgentLaunchers_CoversTheLaunchTable`: a line for every name of `lane.Agents()`.
- [x] AC5 -> `TestCheckAgentLaunchers_SkipsOffWindows`: linux and darwin; the lookups fail the test if called.
- [x] AC6 -> `TestCheckAgentLaunchers_AsksOnlyTheSeam`: the seam is asked about the two batch launchers and nothing else. `launcherSys` fails the test on any `CommandOutput`, which does not cover `lane.MiseNative` (it starts mise by itself); the seam test is what does. `TestCheckAgentLaunchers_WithoutAToolManagerAnswer` runs with a nil seam.
- [x] Both launch tables -> `TestCheckAgentLaunchers_CoversTheReviewRunners` parses the `switch e.Runner` of `cli/internal/spec/review_launch.go` and requires each runner in `lane.Agents()`.
- [x] On this machine (Windows 11, 2026-11-17), `go run ./cmd/dotf doctor` at the merged code:

  ```
  [Agent launchers (what dotf starts)]
    [WARN] opencode: PATH resolves it to the batch file C:\Users\...\scoop\apps\nodejs-lts\current\bin\opencode.cmd and no native executable was found behind it, so dotf lane and dotf spec review refuse to start it. Install a native build (`mise use -g opencode` where mise has one)
  ```

  and with `--verbose` the four `[ OK ]` lines too: agy, claude and copilot start their `.exe` on PATH, pi starts `...\mise\installs\pi\1.5.2\pi.exe`, "the native executable behind the batch file" `...\.pi\agent\bin\pi.cmd`. It agrees with the measurement in #3572.

  How `Report` prints the section (corrected after the review; `proposal.md` says a skipped line needs `--verbose`, which is wrong and stays as written because the contract is closed): a `[WARN]` and a `[SKIP]` always print; an `[ OK ]` prints with `--verbose`; a section with any printed line has no `(N checks, all ok)` summary, and its hidden passes count only in the final `Results:` line.

## Test status

- `cd cli && go test ./internal/doctor/ -count=1 -run TestCheckAgentLaunchers` -> `ok ... 5.366s`.
- `go vet ./internal/doctor/` for Windows and for Linux: clean.
- The whole package, `go test ./internal/doctor/ -count=1`, fails on this machine with four tests that fail the same way without this change (three `TestCheckMiseTools_*` that need symlinks, `TestCheckWorkstationTarget_ByStatus` on a domain account). They are tracked in #1804 and #3247, not part of this change; CI runs the package on all three systems.

## Decisions made during implementation

- A warning, not a failure: the agent works from the shell, which does not go through dotf; only dotf's own launches refuse it.
- An agent that is not installed is a skipped line, as the ticket asks, not silence: the section then always shows the whole launch table.
- The tool-manager answer is a `System` seam (`NativeBehind`), nil in tests, so no test asks the machine's mise (lesson 611).
- The fix named is `mise use -g <agent>` with "where mise has one": doctor cannot know whether a native build exists without asking the registry.

## Review dispositions

Independent review: `review.md`, PASS WITH GAPS, `agy/gemini-4-hydrogen-high` on `382ad32a`.

- Major, THEORETICAL, spec-launch-parity (the warning says `dotf spec review` refuses a batch launcher, and a login-auth runner, agy, is started without the rule): ticketed as #3599 (BUG-175), with the fix in the review launcher. The warning's text is left as it is: it states the rule, and the hole is in the one launch path that does not apply it.
- Minor, REAL, verification-staleness (an outdated warning string pasted here, and `Report`'s non-verbose behaviour misstated): applied above, in this file. No non-verbose test added: what prints without `--verbose` is `Report`'s, held by `TestReportSuppressesPassesUnlessVerbose` and `TestReport_Golden`.
- Minor, THEORETICAL, lookup-consistency (PATH is asked twice per agent): declined. Two answers can differ only if PATH changes inside one check, and the worst outcome is a line naming a file one lookup old.
- Minor, THEORETICAL, wiring (nothing asserts the production System carries the seam): applied, `TestRealSystemAsksTheToolManager`.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`.

- [x] Lesson for the repo's `docs/lessons/`? no: the rule and its lesson (611) landed with #3570; this reports it.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: no decision changes.
- [x] New pattern candidate for `00_meta/patterns/`? no: a doctor check that asks the function the action asks is already how doctor's checks are written.

## Archive checklist

- [x] Independent adversarial review recorded in `review.md`
- [x] Issue #3572 closed by the archiving change
