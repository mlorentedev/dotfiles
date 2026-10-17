---
spec: "CLI-131-lessons-index-on-read"
verdict: "FAIL"
reviewed_sha: "484070a3aeecb885f81a47de7d52a485be65c3a8"
reviewer: "nan/glm5.3-flash"
date: "2026-10-17"
---

## Adversarial review

**Scope**: CLI-131-lessons-index-on-read / head `484070a3`, base `ca2f673a65adaf7f4aa8a8b07492a22890324a00` (launcher-provided; verified ancestor of head). Round 2 — round 1 (`review-round1.md`, nan/qwen3.8-flash) was a FAIL on a review-binding Blocker and four smaller findings; all five are dispositioned in `verification.md` and re-checked below.
**Sources**: `specs/CLI-131-lessons-index-on-read/{proposal,tasks,features,verification}.md` + `git diff ca2f673a...HEAD` + this session's runs: `go build/vet` (+`GOOS=windows`), `go test ./...` (43 pkgs, 0 fail), golangci-lint v2.12.2 over the four touched packages (0 issues), all 7 `features.json` verification commands, `dotf lessons index`/`fmt --check` on the real tree, three mutation edits (reverted), and the `tests/*.bats` suite.

### Binding check (round-1 Blocker F1)

The round-1 Blocker was that `review-request.json` digests matched no file on disk. Re-derived the launcher's normalization (`normaliseContract`: CRLF fold, status line folded to `-` inside frontmatter only, list checkboxes unticked, `state`/`evidence` dropped from `features.json`) and reproduced all three digests byte-for-byte: `features.json 04a81dbc…`, `proposal.md ac9ea8d9…`, `tasks.md 5798ea71…` — each equals the value in `review-request.json`. F1 was a hashing-convention mismatch, not a defect; this review is bound to the contract it will be measured against at archive.

### Spec and task alignment

- Diff maps one-to-one onto the proposal's seven bullets: `Index`/`StrayIndexes` replace index read/write in `cli/internal/lessons/lessons.go`; `dotf lessons index` added and `fmt` help/behavior reworded in `cli/internal/cmd/lessons.go`; scaffold template gone (`scaffold.go`, template file deleted); staleness reads the newest lesson (`session_start.go`); `docs/lessons/_index.md` deleted (485 lines); stale pointers reworded (docs README, `docs/lessons.md` legacy stub, lane prompt, pre-commit name+comment).
- All 7 `features.json` verification commands re-run in this session: f1–f7 **PASS**. Real tree: `lessons index --dir docs/lessons` → **464 rows for 464 lesson files** (matches the re-measure instruction in `verification.md`, whose "463 at cdf425af" note pins the drift honestly), byte-identical across two runs, and `git status --porcelain docs/lessons` clean afterwards — nothing written. `fmt --check --dir docs/lessons` → `[OK]`, exit 0, no committed index.
- All implementation and Closing tasks ticked; every AC carries `[ACn]` markers and a non-vacuous command that greps a named `--- PASS:` line, so a no-match filter cannot fake a pass.
- Round-1 disposition check: parent `Short` no longer promises a generated index and `TestLessonsHelp_DoesNotPromiseACommittedIndex` pins it; the 11-repo blast-radius command is now recorded in `verification.md`; tasks boxes ticked. The WIP-override / `status: draft` question was left to the owner as recorded — still open as a Question below, not re-litigated.

### Independent adversarial evidence

