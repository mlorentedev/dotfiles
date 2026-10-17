---
spec: "CLI-131-lessons-index-on-read"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "2888f9cc3bbadd012238baf8394819c64668d8e0"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-17"
---

## Adversarial review

**Scope**: CLI-131-lessons-index-on-read / head `2888f9cc`, base `ca2f673a65adaf7f4aa8a8b07492a22890324a00` (launcher-provided; verified ancestor of head — `git merge-base --is-ancestor` exit 0). Round 3.
**Sources**: `specs/CLI-131-lessons-index-on-read/{proposal,tasks,verification,features}.md` + `git diff ca2f673a...HEAD` + `harness/reviewer-pool.json`, plus this session's own runs: `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`, `go test ./... -count=1`, all 7 `features.json` verification commands, `golangci-lint` v2.12.2 over the four touched packages with a cold cache, the 14 lessons-related `tests/*.bats` files, the installed `dotf` on the real tree, six mutation edits (all reverted), and `gh issue view` for the tickets the spec leans on.

### Binding check

`review-request.json` records `reviewed_sha: 2888f9cc` (== this review's head) and three contract digests. The contract files on disk are unchanged since before the request: `proposal.md` last touched by `e5889a10` (01:01), `tasks.md` by `ab176f3e` (03:18), `features.json` by `e5889a10`, request timestamped 03:47:07 — so this verdict describes the same contract the archive gate will hash (round 2 reproduced all three normalised digests byte-for-byte; nobody has edited the contract set since). Reviewer id `nan/qwen3.8-flash` is a `signs: first` member of `harness/reviewer-pool.json`; `proposal.md` declares no `risk:`, so a first signature is the whole requirement. No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tag remains in any spec file.

### Spec and task alignment

- The CLI-131 production change is three commits — `e5889a10` (feature), `ab176f3e` (round-1 findings), `2888f9cc` (round-2 findings). Their files map one-to-one onto the proposal's seven bullets: `lessons.Index` + `lessons.StrayIndexes` replacing the index read/write in `cli/internal/lessons/lessons.go`; `dotf lessons index` added and `fmt` reworded in `cli/internal/cmd/lessons.go`; the scaffold template deleted (`scaffold.go`, `templates/lessons-index.md`); staleness via `newestLesson` (`session_start.go`); `docs/lessons/_index.md` deleted (485 lines); pointers reworded in `docs/README.md`, `docs/lessons.md`, `.dotf/lane-prompt.md`, `.pre-commit-config.yaml`.
- **The rest of the stated range is not this change.** `ca2f673a...HEAD` also carries nine other merged PRs (`#2675` pmset, `#2679` pwsh + `dotf lint powershell`, `#2666` lane concurrency, `#2665` spec review-base, `#2663`, `#2669`, `#2676`, `#2687`, `#2608` release 0.73.0), because the launcher's base predates them and this branch now sits at `origin/main` minus one. Verified, not assumed: `git log --oneline origin/main..HEAD` is empty. Scope below is graded on the three CLI-131 commits; that other work is not creep by this change and no finding is written against it.
- Every AC has an `[ACn]` marker in `tasks.md`, a named Go test, and a non-vacuous `features.json` command that greps a `--- PASS: <name>` line under `-count=1 -v`, so an empty `-run` filter cannot fake a pass. All seven re-ran green in this session (f1–f7 PASS, exit 0).

### Independent verification (this session, not quoted from verification.md)

- `go build ./...` and `go vet ./...` clean; `GOOS=windows go vet ./...` clean.
- `go test ./... -count=1`: 44 packages `ok`, no `FAIL` line anywhere in the output.
- `golangci-lint run` v2.12.2 (the pin) with a cold cache over `internal/lessons`, `internal/cmd`, `internal/mem`, `internal/initrepo`: **0 issues**. (A warm-cache run printed a `forbidigo` warning sourced from a *sibling worktree* path that no longer exists — `../../fix-lessons-index-test-strength/...` — a stale cache artifact, not an issue in this tree; `.golangci.yml` excludes `_test.go` from forbidigo, so the `runtime.GOOS` skip guard in `TestIndex_ReportsAnUnreadableCategoryDirectory` is legal.)
- `bats` over the 14 `tests/*.bats` files that mention lessons: 228 tests, `ok` on all, none failing — reproduces the line in `verification.md`.
- Real tree, current sha: `dotf lessons index --dir docs/lessons | grep -c '^| \['` = **470**, `ls docs/lessons/lesson-*.md | wc -l` = **470** (the re-measure instruction `verification.md` gives, honoured; the earlier 463/464 counts were correct at their own shas).
- Installed hook: `dotf --version` → `0.73.0`; `dotf lessons fmt --check --dir docs/lessons` → `[OK] lessons are in their canonical form`, exit 0.
- Mutations, each reverted, tree verified clean after (`git status --porcelain -- cli` empty):

| Mutation | Named test that went red |
|---|---|
| delete `sort.Slice` in `renderTable` | `TestIndex_SortsByNumberNotByFileName` |
| drop the `\|` escaping in the rendered row | `TestIndex_EscapesAPipeInATitleAndRefusesAnUnreadableLesson` |
| `StrayIndexes` restricted to the root directory | `TestStrayIndexes_NamesACommittedIndexAtAnyLevel` |
| remove the stray-index report from `--check` | `TestLessonsFmt_CheckNamesACommittedIndexAndFmtLeavesItAlone` |
| `newestLesson` root-only glob (drop `*/lesson-*.md`) | `TestLessonsStaleness/one_fresh_lesson_keeps_it_silent,_whichever_directory_holds_it` |
| restore "generated index" in the parent `Short` | `TestLessonsHelp_DoesNotPromiseACommittedIndex` |

- **Round 2's Major is closed.** Its mutation 1 reported that the sort-blind test still passed at `484070a3`; at this sha the fix `TestIndex_SortsByNumberNotByFileName` (`lesson-1000` vs `lesson-101`, where file-name order inverts numeric order) exists and is the test that goes red when the sort line is deleted. The AC1 sort property is now genuinely pinned.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|---|---|---|---|---|---|
| Minor | REAL | `StrayIndexes` depth vs spec wording | `proposal.md` promises `--check` reports a committed `_index.md` **anywhere under the lessons directory**; `StrayIndexes` walks `lessonDirs`, which is the directory plus its **first-level** entries only, so `docs/lessons/category/sub/_index.md` is not reported. The function's own doc comment ("names every committed `_index.md` under dir") and the test name (`…AtAnyLevel`) overclaim the same way. | Reproduced this session: synthetic tree `/tmp/c131tree` with `_index.md` at root, level 1 and level 2 → `dotf lessons fmt --check` named the first two and exited 1, silent on the third. Blast radius measured as zero: `find ~/Projects -maxdepth 6 -path '*/docs/lessons/*/*/_index.md'` and `find … -type d -path '*/docs/lessons/*/*'` both print nothing, and `git log --all --diff-filter=A -- 'docs/lessons/*/*/_index.md'` is empty — neither the old generator nor `dotf init` ever wrote below level 1, and dotfiles' `docs/lessons/` has no subdirectories at all. Depth-2 lessons are invisible to `Plan`, `Index` and `StrayIndexes` identically, so the tool is at least self-consistent. | Level ≥2 is **UNTESTED** (`TestStrayIndexes_NamesACommittedIndexAtAnyLevel` passes while the promise is broken there) | code (`filepath.WalkDir` for stray detection, or narrow the promise) + tests (level-2 case). The wording lives in the contract set, so under this verdict it is dispositioned in `verification.md`, not edited here. |
| Minor | REAL | stale pointer the proposal said was reworded | `.github/workflows/repo-hygiene.yml:3-4` still reads "…and generated index (`dotf lessons fmt --check`, run from the checkout…)" and its step name on line 28 is still "Lessons in their canonical form, index generated from them". `proposal.md` → *Out of scope* says of the two files' comments: "Only their comments mention a generated index, and those are reworded." The `.pre-commit-config.yaml` comment and hook name were reworded; the workflow was not. | `grep -n -B4 -A8 lessons .github/workflows/repo-hygiene.yml` in this session; the change is absent from the diff (`git diff --stat` lists no workflow file). Already ticketed: `gh issue view 2659` → **OPEN**, "CLI-132: lessons index references left after CLI-131 (skill records, workflow comment, lane-prompt note)". | UNTESTED (nothing pins CI prose) | code (the workflow comment + step name — outside the contract set, so it can land after this verdict) |
| Minor | REAL | instruction/tool conflict in the skill records | Seven `harness/skills/*/SKILL.md` still tell an agent that `dotf lessons fmt` regenerates `docs/lessons/_index.md`. `harness/skills/place-knowledge/SKILL.md:46-47` goes further and tells an onboarding agent to **create** `docs/lessons/_index.md` — a file the new `--check` now reports as drift, so a `/place-knowledge` run following the text produces the red build the change was meant to stop. | `grep -rln 'dotf lessons fmt.*_index' harness/skills/` → architecture-session, handoff, spec, autopilot, crystallize (plus `insights`/`place-knowledge` from the earlier pointer grep at lines 84 and 46-47). Tracked under the same #2659 (OPEN); these records are generated from the vault, hence the owner-lands path. | UNTESTED | vault + code (regenerate the harness skill records) |
| Minor | REAL | `verification.md` understates its own evidence, and names a version that is not the pin | The line "Not measured: the pre-commit `lessons-fmt` hook, which runs the installed 0.72 `dotf` and so fails on a tree with no index until a release carrying this change is installed (lesson 464)" is no longer true and *is* measurable: the pin is `DOTF_VERSION=0.73.0` (`versions.conf:180`), the installed binary reports `dotf version 0.73.0`, `e5889a10` is an ancestor of the `v0.73.0` tag (`git tag --contains e5889a10` → `v0.73.0`; `git merge-base --is-ancestor e5889a10 cdf425af` → 0), and the hook's own command exits 0 on this tree. | `dotf --version`, `versions.conf`, ancestry checks and the hook command, all run in this session. | n/a — this is an evidence-artifact finding, and the hook path has no committed test by design (the hook runs the *installed* release, which CI cannot pin) | verification.md (outside the contract set) |
| Minor | SPECULATIVE | render fidelity | `renderTable` escapes `|` unconditionally, so a title that already carries `\|` becomes `\\|` (double escape). Cosmetic; GitHub renders `\|` as a literal bar either way. | Measured on the real tree: `dotf lessons index --dir docs/lessons \| grep -c '\\\\\|'` → 0, i.e. no lesson title currently hits it; two titles with a bare `|` (302, 398) escape correctly. | `TestIndex_EscapesAPipeInATitleAndRefusesAnUnreadableLesson` covers the single-escape path | — (surface only; do not gate) |
| Question | REAL | board state (Definition of Done §3) | `mlorentedev/dotfiles#2554`, the issue this spec is gated on, is still **OPEN** while the change itself is merged to main (`e5889a10` ⊂ `v0.73.0`), and the `verification.md` archive checklist line "Bitácora board ticket … closed with PR link" is unticked. Closing at archive is the normal order, so this is a confirmation, not a defect. | `gh issue view 2554 --json state` → `OPEN`, live query in this session. | n/a | owner action, not code |
| Question | REAL | contract hygiene, left to the owner | `proposal.md` frontmatter is still `status: draft` with a `wip_override:` describing the *archived* CLI-128 override. Rounds 1 and 2 both recorded this as the owner's call; `dotf spec archive` sets `status` itself, and touching `proposal.md` now would invalidate this verdict. Not re-litigated. | file read; unchanged in the diff since `e5889a10`. | n/a | spec (owner's decision, at archive time) |

