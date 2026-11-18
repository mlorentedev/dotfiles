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
- [x] AC6 -> `launcherSys` fails the test on any `CommandOutput`; `TestCheckAgentLaunchers_WithoutAToolManagerAnswer` runs with a nil seam.
- [x] On this machine (Windows 11, 2026-11-17), `go run ./cmd/dotf doctor`:

  ```
  [Agent launchers (what dotf starts)]
    [WARN] opencode: the only launcher is the batch file C:\Users\...\scoop\apps\nodejs-lts\current\bin\opencode.cmd, which dotf lane and dotf spec review refuse to start. Install a native build (`mise use -g opencode` where mise has one)
  ```

  The four other agents collapse into the section's passing count. It agrees with the measurement in #3572: pi has a native executable in mise, opencode has only a `.cmd`.

## Test status

- `cd cli && go test ./internal/doctor/ -count=1 -run TestCheckAgentLaunchers` -> `ok ... 5.366s`.
- `go vet ./internal/doctor/` for Windows and for Linux: clean.
- The whole package, `go test ./internal/doctor/ -count=1`, fails on this machine with four tests that fail the same way without this change (three `TestCheckMiseTools_*` that need symlinks, `TestCheckWorkstationTarget_ByStatus` on a domain account). They are tracked in #1804 and #3247, not part of this change; CI runs the package on all three systems.

## Decisions made during implementation

- A warning, not a failure: the agent works from the shell, which does not go through dotf; only dotf's own launches refuse it.
- An agent that is not installed is a skipped line, as the ticket asks, not silence: the section then always shows the whole launch table.
- The tool-manager answer is a `System` seam (`NativeBehind`), nil in tests, so no test asks the machine's mise (lesson 611).
- The fix named is `mise use -g <agent>` with "where mise has one": doctor cannot know whether a native build exists without asking the registry.

## Promotion candidates

- None.

## Archive checklist

- [ ] Independent adversarial review recorded in `review.md`
- [ ] Issue #3572 closed by the archiving change
