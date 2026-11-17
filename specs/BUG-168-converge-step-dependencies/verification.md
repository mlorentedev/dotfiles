---
tags: [spec, verification, templates]
created: "2026-11-16"
id: "BUG-168-converge-step-dependencies"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-168-converge-step-dependencies

## Evidence

Tests are in `cli/internal/converge/` unless a path says otherwise.

- [x] AC1 -> `TestRun_AFailureHoldsBackOnlyItsDependents`
- [x] AC2 -> `TestRun_JoinsEveryFailure`, and the first assertion of `TestRun_AFailureHoldsBackOnlyItsDependents` (a later step changed the machine; the error still names the failure)
- [x] AC3 -> `TestRunObserved_ReportsEachStepItRuns`; `TestConvergeConsole_Streams` in `cli/internal/cmd/` with its two goldens
- [x] AC4 -> `TestRun_AnUndeclaredStepNeedsEveryStepBeforeIt`
- [x] AC5 -> `TestRun_AnAbsentOrSkippedNeedDoesNotBlock`
- [x] AC6 -> `TestRun_RevisitSkipsAFailedStepAndItsFailureIsNotTerminal`, `TestRun_ARevisitThatFailsHoldsBackItsDependents`
- [x] AC7 -> `TestRegistry_EveryStepDeclaresItsNeeds`, `TestRegistry_EachNeedIsAnEarlierStep`
- [x] AC8 -> `TestRegistry_WhatAFailureHoldsBack`, `TestRegistry_TheQuietDayStepsDoNotNeedEachOther`
- [x] AC9 -> `TestCheckSelfDeploy_ByState` in `cli/internal/doctor/`, case "last run failed in two steps"

## Test status

Run on Windows 11, 2026-11-16, from this worktree.

- `go test ./internal/converge/ -timeout 30m` -> `ok` (207s)
- `go test ./internal/cmd/ -run 'Converge|Update' -timeout 30m` -> `ok` (62s)
- `go test ./internal/doctor/ -run TestCheckSelfDeploy -count=1` -> `ok`
- `go build ./...`, `go vet ./...`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...` -> clean
- `go run ./cmd/dotf converge --plan` -> exit 0, 24 steps reported in registry order (4 to change, 16 ok, 1 opaque, 3 skipped, 0 failed). A plan writes nothing.
- **Not run locally:** the whole `go test ./...` (on this machine several packages exceed the default timeout with or without this change; CI is the reference), `golangci-lint` (the checkout is CRLF and the linter reports gofmt noise), and any applied `converge`. The installed `dotf` does not carry #3555, so an apply raises elevation prompts on this machine. No real run with a failing step was made: the failure paths are proven with fakes only.
- No regressions in the suites run: yes.

## Decisions made during implementation

- **The order of the registry is not touched.** Needs decide what a failure holds back, not when a step runs. A re-sort by needs would have changed ADR-041 decision 4 instead of amending it.
- **An undeclared step needs every earlier step.** The existing fake-based tests kept their meaning, and a step added without a declaration is safe until someone decides.
- **`legacy-setup` stays undeclared.** So a failed leaf step still holds back the setup script. That is a deliberate limit, stated in ADR-064 decision 6, not an oversight: the script re-runs native work.
- **Uncertain edges were kept**, including ones only a comment asks for (`system-upgrade` after `tools`, `env-generate` after `records-mirror`). The table in `proposal.md` marks them.
- **`records-bind` and `agent-extensions` need `records-harness`.** No code reads it there. Before this change a failed revisit of `records-harness` stopped the run; these two write harness settings, so they keep that protection.
- **A held-back step is told to the observer.** Before, the console printed unreached steps as a tail after the run. They can now sit between steps that ran, so they print in place.
- **The detail keeps the `not reached` prefix** and appends the cause, so a reader matching the prefix still matches.
- **`dotf doctor` cuts a recorded error at its first line.** Found while checking who reads the joined error; step errors could already be multi-line before this change.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the reasoning is a decision, recorded in ADR-064; nothing here was learned by surprise
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-064-a-failed-converge-step-holds-back-only-its-dependents.md
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: one project, one runner

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-168-converge-step-dependencies/` -> `specs/archive/BUG-168-converge-step-dependencies/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