Not a finding, but the next round should not re-derive it: `#2690` ("`newestLesson` drops I/O errors, so an unreadable `docs/lessons/` silences the staleness warning"), raised by round 2, is **OPEN** (`gh issue view 2690`) and correctly out of this PR. `filepath.Glob` on those two fixed patterns cannot itself error, so the surviving hole is the ignored `os.Stat` error — the ticket is the right home.

### Rubric evidence notes

- **Correctness** — all 7 ACs verified by their own commands *and* by independent probes at this sha; negative paths (unreadable category directory, duplicate number across directories, dangling wikilink, pipe in a title, `fmt` must-not-write) all have named tests. Deducted to B for the level-2 stray-index gap, which is a promise the code does not keep.
- **Verification** — reproducible commands with the re-measure recipe (`grep -c '^| \['` vs `ls | wc -l`), named tests per AC, mutation evidence. Deducted for the stale hook line above.
- **Scope** — 1:1 against the proposal for the three commits; two promised comment rewords are missing (tracked as #2659).
- **Reliability** — `readDir` uses `os.ReadDir` rather than `filepath.Glob` so an unreadable directory cannot silently drop rows; `Index` reports a non-normalisable lesson as an error instead of a missing row; every `fmt` failure path is wrapped as `lessons fmt: <err>`; `--check` exits non-zero rather than warn.
- **Maintainability** — small functions, comments that say WHY (the `h1PrefixRE`, the `dirs[:1]`-style depth convention, the `//nolint:gosec` rationale); cold-cache lint 0 issues; `gocyclo` cap 22 applies to `internal/cmd` and `newLessonsFmtCmd` clears it.
- **Handoff-readiness** — rounds 1 and 2 dispositioned in `verification.md` with per-finding outcomes, lesson 464 captured in `docs/lessons/`, both follow-ups ticketed and verified OPEN. Deducted for the still-open #2554 and the archive checklist lines left unticked.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All 7 ACs verified green at `2888f9cc` + six mutations each caught by a named test; one promise (`--check` "anywhere under") holds only to depth 1. |
| Verification | B | Named tests and reproducible commands per AC, all re-run here (44 pkgs ok, 7/7 features, 228 bats ok); one stale "not measured / 0.72" line understates the hook evidence. |
| Scope | B | Three commits map 1:1 to the proposal; the CI workflow comment and seven skill records still say "regenerates the index" (tracked #2659); the rest of the range is other merged PRs, verified by `origin/main..HEAD` being empty. |
| Reliability | B | I/O errors surface instead of silently dropping rows; failures wrapped; `--check` fails closed; the ignored `os.Stat` error in `newestLesson` is ticketed (#2690). |
| Maintainability | B | Small well-commented functions, lint 0 issues, no dead index-write code left; the `AtAnyLevel` test name and the `StrayIndexes` doc comment overclaim what depth is covered. |
| Handoff-readiness | B | Round-1/2 dispositions, lesson 464, follow-ups filed and OPEN; board issue #2554 still open and archive checklist unticked at review time. |

Aggregation: no D, no C → all B or above → **PASS** on the rubric path. No Blocker and no REAL Major in the findings; four Minors (three REAL-and-tracked, one SPECULATIVE) plus two Questions. Both paths agree on a passing verdict; it is recorded as **PASS WITH GAPS** because the gaps are tracked rather than fixed — F1's level-2 coverage is UNTESTED and F2/F3 leave durable text stale until #2659 lands.

### Verdict

**PASS WITH GAPS**

`dotf spec archive` **is advisable** in the current state: this frontmatter is machine-readable, `reviewer` is spelled exactly as the pool's `nan/qwen3.8-flash` entry, `reviewed_sha` is the examined head, and the contract set (`proposal.md`, `tasks.md`, `features.json`) is untouched by this review, so the staleness check stays green. Close #2554 as part of the archive step (DoD §3) — the ticket is still open while the code is already released in 0.73.0.

### Recommended next steps

Contract set is **closed** by this verdict; nothing below asks for an edit to `proposal.md`, `tasks.md` or `features.json`.

- **In `verification.md`** (outside the staleness check, for the implementer to apply / ticket / decline with a reason):
  - Replace the "Not measured … installed 0.72 `dotf`" line with the measurement: pin `DOTF_VERSION=0.73.0`, `dotf version 0.73.0`, `e5889a10 ⊂ v0.73.0`, and `dotf lessons fmt --check --dir docs/lessons` → exit 0 (F4).
  - Record the depth convention explicitly: `lessonDirs` = the directory plus first-level categories, and `StrayIndexes` follows it, so a level-2 committed index is not drift. Either state that as the accepted limit, or point at the widening (F1).
  - State that `--check` is green with the *installed* release, and lesson 464's window is closed by `v0.73.0`.
- **In code** (no contract impact, safe to land after this verdict):
  - Widen `StrayIndexes` to `filepath.WalkDir` (or rename it and its test to what it covers, `…AtRootOrFirstLevel`) and add the level-2 case to `TestStrayIndexes_NamesACommittedIndexAtAnyLevel` so the UNTESTED path gets a named test (F1).
  - Reword `.github/workflows/repo-hygiene.yml:3-4` and its step name on line 28 — already inside #2659's scope (F2).
- **Owner-lands (vault)**: the seven `harness/skills/*/SKILL.md` records, `place-knowledge` first, since its instruction to create `docs/lessons/_index.md` now produces a red `--check` (F3, #2659).
- **Board**: #2554 closed with the archive; keep #2659 and #2690 as the two open tails; tick the `verification.md` archive checklist as each step actually happens.
- **For the next round, if there is one**: re-record `reviewed_sha`; grade scope on the CLI-131 commits, and re-run the six mutations in the table above rather than trusting this list — the sort blindness that cost round 2 a cycle is exactly the property a mutation pins.
