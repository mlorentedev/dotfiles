---
tags: [spec, tasks, templates]
created: "2026-08-08"
---

# Tasks - BUG-041-spec-tag-preflight

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Worktree from `origin/main`: `dotfiles-wt-769`, branch `fix/spec-archive-markdown-aware`
- [x] Gating issue open: dotfiles#769
- [x] `proposal.md` complete, acceptance criteria testable

## Implementation

- [x] Establish the canonical emitted form from `harness/skills/spec/SKILL.md`
      rather than from the existing pattern, which is the artefact under suspicion
- [x] Confirm the false negative against the tree, not by reasoning: grep the
      archive for the suffixed form and find `CLI-002-repo-structure` carrying one
- [x] [AC2][AC3] Widen the pattern to both shapes
- [x] [AC1][AC4] `ScanUnresolvedTags` — fence tracking, code-span stripping,
      ticked-checkbox skipping, as one exported predicate
- [x] [AC1-AC4] Table-driven Go tests, both directions
- [x] [AC5] Point the session-start injector at the same predicate; drop its
      independent substring scan
- [x] [AC5] Verify against the live repro: the patched binary no longer flags
      `AI-028-hive-install-model-migration`

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] `gofmt` clean on the touched files; `go build ./...` and `go vet ./...` pass
- [x] Full Go suite green (12 packages)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
- [x] Spec archived in the same PR, using the freshly built binary — the old one
      cannot archive this spec, which is the bug
