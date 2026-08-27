---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - WIN-007-harness-mirror

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/harness-mirror`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC1] Write failing tests: `TestMirror_CopiesTheTreeAndEveryDeclaredTarget`, `TestMirror_IsIdempotentAndDoesNotRewriteIdenticalFiles` (`cli/internal/harness/mirror_test.go`)
- [x] [AC1] Implement `harness.Mirror` (tree walk + manifest `.targets[].file`, byte-compare before write, atomic temp+rename)
- [x] [P] [AC2] Write failing test: `TestMirror_NamesADeclaredTargetTheCheckoutLacks`; implement `ErrMissingTargets` after mirroring the rest
- [x] [P] [AC3] Write failing test: `TestMirror_DoesNotPrune`; confirm no delete path exists
- [x] [AC1] [AC2] Write failing test: `TestHarnessMirrorCmd` (counts on stdout, gap on stderr, exit 1); implement `dotf harness mirror` and wire it into `newHarnessCmd`
- [x] [P] [AC4] Replace the `setup-linux.sh` bash+jq block with `dotf harness mirror` at the same position (after `--refresh`); insert the call in `setup-windows.ps1` beside `dotf tools install`
- [x] [AC4] Rewrite the `tests/setup-linux.bats` guards that asserted the bash block (mirror present, derivation in Go, ordering after `--refresh`); delete the jq-by-path guard (#1202) now that jq is not involved; add the Windows + parity guards in `tests/setup-windows.bats`
- [x] [P] [AC5] Flip the Windows subtest in `checks_harness_mirror_test.go` (orphan reported, not silent); drop the `GOOS == "windows"` early return and its false comment
- [x] [P] [AC5] Write failing test: `TestPiPackagesManifest_ReadsTheCheckoutBeforeTheMirror`; make `piPackagesManifest` checkout-first
- [x] Refactor for clarity: none needed (single-purpose file, no duplication with `deploy.go`'s atomic write beyond the two-line temp+rename idiom)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass (`go build ./... && go vet ./...`, Windows and `GOOS=linux`)
- [x] Lint passes (`golangci-lint run ./...` pinned 2.12.2; `shellcheck setup-linux.sh`; PowerShell parser + ASCII-only on the `.ps1` insert)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
