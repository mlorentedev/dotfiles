---
spec: "MEMORY-018-handoff-line-budget"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "27e4d097c0c33b1ed9ffbb40deebbfccbede5136"
reviewer: "nan/deepseek-v4-flash"
date: "2026-11-12"
---

## Adversarial review

**Scope**: MEMORY-018-handoff-line-budget (PR #3394, head `27e4d097`), diff range
`b4e9aa420e2f26b8b0195e9866fda01f3c54b360...HEAD` — one commit on the branch,
`27e4d097 fix(mem): refuse a handoff write that takes MEMORY.md past 200 lines`.
The range holds no merge and no other work.

**Sources**:
- `specs/MEMORY-018-handoff-line-budget/{proposal,tasks,verification}.md`, `features.json`,
  `review-request.json` (contract digests re-derived with `dotf spec digest
  MEMORY-018-handoff-line-budget` — they match the recorded `contract_digests`, so the
  contract is unchanged since the request was made).
- Code under review: `cli/internal/mem/thread_store.go` (+35/−7),
  `cli/internal/mem/thread_store_test.go` (+69).
- `cli/internal/cmd/mem_handoff.go` (caller, unmodified), `docs/adr/adr-056-*`,
  `harness/skills/handoff/SKILL.md`, PR #3394 body.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec folder.

### Spec and task alignment

- **AC1** (`checkable: a sub-25,000-byte MEMORY.md taken past 200 lines is refused, naming
  count, cap and the oldest threads`): implemented in `CheckMemoryBudget`'s second `switch`
  case, message built by the shared `budgetRefusal`. Proven by
  `TestCheckMemoryBudgetRefusesAWritePastTheLineCap` (the test asserts the fixture is under
  the byte budget, so only the line budget can fire) and reproduced end-to-end by me: a
  synthetic 217-line, 3,858-byte `MEMORY.md` built beside a `threads/` dir, written with the
  built binary, exits 1 with *"this write takes MEMORY.md to 224 lines, past the 200 a session
  start loads … Least recently updated: t0 (2026-09-01) … t28 … t1 …"* and leaves the file at
  217 lines.
- **AC2** (`a write that holds or lowers the line count of a file already past the cap
  passes; a file at or under 200 lines passes`): the growth guard `lineCount(after) >
  lineCount(before)` plus `> MemoryLineBudget`. Covered by
  `TestCheckMemoryBudgetLetsAFilePastTheLineCapShrink` (lowered count, held count with more
  bytes, and the refused growth) and `TestCheckMemoryBudgetPassesAFileAtTheLineCap` (exactly
  200 passes; 200 plus an unterminated final line is refused).
- **AC3** (`the byte budget still refuses on its own terms`): the byte case is
  `len(after) > MemoryByteBudget && len(after) > len(before)`, a literal restatement of the
  previous early-return, and the headline composes to the byte-identical message. Covered by
  the new `TestCheckMemoryBudgetStillRefusesPastTheByteBudget` plus the untouched
  `TestCheckMemoryBudget`, `TestTheBudgetRefusalSuggestsOnlyDatedThreadsAndHowToArchiveEach`
  and cmd-level `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget`.
- **Out of scope respected**: no change to the byte budget's threshold, to the thread store's
  entry shape, and no on-write archiving (#3393 stays out). The diff touches two code files
  and the spec folder; nothing else.
- **Contract set untouched by me**: `proposal.md`, `tasks.md`, `features.json` unmodified, so
  this review is not self-stale.
- `tasks.md` closing box (`verification.md` filled in; independent review; archive) is
  correctly still `[ ]` — this review is that step.

### Verification I ran (commands and results, this session)

- `cd cli && go build ./... && go vet ./...` → build OK, vet OK.
- `GOOS=windows go vet ./...` → clean; `golangci-lint run ./...` → `0 issues.`;
  `GOOS=windows golangci-lint run ./internal/mem/... ./internal/cmd/...` → `0 issues.`
- `go test ./... -timeout 10m` in `cli/` → every package `ok` (58 packages, incl.
  `internal/mem` 1.7 s and `internal/cmd` 105 s).
- Red-green, four mutations of `thread_store.go`, each reverted (`git status` clean after):
  dropping the line case reddens 3 line tests; `>` → `>=` on the cap reddens the at-cap test;
  dropping `lineCount`'s unterminated-tail increment reddens the 201-line case; dropping the
  growth guard reddens both halves of the shrink/hold test. The tests do exercise the guard,
  not merely sit beside it.
- Scope of the guard's reach: `CheckMemoryBudget` has exactly one caller
  (`cli/internal/cmd/mem_handoff.go:130`), inside the handoff lock and before either file is
  written (`TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget` asserts the refused write
  leaves MEMORY.md and `threads/` untouched). The only other writer of MEMORY.md,
  `ArchiveAndPruneThread`, only removes a thread's entry, so it cannot grow the file past a
  budget — no bypass.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | CLI help / user-facing contract | `dotf mem handoff-write --help` still documents only the byte refusal and still calls the byte budget *"what a session start reads"* — the very claim this change refines into *"the other half of what a session start reads"*. A user whose first line refusal arrives in a terminal has no help text explaining it. | `--help` output from the built binary, second paragraph; `cli/internal/cmd/mem_handoff.go:36-42` | UNTESTED — `help_smoke_test.go` pins only that help renders (`TestEveryCommandHelpRenders`) and that `docs/…md` refs exist (`TestHelpDocReferencesExist`); nothing pins the budget wording | code (help text); outside the contract set |
| Minor | REAL | ADR / knowledge | `docs/adr/adr-056-a-handoff-thread-is-stored-in-its-own-file.md` is the ADR that owns this contract, and its Context ("A session start reads 25,000 bytes of that file and drops the rest") and Decision 7 ("A write that grows `MEMORY.md` past 25,000 B is refused") now describe one of two refusal rules. PR #3394's `## Knowledge` declares `ADR: none: extends the #1477 budget, no new decision`; a second, independently-triggered cap in the same function is a contract change a reader of Decision 7 will not find. I am flagging the declared `none`, not a silent omission. | `docs/adr/adr-056-…md:18,30`; `dotf`-free grep shows the only remaining byte-budget prose outside the spec is this ADR and the help text above | UNTESTED — no test or gate compares ADR text to the constant | spec/docs (repo `docs/adr/`, outside the contract set) |
| Minor | THEORETICAL | tests / precedence | `verification.md` records a decision — *"The byte check runs first. A write past both budgets gets the byte message"* — and no test pins it: swapping the two `switch` cases leaves `go test ./internal/mem ./internal/cmd -run 'TestCheckMemoryBudget\|TestTheBudgetRefusal\|TestHandoffWrite'` fully green (probe run, reverted). Impact is the message text only (both orders refuse, and AC3 still holds), which is why this is a tracked gap rather than a defect. | Mutation probe above; the both-crossed case is otherwise untested | UNTESTED — the gap *is* the finding | tests (one case crossing both budgets, asserting the byte wording) |
| Minor | REAL | spec artifacts (`verification.md`) | `verification.md` has no `## Promotion candidates` section, which `dotf spec archive`'s third pre-flight requires and no flag skips: `CheckPromotions` returns *"verification.md has no \"## Promotion candidates\" section"*, and the section is in the scaffold template (`cli/internal/spec/templates/verification.md`) and in 266 of the 311 archived specs. As written, this spec cannot archive whatever this review says. | `cli/internal/spec/promotion.go:52-62`; `specs/MEMORY-018-handoff-line-budget/verification.md` (only Evidence, Test status, Decisions); `dotf spec archive --help` third pre-flight | UNTESTED — the absent-section branch has no test (`TestPromotions*` covers the placeholder, a bare `no`, and the `yes` paths) | spec (`verification.md`, outside the contract set) |
| Question / assumption | REAL | operations | With the dotfiles MEMORY.md at 217 lines, every handoff write that adds a line now fails until at least four threads are archived by hand — and `harness/skills/handoff/SKILL.md:183` forbids editing MEMORY.md, so the workflow has no fallback while #3393 is open. This is declared and accepted in `proposal.md` → *Risks* ("Resolved: accept"), and reproduced by my 217-line probe, so I am not re-litigating it: it needs a disposition, not a fix. | 217-line probe (exit 1, three threads named); PR #3394 *Heads-up* | n/a — runtime behaviour, reproduced | verification.md disposition (outside the contract set); mechanism tracked as #3393 |

No Blocker and no REAL Major found. Areas I attacked and could not break: the both-budgets
interaction (`len` growth vs. line growth is evaluated per measure, so a file over one budget
can still be rewritten in place in the other); a WRITE that shrinks bytes but grows lines
(`24000 B/210 lines` from `26000 B/190 lines` → line refusal, correct); a write that crosses
both budgets; the `\r\n` case (`strings.Count` on `"\n"` is line-ending agnostic); the
unterminated final line (counted, per the spec's resolved definition); and `--dry-run` (refused
too — identical pre-existing behaviour for the byte budget, not a regression).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | AC1–AC3 each verified by a named test and by an end-to-end probe; negative paths (at-cap, unterminated, shrink, hold, growth, both budgets) covered; mutation-verified; no defect observed. |
| Verification       | B | Evidence is reproducible and names the tests, but `verification.md`'s smoke test is prose over a copy of the real vault file (not reproducible from this repo) and one recorded decision (byte-vs-line precedence) has no test behind it. |
| Scope              | A | Diff is the proposal exactly: two code files plus the spec folder; the deferred archiving (#3393) and the byte budget are untouched. |
| Reliability        | A | Refusal happens under the handoff lock and before either file is written (nothing left behind on refusal); the guard is a pure function; no new error path can fail open. |
| Maintainability    | A | Named constant with the WHY in its comment, shared `budgetRefusal` instead of duplicated message assembly, `lineCount` isolated and documented, flat `switch`, no dead code; `golangci-lint` 0 issues. |
| Handoff-readiness  | B | Spec artifacts complete and aligned, PR body carries a `## Knowledge` section with all three lines; the CLI help and ADR-056 that describe this contract were not brought along. |

### Verdict

PASS WITH GAPS

No Blocker, no REAL Major (`severity × reality` keeps this out of FAIL), rubric has no C and no
D. The four findings are tracked, each with a disposition line below; none blocks the archive.

### Recommended next steps

Every item below is **outside the contract set** (`proposal.md`, `tasks.md`, `features.json`), so
none of them invalidates this verdict — do not edit those three files to satisfy this list.

1. **Disposition the five findings in `verification.md`** (applied / ticketed / declined with a
   reason) — that file is excluded from the staleness check for exactly this purpose.
2. **CLI help** (code, follow-up commit or ticket): extend the `handoff-write` `Long` so the
   refusal list reads as two budgets and drop the claim that 25,000 bytes alone "is what a
   session start reads". No test currently pins that paragraph.
3. **ADR-056** (repo `docs/adr/`, follow-up PR): amend Context and Decision 7 (or add
   Decision 8) to record the 200-line cap as a second refusal rule with the same
   shrink-or-hold allowance. If the owner prefers otherwise, record that reasoning instead of
   `ADR: none`.
4. **Precedence test** (tests): one case crossing both budgets, asserting the byte wording —
   otherwise the decision recorded in `verification.md` is advice, not a contract.
5. **`verification.md` Promotion candidates** (spec artifacts, outside the contract set): add the
   section the archive gate requires, with at least one answered line — `yes: docs/adr/…` or
   `no: <reason>`.
6. **Owner action before the release ships**: archive four or more of the oldest dated threads
   from the 217-line dotfiles `MEMORY.md` (the refusal names them), so handoffs are not blocked
   on the first session after upgrade; #3393 remains the mechanism.

**Is `dotf spec archive` advisable?** The verdict is passing, so the review pre-flight is
satisfied — but the archive will still **refuse** until `verification.md` gains its
`## Promotion candidates` section (finding 4), a `verification.md` edit outside the contract set
that therefore does not stale this review. Add that section with at least one answered candidate
line (the ADR-056 amendment as `yes:`, or `no: <reason>`), then archive. Items 1–3 and 6 are
follow-ups that need not precede it; items 2 and 3 can equally ride a follow-up PR, since this
branch's contract files are untouched by them.
