---
tags: [spec, tasks, templates]
created: "2026-11-05"
id: "BUG-165-doc-path-lint-engine"
type: spec-tasks
status: active
owner: manu
---

# Tasks - BUG-165-doc-path-lint-engine

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from `origin/main`: `fix/3199-doc-path-lint-engine`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"
- [x] Run `/spec check BUG-165-doc-path-lint-engine`; expected: `PASS`
- [x] Set `proposal.md` status to `implementing` only after the consistency check passes

## Implementation

### PR 1 - Go engine, checkout policy and CI coverage

- [x] [P] [AC1] [AC6] [AC8] Port every accepted/rejected token case from `tests/check-doc-paths.bats` into table-driven tests under `cli/internal/docpaths/`; use temp repositories/vaults only. Run `cd cli; go test ./internal/docpaths`; expected: FAIL because the package does not exist.
- [x] [AC7] Add policy-loader tests for version 1, missing file, malformed JSON, unknown fields, unsupported version, empty governed names and escaping exclude prefixes. Expected: FAIL.
- [x] [AC7] Add `harness/doc-paths.json` and implement strict policy loading in `cli/internal/docpaths/policy.go`.
- [x] [AC1] [AC6] Implement token extraction and classification in `cli/internal/docpaths/scanner.go` without subprocesses per token; keep deterministic sorted findings.
- [x] [AC1] Implement repo-path, glob, escape and optional vault checks using native Go filesystem APIs.
- [x] [AC1] [AC6] Implement tracked-file discovery through one bounded Git invocation; sanitize incomplete `GIT_CONFIG_*` overrides and support linked Windows worktrees.
- [x] [AC1] Port the governed-pattern discovery-floor and zsh bare-source invariants from BATS into Go tests before any shell test is deleted.
- [x] [AC3] Add failing command tests in `cli/internal/cmd/lint_docpaths_test.go`; expected: `dotf lint doc-paths` is unknown.
- [x] [AC3] Add `dotf lint doc-paths [files...] --repo <path> --policy <path>` in `cli/internal/cmd/lint.go`; explicit files and auto-discovery share one engine.
- [x] [AC2] [AC8] Add a read-only current-repository test/benchmark that checks all governed files and fails above 2 seconds on Windows.
- [x] [AC5] Confirm `.github/workflows/cli.yml` has no PR/push path filter, so the current-repository Go test already runs when governed docs or policy change; no workflow edit is needed in PR 1.
- [x] [AC3] Verify PR 1 does not modify `.pre-commit-config.yaml`, `scripts/check-doc-paths.sh`, or either BATS file.
- [x] [AC1] [AC2] [AC3] [AC6] [AC7] [AC8] Run `cd cli; go test ./internal/docpaths ./internal/cmd`, `go vet ./internal/docpaths ./internal/cmd`, and the exact `features.json` commands.
- [x] [AC1] [AC6] Add review regressions for working-directory-relative explicit files, sanitized Git root discovery, surfaced glob traversal errors, linked worktrees, and the exact governed basename set.
- [x] Commit and open PR 1 as #3201.
- [ ] Verify CI and the existing shell hook pass on the remediation head.

### Release gate

- [ ] Merge PR 1 after CI/review triage.
- [ ] Cut and owner-merge the release PR containing `dotf lint doc-paths`.
- [ ] Update/install the pinned released `dotf`; verify `dotf lint doc-paths --help` exits 0 before PR 2.

### PR 2 - cut over and retire shell implementation

- [ ] [AC4] Add a failing guard test proving pre-commit and CI invoke `dotf lint doc-paths` and no longer invoke `scripts/check-doc-paths.sh`.
- [ ] [AC4] Switch `.pre-commit-config.yaml` to the released command and replace/remove the old `ci.yml` shell step according to the PR-1 CLI coverage.
- [ ] [AC4] Delete `scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`, and `tests/check-doc-paths-real.bats`.
- [ ] [AC4] Run the twin/test retirement guard and a repository-wide grep; expected: no live reference to the deleted script or BATS files.
- [ ] [AC1] [AC2] [AC5] [AC6] Run `dotf lint doc-paths` twice on Windows; expected: identical output, exit 0 and each run <=2 seconds.
- [ ] [AC1] Run the command on Linux CI and Windows CI; expected: parity in governed files and findings.
- [ ] Commit and open PR 2; verify no commit path depends on a pre-release binary.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass
- [ ] Lint passes
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/BUG-165-doc-path-lint-engine/features.json`):

```json
[
  {
    "id": "BUG-165-doc-path-lint-engine-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