- **Mutation 1 (sort removed from `renderTable`)**: deleted the `sort.Slice` line — `TestIndex_OneTablePerDirectorySortedByNumber` **still passes**. See Major F2.
- **Mutation 2 (staleness fix reverted: `newestLesson` returns not-found)**: `TestLessonsStaleness` **fails** as designed — the AC6 fix is pinned.
- **Mutation 3 (`StrayIndexes` returns nil)**: both `TestStrayIndexes_NamesACommittedIndexAtAnyLevel` and `TestLessonsFmt_CheckNamesACommittedIndexAndFmtLeavesItAlone` **fail** as designed — AC3 is pinned.
- All mutations reverted; working tree clean apart from this review.
- **bats**: `verification.md` claims "bats over the files that mention lessons … 98 tests, none failing" and enumerates ci-path-filtering among them. Re-run: `tests/ci-path-filtering.bats` **fails** at the reviewed sha. See Major F1.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Major | REAL | CI guard | `tests/ci-path-filtering.bats` `@test "HARNESS-041: every top-level entry is in the code filter or declared docs-only"` **fails at the reviewed sha**: `.agent/` — added by `f9195635 chore(autopilot): arm run pilot-r5`, a branch-local commit inside this review scope and not on origin/main — is in neither the code filter nor the docs-only list, so CI will not react to changes under it. This also falsifies `verification.md`'s "98 tests, none failing" (ci-path-filtering is in its enumerated list; the claim was true at the lessons PR's sha but is false at `reviewed_sha`). Not caused by the lessons change, but it is red in the change under review. | Reproduced in this session: `bats tests/ci-path-filtering.bats` → `not ok 4 … top-level entries in neither list: ['.agent/']`; `git branch -r --contains f9195635` → not on origin/main; `git show f9195635 --stat` → only `.agent/unattended.yml`. | The failing guard itself (`tests/ci-path-filtering.bats`, HARNESS-041) — the regression test exists and is red. | code — declare `.agent/` in the HARNESS-041 list with a reason (or do not land the branch-local arming commit); spec (`verification.md`, outside the contract set) — correct the bats line when re-measured. |
| Major | THEORETICAL | test strength / AC1 | The named AC1 test cannot detect the loss of its own subject: with the sort deleted from `renderTable`, `TestIndex_OneTablePerDirectorySortedByNumber` still passes, because its fixture (`lesson-001-a`, `lesson-002-b`, `category/lesson-003-c`) has alphabetical order == numeric order and `os.ReadDir` pre-sorts. Today no misbehavior is reachable — every lesson number is zero-padded to 3 digits (464/464, verified), so alphabetical == numeric — but a 4-digit number (`lesson-1000-*.md`, sorts before `lesson-101` alphabetically) or an unpadded name flips it, and no test would go red. Mutation demonstrated this session; misbehavior argued, not observed. | Mutation run: sort line deleted (`grep -c "sort.Slice"` → 0), `go test ./internal/lessons/ -run TestIndex` → `ok`. | `TestIndex_OneTablePerDirectorySortedByNumber` — named but mutation-blind → the sort path is UNTESTED until the fixture includes an alphabetically/numerically inverted pair (e.g. `lesson-010-*` vs `lesson-002-*`). | tests — extend the `TestIndex_OneTablePerDirectorySortedByNumber` fixture; no code change needed (the sort itself is correct). |
| Minor | THEORETICAL | resilience | `newestLesson` drops glob/stat errors (`continue` on `err`), the exact I/O-error-dropping pattern the change's own `readDir` comment calls out in `filepath.Glob` and replaces with `os.ReadDir` for the index. An unreadable `docs/lessons/` reads as "no lessons", silently falls back to `docs/lessons.md`, and the staleness warning dies — the same silent-death mode AC6 was written to remove, now reachable only via an unreadable dir. | Code read (`session_start.go`, `newestLesson`); no repro. | UNTESTED | code (or a named test if fixed) |
| Minor | REAL | quality | `fmt`'s RunE returns a `StrayIndexes` error bare, while every other failure path is wrapped `lessons fmt: <err>` — inconsistent user-facing error text. | Code read, `cli/internal/cmd/lessons.go` (the `stray, err := lessons.StrayIndexes(d)` block). | UNTESTED | code |
| Question | — | lifecycle | `proposal.md` frontmatter still `status: draft` and `wip_override` still reads "12 active, limit 10" (twice). Carried from round 1 as the owner's call; the archive gate writes `status: archived` itself, but the WIP breach disposition should be stated by the owner, not assumed. | Contract frontmatter at reviewed sha. | n/a — owner confirmation | contract set / owner decision |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All 7 ACs pass on the spec's own commands and independent probes (464/464 rows, deterministic, writes nothing, stray-index and duplicate/dangling error paths verified, mutations 2–3 caught). |
| Verification | C | Most evidence reproduced fresh in this session, but the suite the spec claims green (bats incl. ci-path-filtering) is red at the reviewed sha. |
| Scope | B | The spec-mapped diff matches the proposal exactly, but the scope also carries a branch-local arming commit (`f9195635`, `.agent/`) that breaks a guard — unrelated work inside the reviewed diff with a consequence. |
| Reliability | B | Error paths exit 1 naming file and fix; index is read-only and idempotent; one silent-death path survives in `newestLesson` (Minor). |
| Maintainability | B | Removal-heavy, small functions, help text pinned by a test; the repo's `gocyclo` exclusion outside `internal/cmd` (noted round 1) still means lint cannot catch a complexity outlier in `internal/lessons`. |
| Handoff-readiness | B | Spec folder coherent, lesson-464 captured, follow-up #2659 ticketed; held back by the falsified bats line and the open WIP/`status: draft` question. |

### Verdict

**FAIL** — one **REAL Major** (F1: a red guard test inside the review scope that also falsifies a `verification.md` evidence claim). F2 is THEORETICAL and tracked; the rubric's C in Verification agrees with the more severe path. The lessons change itself verified clean end-to-end; the FAIL is the red guard and the stale evidence line, not the spec's code.

### Recommended next steps

1. **Make HARNESS-041 green** (code): declare `.agent/` in `tests/ci-path-filtering.bats`' list with a reason, or drop the branch-local `.agent/unattended.yml` from what this branch lands. Re-run `bats tests/ci-path-filtering.bats` and paste the output into `verification.md` (outside the contract set — fixing it does not invalidate this review).
2. **Extend the AC1 fixture** (tests): add an alphabetically/numerically inverted pair to `TestIndex_OneTablePerDirectorySortedByNumber` so the sort mutation this review demonstrated becomes detectable.
3. **Disposition the two Minors** in `verification.md` (applied / ticketed / declined with reason): `newestLesson`'s dropped errors, and the unwrapped `StrayIndexes` error in `fmt`.
4. **Owner**: state the disposition of the WIP override (`12 active, limit 10`) so it is a decision, not residue.
5. With 1–2 applied, a re-review round should be short: the spec's seven acceptance criteria are verified green and need no re-implementation.

**Archive advisable now?** No — `dotf spec archive` may pass its own gates, but a change whose review scope carries a red guard test should not archive under a passing review. Archive becomes advisable after step 1 (and mechanically after the re-review that a REAL Major requires).
