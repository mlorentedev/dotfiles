---
tags: [spec, tasks, templates]
created: "2026-06-13"
---

# Tasks - CLI-008-dot-spec-archive

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/dot-spec-archive`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> TDD order, one focused commit each. Domain logic in `internal/spec/`, wiring in `internal/cmd/spec.go`.

- [x] Write failing test for `FindUnresolvedTags` (decoy spec dir with `[AGENT-DRAFT]` + clean file → returns the tagged lines)
- [x] Implement `FindUnresolvedTags(specDir)` (walk + regex `\[AGENT-(DRAFT|SUGGESTION)\]`)
- [x] Write failing test for `setStatus` (frontmatter line rewritten; decoy `status:` in body untouched; no-frontmatter file unchanged)
- [x] Implement `setStatus(content, newStatus)` (first `---`…`---` block only; value-only replace preserves trailing comment)
- [x] Write failing test for `Archive` happy path (move → `specs/archive/ID`, status `archived`) + `--abandoned` route + no-clobber + missing-spec + PR-URL append
- [x] Implement `Archive(repoRoot, id, opts)` orchestrating tag-check → target → move → status rewrite → PR comment
- [x] Write failing cmd test for `dot spec archive` (subcommand listed, flags parsed, blocks on drafts, `--force-with-drafts` overrides)
- [x] Implement `newSpecArchiveCmd` and register it on the `spec` parent
- [x] Refactor for clarity (reused `spec.RepoRoot` + the cmd `now` clock; no premature abstraction)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-008-dot-spec-archive/features.json`):

```json
[
  {
    "id": "CLI-008-dot-spec-archive-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
