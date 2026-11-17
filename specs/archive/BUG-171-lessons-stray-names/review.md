---
spec: "BUG-171-lessons-stray-names"
verdict: "PASS"
reviewed_sha: "7f8ed464fbe4f9da16f363c7059abf07c5f288c9"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: BUG-171-lessons-stray-names — `dotf lessons fmt --check` fails on a markdown file in no `lesson-NNN-<slug>.md` form (issue mlorentedev/dotfiles#3415)
**Sources**: `specs/BUG-171-lessons-stray-names/{proposal,tasks,verification,features}.md` + `git diff a98a270c...HEAD` (spec commits 2ba61577, 7f8ed464; `git log --first-parent --no-merges` shows no other branch work in the range)

### Contract freshness

`dotf spec digest BUG-171-lessons-stray-names` reproduces the `contract_digests` in `review-request.json` byte-for-byte (features.json, proposal.md, tasks.md; spec.md empty as expected for a full triad). This review is fresh against the current contract.

### Spec and task alignment

- Proposal AC1/AC2/AC3 all map to real code and were verified against a freshly built binary, not against assertions.
- Tasks.md: all implementation boxes ticked; the final closing box ("verification.md filled; independent review; archive") is unticked — correct for this phase, this review is its gate.
- `features.json` states are all `pending` with empty evidence — honest pre-harness state; the tasks.md rule refuses `passing`+empty-evidence, which is not the case here. The harness capture remains a next step.
- Refactor of fmt reporting into `lessonsFinding`/`lessonsLeftAlone`/`reportLessonsCheck` is disclosed in tasks.md and verification.md, and the diff matches. The pre-existing committed-index output strings (`committed index:`, `left alone: ... (a committed index goes stale...)`) are byte-identical after the refactor — `TestLessonsFmt_CheckNamesACommittedIndexAndFmtLeavesItAlone` covers this and passes.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file.

### Verification performed (all fresh, this session)

- `go test ./internal/lessons/` and `go test ./internal/cmd/` (full packages): ok (0.09s / 44.3s).
- `TestLessonsFmt_CheckNamesAFileInAnotherNamingScheme` + 7 other lessons tests: PASS.
- `go vet`, `GOOS=windows go vet`: clean. `golangci-lint run` (gocyclo enabled): 0 issues, native and `GOOS=windows`, both packages.
- Built `dotf` from source; probes in a `mktemp -d` scratch:
  - **AC1**: stray `2026-12-01-stray.md` at root + `ops/2026-12-02-also.md` in a category dir → exit 1, both named `unrecognised name: <path>`, fix line states `lesson-NNN-<slug>.md` form. PASS.
  - **AC2**: tree with only `_format.md`, `README.md`, `ops/README.md`, `notes.txt` → exit 0 `[OK]`. This repo's own `docs/lessons` (584 files, flat) → exit 0 `[OK]`, the same command CI runs (`.github/workflows/repo-hygiene.yml:30`). PASS.
  - **AC3**: `fmt` without `--check` → exit 0, `left alone: <path> (not a lesson-NNN-<slug>.md name, so nothing formats or indexes it)` for every stray, file remains in place, idempotent on a second run. PASS.
  - Error paths fail loud, not silent: missing `--dir` → exit 1; `lesson-notes.md` (matches `readDir`'s `lesson-` prefix but not `fileRE`) → Plan errors naming the file, exit 1.
  - Boundary consistency with the code is real: `StrayNames` iterates exactly `lessonDirs` (root + first-level), the same dirs `Plan`/`Index` read, and accepts exactly what `fileRE` plus the docs map accept — the deeper-than-level-1 silent pass is the spec's declared non-goal, not drift.
- **TDD red-green confirmed by mutation** (in scratch copies, repo untouched): unwiring `lessons.StrayNames` from `lessonsLeftAlone` makes the new test fail with exactly the pre-fix symptom verification.md claims (`check passed a tree holding stray lessons: [OK]`).
- Issue #3415: OPEN, assigned — board state matches an un-merged change.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|--------|---------------------------|---------------------------------------------|
| Minor | REAL | tests | The `"_index.md": true` exclusion in `lessonsDirDocs` (a decision documented in verification.md — naming it twice would give two fixes, one of them the wrong fix for an index) has **no covering test**. Deleting the entry keeps every test green: `go test ./internal/cmd/ -run TestLessons` and `./internal/lessons` both pass on the mutated copy. Today's behavior is correct (probe: `ops/_index.md` named once as `committed index` only, never as `unrecognised name`), but nothing pins it. | Mutation run this session (surviving mutant, both packages); probe of built binary | UNTESTED — fix: extend `TestLessonsFmt_CheckNamesAFileInAnotherNamingScheme` with `ops/_index.md` and assert no `unrecognised name: ..._index.md` line (the test file is outside the contract set; it can be amended without invalidating this review) | tests |
| Minor | SPECULATIVE | spec wording | A file named `NOTE.MD` (uppercase extension) still passes `--check` in silence — the original defect class. Probed: not named, exit 0. The whole subsystem consistently treats only lowercase `.md` as markdown (`fileRE`, `readDir`'s `HasSuffix(".md")`), so the behaviour is internally coherent and shell `*.md` glob semantics arguably exclude it; but a case-insensitive filesystem could produce one. | Probe `NOTE.MD` → `[OK]`; code read of `fileRE`/`readDir` | Surface only; no measured repo has such a name | — (spec note if ever a fleet case appears) |
| Question | UNVERIFIED | proposal risk claim | The "Risks" blast-radius measurement ("every local checkout under `~/Projects` … the only non-lesson markdown files are `_index.md` and `_format.md`") cannot be re-verified from this worktree (search outside the repo is out of scope). The claim is plausible and directionally self-checking: this repo passes, and the check fails loud where it holds wrong. | n/a | n/a | — (accept as author evidence) |
| Minor | SPECULATIVE | UX | The stray fix line says "on the next free number" but no command prints that number (verification.md itself confirms `renumber --next` refuses such a file). The user must eyeball `dotf lessons index` or `ls`. Deliberate — the proposal's Out-of-scope names the number as a human choice — so this is the chosen design's residue, not a defect. | verification.md smoke note + `renumber` help | n/a | — (disposition only) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All three ACs verified against the built binary incl. negative and error paths; no observed defect against the spec's stated rule |
| Verification | B | Reproducible commands, red-green proven by mutation, but the `_index.md` exclusion decision is untested and fleet evidence is out-of-repo |
| Scope | A | Diff matches the proposal exactly; refactor disclosed; no unrelated changes on the branch (first-parent log: only the 2 spec commits) |
| Reliability | A | Missing dir / unreadable dir / prefix-but-not-regex names all fail loud; fmt idempotent on re-run; check writes nothing |
| Maintainability | A | Table-driven findings, short functions, gocyclo-enforced lint 0 issues native + GOOS=windows |
| Handoff-readiness | B | Spec triad complete and honest; features.json evidence and the closing box correctly pending; the surviving-mutant gap is not yet tracked |

### Verdict

**PASS** — no Blocker, no REAL Major; every open finding is Minor or a Question, rubric has no C or D (all A/B).

`dotf spec archive` is advisable in the current state, once the harness has run the `features.json` verification commands (that is a lifecycle step, not a review gap). Two follow-ups are tracked here, neither blocking:

### Recommended next steps

- **tests (can be applied without a re-review — `cli/internal/cmd/lessons_test.go` is outside the contract set):** add `ops/_index.md` to the naming-scheme test's tree and assert it is never named as `unrecognised name`; this pins the double-report exclusion that verification.md documents as a deliberate decision (its mutant currently survives the suite).
- **verification.md (disposition line for the implementer):** either add that regression above or decline it with a reason, per Definition of Done §1; the same for the uppercase-`.MD` and "next free number" notes (both fine to decline as spec-chosen boundaries).
- Let the harness capture `features.json` evidence (all three commands in it are executable and were run successfully in equivalent form this session) before archiving.
- #3415 should be linked and closed by the PR that merges this — board check for the shipper, not for this verdict.
