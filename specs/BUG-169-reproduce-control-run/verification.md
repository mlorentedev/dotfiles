---
tags: [spec, verification, templates]
created: "2026-11-12"
id: "BUG-169-reproduce-control-run"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-169-reproduce-control-run

## Evidence

- [x] AC1: `TestRunWithAProbeNeedsTheCleanHeadAsItsControl` has two cases here. "fails with and without the probe" uses `git grep -q three`. "passes with and without the probe, inverted" uses `git status --defect-when-passes`. Both give `could-not-run` with control `reproduced`, the output says "on the clean head", and the tree is left clean.
- [x] AC2: the same test's "fails only with the probe" and "passes only with the probe, inverted" cases give `reproduced` with control `did-not-reproduce`. `TestRunWithAProbeRunsNoControlForARefutation` gives `did-not-reproduce` with no control. The existing `TestRunWithAProbe` and `TestRunWithAProbeFromASubdirectory` pass unchanged.
- [x] AC3: `TestReproductionRowNamesTheControl` checks the full row with a control, and checks that a row without a control names none.

## Test status

- `go test ./...` in `cli/`: every package ok. `go vet` and `golangci-lint run` report 0 issues, natively and with `GOOS=windows`.
- Smoke test, the #3325 incident replayed on this branch: `go run ./cmd/dotf pr reproduce --probe p.diff -- go test -C cli ./internal/reproduce -run TestClassify -count=1`, run from `cli/` with a probe that changes a comment. It printed `Reproduction: could-not-run (exit 1, control reproduced (exit 1) on the clean head) ...` and exited 1. Its output names `go: chdir cli: no such file or directory`. Before this change, the same run printed `reproduced`.

## Decisions made during implementation

- The control runs after the probe is reverted, and only when the probe run reproduced. A refutation needs no control, so a reproduction attempt that refutes costs nothing extra.
- A control that is itself `could-not-run` also makes the result `could-not-run`. Only a control that refutes leaves the reproduction standing.
- The row grows a `control <state> (exit N) on the clean head` clause inside its parentheses. `prtriage` matches only the row's `Reproduction: <state>` prefix, so its parsing is unchanged.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: the cause and the fix are in the issue and in the command's help
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: it applies option A of an existing contract (#1062, TOOL-017)
- [x] New pattern candidate for `00_meta/patterns/`? no: specific to this tool
