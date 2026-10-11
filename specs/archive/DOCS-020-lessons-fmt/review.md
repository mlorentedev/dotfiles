---
spec: "DOCS-020-lessons-fmt"
verdict: "PASS"
reviewed_sha: "e8920db51d75acaaeb40bd933a35e0f6e678330f"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-09"
---

## Adversarial review

**Scope**: DOCS-020-lessons-fmt (PR 1 #2039 / PR 2 #2211, issue #2038)
**Sources**: `specs/DOCS-020-lessons-fmt/{proposal,tasks,verification}.md` + `features.json`; `git diff 517cc597315da322c0ff24bc8367eb175531c7e5...HEAD` (888 files); commits `12688312` (PR 1) and `11c56bc0` (PR 2). Base `517cc597` is an ancestor of HEAD; the remainder of the diff is `origin/main` merged in for the 0.65.0 pin.

### Spec and task alignment

Every task in `tasks.md` is ticked and every `[x]` has matching evidence. All seven ACs map to named tests or workflow files, and I reproduced each `features.json` `verification` command (all 8 exit 0). AC7's claim that "every assertion" of the two deleted bats files lives in Go first is *almost* complete — see F1.

Commands re-run in this session (fresh, not read from `verification.md`):

- `go build ./...` — exit 0.
- `go test ./internal/lessons/... ./internal/initrepo/... -count=1` — `ok` both.
- `go run ./cmd/dotf lessons fmt --check --dir ../docs/lessons` — `[OK]`, exit 0 (AC1, AC2, AC4, idempotence).
- All 8 `features.json` verification commands — PASS (f1–f8).
- Mutation, in a scratch tree (`/tmp/lm`, repo untouched): an unformatted lesson+index → `not formatted: …`, exit 1; a `[[lesson-999-nope]]` link → names `lesson-001-a.md links [[lesson-999-nope]]`, exit 1.
- Content preservation across the migration: a line-signature comparison of the 340 pre-existing lesson files against `517cc597` found **zero** body changes other than the H1 line and the inline `**Date:**` line, and **zero** frontmatter keys dropped (only `date` is retired). The one apparent extra section (lesson 246) is an `## Update (#1908)` added by an unrelated `main` PR.
- Tree state: 366 lesson files, 366 generated index rows, no duplicate number, gaps at 253 and 367 (a gap is not an error), index prose preserved, markers present.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL (deletion bar); scenario THEORETICAL today | test deletion / wikilink coverage | The deleted `guard-lesson-numbers-unique.bats` test 4 scanned **every** file under `docs/lessons` (`grep -rhoE '\[\[lesson-…\]\]' "$LESSONS_DIR"`), including `_index.md`; the replacement `danglingLinks(seen)` iterates only the lesson files in `seen`. A lesson wikilink written into the index prose is therefore no longer checked. `check-lessons.sh` §2 also extracted index `[[…]]` targets (`index_targets`) — that half is gone too. Deletion-bar field 2 is not fully shown for this half. Nothing is missed **today**: `_index.md` has no `[[lesson-…]]` (its two `[[` occurrences are `[[:space:]]` and a title's `[[ ]]`). | Code read of `danglingLinks`; `grep -n '\[\[' docs/lessons/_index.md` shows 0 lesson wikilinks; deleted test shown at `517cc597:tests/guard-lesson-numbers-unique.bats` | `TestPlan_RefusesAWikilinkThatNamesNoLesson` — covers lesson files only, so the index path is UNTESTED | code + tests (extend `danglingLinks` to scan each `_index.md`, add a case with a dangling link in the index) |
| Minor | REAL | enforcement window | The pre-commit hook runs the **installed** `dotf` (DOTF_VERSION 0.65.0), which `verification.md` states lacks the wikilink check; only CI's checkout build enforces it. A contributor gated only by pre-commit can commit a dangling wikilink that CI then rejects. Documented in `verification.md` and the hook comment; self-resolves with the next release. | `.pre-commit-config.yaml` entry + the comment beside it; `verification.md` "Test status — PR 2" | UNTESTED (a release-state assertion; not reproducible in-tree) | none in code — record the follow-up release ticket so DoD *Debt* is a verdict, not a promise |
| Minor | THEORETICAL | test coverage / hook wiring | The hook moved from `language: script` to `language: system`, so `tests/precommit-hooks-portable.bats` (which executes only `language: script` entries) no longer exercises it. The entry string `dotf lessons fmt --check --dir docs/lessons` has no test asserting it resolves. | `tests/precommit-hooks-portable.bats` filters on `language: script`; the new hook is `language: system` | UNTESTED | tests (assert the hook entry, or keep a thin script wrapper) |
| Minor | REAL | behavior change | `Plan` no longer errors when the index lists a file that does not exist: it regenerates the table and drops the stale row (old `check-lessons.sh` printed `indexed but missing: X` and exited 1). This is the deliberate "file is the source, index is derived" model (proposal §What), so it is a design decision to record, not a defect. | `planDir` regenerates for `files == 0`; `TestPlan_RegeneratesAnIndexWhoseDirectoryHasNoLessonsLeft` | `TestPlan_RegeneratesAnIndexWhoseDirectoryHasNoLessonsLeft` | none required; if the stricter semantics are wanted, a follow-up ticket |
| Question | THEORETICAL | reporting fidelity | Deleting `check-lessons.bats`' case "a lesson missing from the index is still reported" changes what `--check` names: the old check named the **lesson**; the new one names the unformatted **index**. AC4 says "naming every unformatted file", which the index satisfies — confirm the looser phrasing is intended. | `check-lessons.bats` test 2 at `517cc597`; `cmd/lessons.go` prints `not formatted: <path>` | `TestLessonsFmt_CheckFailsThenFmtConvergesThenCheckPasses` (asserts the index is named, not the lesson) | spec (`verification.md` note) if the wording should be tightened — do **not** edit the contract set |

No Blocker. No REAL Major. No security-relevant surface (this is a docs formatter; no secrets, no network, no auth, no user input beyond the repository's own files). Code-level checklist clean: no injection seam (no shell), no hardcoded credentials, bounded `ReadDir`/`Glob`, errors returned not swallowed, no blocking/asynchrony, no unbounded buffers. `Apply` writes non-atomically, but `fmt` is idempotent, so a mid-run failure is repaired by re-running.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All 7 ACs verified and reproduced; negative paths (duplicate number, missing index, missing root index, dangling wikilink) are covered — except the `_index.md` half of the wikilink guard (F1). |
| Verification       | A | `verification.md` maps every AC to a named test/command; all 8 `features.json` commands plus build, `fmt --check`, and mutation cases reproduce exactly as written. |
| Scope              | B | The DOCS-020 delta is two focused commits; the reviewed diff also contains `origin/main`'s merge (documented in `verification.md`), which inflates the file count but is not author creep. |
| Reliability        | B | Idempotent (second `--check` exit 0), every structural failure is an error, `--check` exits 1 naming files; `Apply` is sequential/non-atomic but re-run-safe. |
| Maintainability    | A | Small functions, low branching, and comments that explain *why* (`[ \t]` vs `\s`, measured prefix forms, no regex built from filenames); no dead code. |
| Handoff-readiness  | A | Spec triad complete, verification reproducible, decisions log records the two review-driven course corrections, and all three promotion lines are answered. |

### Verdict
PASS

The rubric is all B-or-above with no D, and the findings axis is all Minors, so neither path escalates. `dotf spec archive` is **advisable** in the current state: no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain, and the review is fresh against `proposal.md`/`tasks.md`/`features.json` as recorded in `review-request.json`.

### Recommended next steps

Disposition each in `verification.md` (excluded from the staleness check) or carry it into a follow-up ticket — do **not** edit `proposal.md`, `tasks.md` or `features.json`, which would invalidate this verdict:

1. F1 (code + tests): have `danglingLinks` also read each directory's `_index.md`, and add an index-dangling-link case; this closes the one assertion the deleted bats had that the Go port does not.
2. F2 (ticket): file the release follow-up that the hook comment already describes — "wikilink check reaches pre-commit with the first release after 0.65.0" — so the enforcement window is tracked, not merely commented.
3. F3 (tests): a named test that the `lessons-fmt` hook entry resolves, since the portability guard no longer sees it once it is `language: system`.
4. F4 / Question: record the deliberate semantic shift (a stale index row is dropped, not errored) and confirm AC4's "naming every unformatted file" is the intended wording.
