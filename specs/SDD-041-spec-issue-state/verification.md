---
tags: [spec, verification, templates]
created: "2026-09-23"
---

# Verification - SDD-041-spec-issue-state

## Evidence

All commands were run on 2026-09-23 in `dotfiles-wt-spec-issue-state` (branch `feat/spec-issue-state`). Every `-run` criterion went through the §5 `gorun` guard, which fails unless at least one test ran.

- [x] **AC1**: `gorun ./internal/spec/... 'Archive.*Issue|IssueState'` → `gorun OK tests_run=41`. Tests added:
  - `TestIssueStateResolveFrontmatter`
  - `TestIssueStateResolveEmptyFrontmatterIsUnlinked`
  - `TestIssueStateResolveProse`
  - `TestIssueStateResolveProseRejectsNonTrackingRefs`
  - `TestIssueStateFrontmatterWinsOverProse`
  - `TestIssueStateGHLookup`
  - `TestIssueStateAuditClassifies`
  - `TestIssueStateAuditUnanswerable`
  - `TestIssueStateAuditNoSpecsDir`
  - `TestIssueStateNoActiveSpecIsProseLinked`

  Plus `TestFrontmatterKeepsHashInsideUnquotedValue`, which tests the parser fix and is outside the regex.
- [x] **AC2**: `gorun ./internal/cmd/... 'TestSpecAuditExitIssueState|TestSpecAuditRefusesWithoutAHomeRepo'` → `tests_run=5`. It covers a clean exit, a zombie (non-zero exit, names the spec) and an unanswerable lookup (non-zero exit, names the cause).
- [x] **AC3**: `gorun ./internal/doctor/... 'TestCheckSpecIssueState'` → `tests_run=6`. The test pins the mapping: PASS, closed issue → FAIL, 404 → FAIL, unanswerable → WARN and never PASS, `gh` absent → Skip and never PASS.
  - Live run: `DOTFILES_REPO_DIR=$PWD <dev build> doctor` prints a `[spec-issue-state]` section with 14 FAILs and 1 WARN (AI-024 unlinked).
- [x] **AC4**: GOV-004 now carries `issue: "mlorentedev/dotfiles#673"`. `gorun ./internal/spec/... 'TestIssueStateNoActiveSpecIsProseLinked'` → `tests_run=1`. Red before normalising: `specs/GOV-004-agents-md-diet links its issue only in prose`.
- [x] **AC5**: `dotf spec audit` in `~/Projects/hive` (master `3ddbcf0`) exits 1:
  - FAIL FEAT-015 → hive#380 CLOSED, resolved through its prose link
  - FAIL HIVE-119 → hive#151 CLOSED, resolved through its prose link
  - FAIL HIVE-267 → hive#267 CLOSED
  - WARN HIVE-118, unlinked

  Summary line: 7 active, 3 ok.

Live audit of this repo (`dotf spec audit`, 2.3 s wall): 42 active specs, 27 ok, 1 warn, 14 fail, 0 unanswerable.

## Test status

- `go build ./... && go vet ./... && GOOS=windows go vet ./...`: clean.
- `go test ./... -count=1`: every package ok.
- `golangci-lint run ./...` at the pinned 2.12.2: `0 issues.`
- `bats tests/spec-gate*.bats tests/check-spec-gate.bats tests/guard-spec-ids-unique.bats tests/agents-md.bats`: 113/113. These run because the spec gate reads the `issue:` field this change adds to GOV-004.
- **Mutation battery**: 12 mutants, one per invocation, each under `systemd-run --user --scope -p MemoryMax=1500M -p MemorySwapMax=0` and restored by `git checkout` in a trap (lesson 286). Build failures are reported separately from kills (lesson 284).
  - 12 of 12 were killed by a test failure; none were build errors.
  - An earlier, uncapped pass found that `nonTrackingWords` survived. Every measured negative is also rejected by the owner filter or the one-word label rule, so the pass added `same-owner related issue` and `same-owner upstream issue` to make that filter load-bearing.
- **Equivalent mutant, recorded rather than killed**: dropping the explicit `e.Name() == "archive"` skip in `resolveActiveSpecs` changes nothing, because `specs/archive/` never has a `proposal.md` at its own root. The check stays, for explicitness.

## Decisions made during implementation

- **Prose fallback is labelled and filtered by owner, never "the first `#N` near the top".** GOV-004's link is on line 85, and a survey of the 68 proposals with no frontmatter link (dotfiles and hive, active and archive, counted before GOV-004 was normalised) found false-positive shapes: `Upstream issue: anthropics/claude-code#59870`, `Related upstream issue: …`, `Sister sunset issue: … PR #121`, `(GH #197)`. A prose link grades WARN even when its issue is open, so the fallback stays a migration aid and does not become a second source of truth.
- **REST, not GraphQL.** REST has its own 5,000/h budget, away from the GraphQL budget that board automation exhausts, and a 404 cleanly separates "no such issue" (FAIL) from "could not ask" (unanswerable). Lookups run once per distinct issue, with 8 workers.
- **`frontmatterFields` fix.** It used to cut an unquoted value at any `#`. YAML opens a comment only at the start of a value or after whitespace. Without the fix, an unquoted `issue: owner/name#N` would have resolved to a truncated, invalid ref.
- **The repo-level guard fails; it never skips.** `RepoRoot(".")` never climbs (`filepath.Dir(".") == "."`), so the first version of `TestIssueStateNoActiveSpecIsProseLinked` skipped and passed while GOV-004 was still prose-linked. It now resolves from `os.Getwd()` and fails outside a checkout.
- **Inventory drift found by the tool itself.** The synthesis (2026-09-22) counted 16 dotfiles zombies. #1596 has since been reopened, so CLI-078 and CLI-080 are active work again. The synthesis also counted hive's FEAT-015 and HIVE-119 as "unlinked", and they are zombies. W1.4 (#1626) takes its inventory from `dotf spec audit`, not from the synthesis table.
- **Out of scope, already ticketed:**
  - #1358 (CLI-073): doctor calls a worktree "not a git checkout" because `.git` is a file there. Seen when running the dev doctor against this worktree.
  - #1251 (DX-010): `git checkout -- <file>` runs pre-commit, which stashes and restores unstaged edits.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **yes**: lesson-287, "a guard that skips is a guard that passes" (the vacuous `RepoRoot(".")` skip, next to the `gorun` rule).
- [x] ADR-worthy decision? **no**. The detective-over-preventive choice is recorded in the proposal and in #1087.
- [x] New pattern candidate? **no**. The vault already carries the rule that an unanswerable question is never reported clean (`pr-stewardship`).

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-041-spec-issue-state/` -> `specs/archive/SDD-041-spec-issue-state/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
