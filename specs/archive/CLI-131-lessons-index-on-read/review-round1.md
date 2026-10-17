---
spec: "CLI-131-lessons-index-on-read"
verdict: "FAIL"
reviewed_sha: "f9195635f7e7e184adef364f0dfdd854cdc2e97e"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-17"
---

## Adversarial review

**Scope**: CLI-131-lessons-index-on-read / head `f9195635`, base `ca2f673a65adaf7f4aa8a8b07492a22890324a00` (verified: base **is** an ancestor of head — `git merge-base --is-ancestor` exit 0)
**Sources**: `specs/CLI-131-lessons-index-on-read/{proposal,tasks,features,verification}.md` + `git diff ca2f673a...HEAD` + the built binary `cli/cmd/dotf` at head. No open PR was found for this branch (`gh pr list --repo mlorentedev/dotfiles --head chore/autopilot-pilot-r5-run --state all` → `[]`).

> **Read this first.** The **implementation is green**: all seven acceptance criteria verified passing, by the spec's own commands and by independent CLI probes. The FAIL is not about the code — it is F1, a review-binding defect that makes *this artifact* uncertifiable against the contract that would be archived. A re-dispatched round should cost minutes, not a cycle.

### Spec and task alignment

- `proposal.md`, `tasks.md`, `features.json` and `verification.md` are **mutually coherent** and describe one change: `docs/lessons/_index.md` stops being committed; `dotf lessons index` renders it to stdout; `fmt`/`--check` stop requiring it; `--check` reports a committed index as drift; `dotf init` stops scaffolding it; session-start staleness reads the newest lesson.
- The diff at head matches that proposal and nothing else: `D docs/lessons/_index.md`, `D cli/internal/initrepo/templates/lessons-index.md`, `M cli/internal/lessons/lessons.go`, `M cli/internal/cmd/lessons.go`, `M cli/internal/mem/session_start.go`, plus their tests. No creep found.
- Every Implementation task in `tasks.md` is ticked; `[ACn]` markers cover AC1–AC7. All 7 `features.json` verification commands exit 0 at head (run in this session: `f1 PASS f2 PASS f3 PASS f4 PASS f5 PASS f6 PASS f7 PASS`).
- AC markers are present on every implementation task, so the §4 traceability gate is met in form **and** I re-checked it in substance: the commands grep for `^--- PASS: <TestName> `, which cannot pass on a filter that matched nothing (this repo's lesson-309 defence), so "f1–f7 pass" means those named tests actually ran.

### Independent behavioural evidence (not taken from `verification.md`)

Built `go build -o /tmp/dotf-head ./cmd/dotf` at head and probed a scratch tree:

- AC1 render: three lesson files fed out of order print `002, 003, 005` — sorted by number, one row per lesson; a nested `shell/` directory gets its own `## shell` section; **no file written** (`git status --porcelain docs/lessons` clean after `lessons index --dir docs/lessons` on the real tree).
- Determinism: two consecutive renders produce identical `cksum`.
- AC2/AC3: with a committed `_index.md` present, `fmt --check` exits 1 with `committed index: <path>` and the fix line `run: git rm the index; render it with \`dotf lessons index\``; a bare `fmt` exits 0 and **leaves the index in place** (listed after the run). A stray index in a **category subdirectory** is also caught (`committed index: …/shell/_index.md`) — the "any level" claim holds.
- AC4: duplicate number → exit 1, `lesson number 005 is used twice: <pathA> and <pathB>`; dangling link → exit 1, names the linking file and the missing `[[lesson-999-does-not-exist]]`.
- AC7: `docs/lessons/_index.md` absent from head (`git cat-file -e HEAD:…` fails) and `lessons fmt --check --dir docs/lessons` prints `[OK] lessons are in their canonical form`, exit 0.
- On the real tree `lessons index` prints **463** rows for **463** `lesson-*.md` files.
- `dotf lessons` at head exposes only `fmt` and `index`: no `read`, no `reindex`, no `--root`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| **Blocker** | **REAL** | review binding | The three contract digests pinned by the launcher match **none** of the three contract files sitting at the reviewed sha. `review-request.json` says `features.json 04a81dbc…`, `proposal.md ac9ea8d9…`, `tasks.md 5798ea71…`; the files on disk hash to `f0bfe489…`, `0f78b818…`, `c1234c0f…`. So this verdict is provably not about the same contract bytes the round was dispatched on — and `dotf spec archive` measures staleness against exactly those three files. | Measured three ways in one session, none matches: raw `sha256sum`, trailing-newline-stripped `printf '%s' \| sha256sum`, `git hash-object`. Corroborated by the tree moving mid-review: `docs/lessons/lesson-465-…`, `specs/…/review.md` (15645 bytes, a Round-4 review of an *older* `index.json` design) and commits `37737f10` / `0327f290` were readable at 01:52 and are `Not a valid object name` by 01:55 — an autopilot run armed in this same worktree at 01:50 (`chore(autopilot): arm run pilot-r5`). | UNTESTED — no gate checks that a review's dispatch digests still equal the contract at `reviewed_sha`; the freshness check watches *change after* the sha, not *mismatch at* it. | process: re-dispatch `dotf spec review CLI-131-lessons-index-on-read` from a quiesced branch, then re-run this review; secondarily **tests** — `dotf spec review` should refuse to pin digests that don't match the sha it records. |
| Major | THEORETICAL | blast radius | "`--check` flags a committed index in the other 11 repos (loud)" is recorded as *measured* (`11 repos carry an _index.md and only dotfiles runs lessons fmt`) with **no reproducible command** anywhere in the artifacts. The claim is the whole justification for shipping `--check` as a break; if one of those repos runs `lessons fmt --check` in a hook or CI, it goes red on its next `dotf` upgrade and this spec is the cause. The unit test proves the *checker* finds stray indexes; it cannot prove anything about which repos run the checker. | `proposal.md` Risks; `verification.md` "Decisions made during implementation". Not re-derivable in this review: the other repos are outside the reviewed tree. | `TestStrayIndexes_NamesACommittedIndexAtAnyLevel` (covers the detection, **not** the 11-repo population claim) → the population claim is UNTESTED. | spec: record the enumeration command in `verification.md` (outside the contract set, so it can be fixed without invalidating the next verdict), or ticket the sweep. |
| Minor | REAL | evidence drift | `verification.md` AC1 asserts the render "prints 461 rows for 461 lesson files". Measured at this sha: **463 / 463**. The number rotted because other PRs added lessons after it was written — this repo's own lesson-336 pattern, and the file quotes lesson-464 two lines away. | `ls docs/lessons/lesson-*.md \| wc -l` → 463; `dotf lessons index --dir docs/lessons \| grep -c '^| \['` → 463. | `TestIndex_OneTablePerDirectorySortedByNumber` tests the render, not the prose count → the count itself is UNTESTED (a stale measurement no test can fail on). | spec (`verification.md` — outside the contract set): re-measure, or pin it to a sha ("461 as of 20e23d33"). |
| Minor | REAL | stale pointer | AC7's scope is "reword the stale pointers", and README/lane-prompt/pre-commit were reworded — but the user-facing parent summary still reads **`Keep a repository's lessons in one format with a generated index`**, which is precisely the thing this change removes. A user reading `dotf lessons --help` is told the tool maintains a committed generated index. | Reproduced: `/tmp/dotf-head lessons --help` at head prints that string. | UNTESTED — no test pins the parent command's `Short`; `TestLessonsIndex_PrintsTheTableAndWritesNothing` covers the subcommand, not the parent summary. | code: `cli/internal/cmd/lessons.go` parent `Short`; add it to the help-string test if one is introduced. |
| Minor | REAL | internal contradiction | `tasks.md` "Closing" leaves four boxes unticked while `verification.md` claims their contents: `Type checks pass`, `Lint passes`, `verification.md filled in`, `PR opened referencing this spec folder` — yet `verification.md` **is** filled and records `go build && go vet && GOOS=windows go vet` clean and `golangci-lint run` 0 issues. One of the two files is wrong; the archive step will not notice. | Read of both files at head. | UNTESTED — no cross-check between `tasks.md` ticks and `verification.md` claims. | **contract set** (`tasks.md`) — which is why it is listed and not asked for here: on the next verdict, tick or contradict deliberately. |
| Question | — | lifecycle / WIP | `proposal.md` frontmatter is `status: draft` while every Implementation task is `[x]`, and `wip_override` self-reports **"12 active, limit 10"** twice in one line. Confirm the override is the owner's live decision and not residue from a copied header. | `proposal.md` frontmatter at head. | n/a — owner confirmation. | contract set / owner decision. |

### UNVERIFIED in this window (claimed by `verification.md`, not reproduced here — no reason to doubt them, but they are not this review's evidence)

- `go test ./...` for the **whole** cli module (I ran only the four touched packages: `internal/lessons`, `internal/cmd`, `internal/initrepo`, `internal/mem` — all `ok`, `internal/cmd` 27.7s).
- `golangci-lint run` v2.12.2 "0 issues"; `GOOS=windows go vet`; the `bats` 98-test claim.
- The recorded non-measurement, which `verification.md` states honestly: the pre-commit `lessons-fmt` hook runs the installed 0.72 `dotf` and so is red on a tree with no index until a release carrying this lands (lesson-464). Accepted as disclosed, not as verified.
- Maintainability thresholds (≤40-line functions, CC ≤10): not measured. Context that matters when someone does: `cli/.golangci.yml` sets `gocyclo min-complexity: 22` and **excludes everything outside `internal/cmd/`** from `gocyclo` — so CI lint cannot fail a new outlier in `internal/lessons`.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | **A** | All 7 ACs pass on the spec's own commands and again under independent CLI probes, including the negative paths (duplicate number, dangling wikilink, stray nested index, `fmt` not deleting it). |
| Verification | **B** | Reproducible commands and outputs throughout, but one prose measurement rotted (461→463) and the 11-repo blast radius is asserted with no command. |
| Scope | **A** | Diff is exactly the proposal's seven bullets; deletions, edits and test additions map one-to-one; no unrelated change. |
| Reliability | **B** | `index` writes nothing and is byte-stable across runs; error paths exit 1 naming the file and the fix. Not probed: unreadable-file and permission paths (named test exists, not run individually). |
| Maintainability | **B** | A removal plus one render command, no new large surface; lint/CC/function-length claims and the `gocyclo` exclusion outside `internal/cmd` left UNVERIFIED rather than assumed good. |
| Handoff-readiness | **B** | Spec folder coherent, lesson-464 captured, follow-up explicitly ticketed as #2659; held back by the unticked-against-claimed contradiction in `tasks.md`. |

### Verdict

**FAIL** — one **Blocker** (any reality blocks): this review cannot be certified as bound to the contract that would be archived, because the digests the launcher pinned hash differently from the three contract files at the sha it pinned, under all three plausible hashing forms. Rubric alone would say PASS-WITH-GAPS (no D, one open Major that is THEORETICAL); the Blocker escalates independently, per the skill's "more severe of the two paths".

Do not read this as a defect in the work. The change itself is verified good and I would sign it on a clean re-dispatch.

### Recommended next steps

Cost-ordered; the first is the only one that reopens a round.

1. **Re-dispatch, then re-review, from a quiesced branch.** Let the autopilot run in this worktree settle, confirm the contract files are stable (`sha256sum specs/CLI-131-lessons-index-on-read/{features.json,proposal.md,tasks.md}` twice), then run `dotf spec review CLI-131-lessons-index-on-read` and re-run this review. The code findings below are cheap; nothing needs re-implementing.
2. **Process fix worth its own ticket** (root cause, not a symptom): the reviewer dispatcher should assert that the digests it pins equal the digests at the `reviewed_sha` it records, and refuse otherwise. Today a review can be dispatched against bytes that are already gone and still look well-formed — which is F1's whole class, and it will recur on any branch an unattended session is moving.
3. **Route the tracked gaps to `verification.md`** (outside the contract set, so these do **not** invalidate the next verdict), each applied, ticketed, or declined with a reason: the 461→463 re-measure; the 11-repo enumeration command; the parent-`--help` string that still promises "a generated index" (this one is a **code** fix in `cli/internal/cmd/lessons.go`, and is the only item I'd fix before landing).
4. **Contract-set items — for the next round, not asked for alongside this verdict**: the four unticked-vs-claimed boxes in `tasks.md` Closing, and the `status: draft` / `wip_override: "12 active, limit 10"` frontmatter. Under a FAIL these are the point; state the disposition of the WIP breach explicitly so the owner is the one who accepted it.
5. **Before archive**, close the `verification.md` honesty items it already declares: `proposal.md` frontmatter → `status: archived`, folder → `specs/archive/…`, board item `mlorentedev/dotfiles#2554` → Done, and the #2659 follow-up (vault skill records still saying `fmt` regenerates `_index.md`) filed and linked rather than remembered.
