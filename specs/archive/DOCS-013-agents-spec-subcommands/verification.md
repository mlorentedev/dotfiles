---
id: "DOCS-013-verification"
type: spec
status: draft
owner: manu
tags: [spec, verification, templates]
created: "2026-08-18"
---

# Verification - DOCS-013-agents-spec-subcommands

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof:

- [x] AC1: `AGENTS.md` accurately documents `dotf spec` CLI subcommands (`init`, `review`, `archive`) and distinguishes skill-only steps.
  - Proof: `grep -q 'dotf spec init' AGENTS.md && grep -q 'dotf spec review' AGENTS.md && grep -q 'dotf spec archive' AGENTS.md && ! grep -q 'dotf spec fill' AGENTS.md` -> Exit code 0.
- [x] AC2: `TestSpecSubcommandsProseMatchesCode` passes on clean `AGENTS.md`.
  - Proof: `go test -count=1 -v -run TestSpecSubcommandsProseMatchesCode ./internal/spec/...` -> `PASS` (0.01s).
- [x] AC3: Mutation test demonstrates `TestSpecSubcommandsProseMatchesCode` goes RED when an invalid subcommand is inserted into `AGENTS.md`.
  - Proof: Injected `fakecommand` into `AGENTS.md` subcommands list $\rightarrow$ `drift_test.go:178: AGENTS.md introductory list claims dotf spec fakecommand exists, but fakecommand is not a binary subcommand` (FAIL, exit code 1). Reverting restored green (PASS).

## Test status

- Unit test suite: `go test -count=1 ./...` in `cli/` -> 100% passing across all 15 packages.
- No regressions: clean test execution.

## Decisions made during implementation

- Bound the prose in `AGENTS.md` directly to the Cobra command tree via `TestSpecSubcommandsProseMatchesCode`, following the precedent of `TestIDPatternProseMatchesCode`.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? No (covered by existing prose-to-code drift pattern).
- [x] ADR-worthy decision? No (aligns prose with ADR-020).
- [x] New pattern candidate? No.

