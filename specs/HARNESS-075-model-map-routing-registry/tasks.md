---
tags: [spec, tasks, harness, orchestration]
created: "2026-08-21"
---

# Tasks - HARNESS-075-model-map-routing-registry

> TDD order. One task = one focused commit. `[P]` marks a task with no dependency on another
> unchecked task; `[AC<n>]` maps it to an acceptance criterion in `proposal.md`.

## Setup

- [x] Branch created from main: `feat/harness-075-model-map`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — ADR-035 closed all four
- [ ] **PR #1136 (ADR-035) merged.** This spec transcribes its decisions; opening this spec's PR
      first would cite an ADR that does not exist on `main`.

## Implementation

> Schema before file, deliberately. Writing the file first and validating it afterwards means the
> first schema is shaped to fit whatever was written, which is how AC3's rejection case ends up
> untested.

- [ ] [P] [AC3] Write a failing Go test: a fixture whose `harnesses.<h>.pools[]` names a pool absent
      from `pools` must FAIL schema validation. This is ADR-032 §3's dangling `codex` defect, and it
      is the one rule the schema exists to enforce that a naive schema would miss.
- [ ] [AC2] [AC3] Author `harness/model-map.schema.json` until that test passes — the repo's first
      registry schema. Cross-block reference integrity is the hard part; the rest is types.
- [ ] [AC1] [AC4] Author `harness/model-map.json` from ADR-032 §3 **as amended by ADR-035**: seven
      blocks, both idioms, **no `openrouter` pool**, no dangling `codex` reference. The `$comment`
      carries the measured rationale — why `top` has no fallback, why the NaN reserve is static,
      and that the concurrency limit is per model rather than per API key.
- [ ] [AC1] [AC2] Test: the shipped map validates against the shipped schema. This is the assertion
      that keeps the two from drifting apart later.
- [ ] [P] [AC5] Write failing tests for the loader's two consumer classes — tier resolution
      (compile time) and chain resolution (run time) — asserting they are reachable separately.
- [ ] [AC5] Implement the loader in `cli/internal/harness/`, following the shape of `triggers.go`
      so it reads like its neighbours — **except for the embed**. No `//go:embed`, no fallback: an
      unreadable map is an error, because a build-time default is what C15 forbids (#1137).
- [ ] [AC7] Expose the budget fields (`concurrency`, `reserve_interactive`, `shared_with`) read-only,
      **with no counter**. Name the accessor so it cannot be mistaken for enforcement.
- [ ] [P] [AC6] Write failing tests for the doctor check across **three** broken states — absent,
      unparseable, schema-invalid — each asserting a distinct loud outcome and none rendering as an
      empty or permissive map.
- [ ] [AC6] Implement the check in `cli/internal/doctor/` — the repo's first doctor check over a
      registry.
- [ ] [AC8] bats coverage for the file's presence and the doctor check's output, matching how
      `check-review-attestation.bats` covers its registry.
- [ ] Refactor for clarity; confirm no unrelated changes in the diff.

## Closing

- [ ] Every acceptance criterion is covered by at least one test
- [ ] Every acceptance criterion has a matching `features.json` entry with a non-vacuous
      verification command
- [ ] [AC9] `go build ./... && go vet ./... && go test ./...` green
- [ ] [AC9] `GOOS=windows go vet ./...` green — the Windows CI leg compiles this tree and fails the
      whole package on one error
- [ ] [AC9] `golangci-lint run` green **on the version pinned in `versions.conf`**, not whatever is
      installed (BUG-071)
- [ ] [AC9] bats suite green
- [ ] `verification.md` filled in with the command output that proves each criterion
- [ ] Independent adversarial review from `harness/reviewer-pool.json` — required by the archive
      gate, and the reviewer must not be the implementer
- [ ] PR opened referencing this spec folder, **after PR #1136 has merged**

## Machine-readable features

`features.json` sits beside this file. The agent cannot write `"state": "passing"` — only the
harness may, after running the `verification` command and capturing exit 0.

Note: `dotf spec init` did not scaffold `features.json` at the time this spec was created; #1127
adds that. It was written by hand here.
