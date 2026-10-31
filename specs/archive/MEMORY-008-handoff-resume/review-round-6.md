---
spec: "MEMORY-008-handoff-resume"
verdict: "FAIL"
reviewed_sha: "0ecee6e371b0696c38d5e6df690f17e1e388e3d0"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-30"
---

## Adversarial review

**Scope**: MEMORY-008-handoff-resume — round 6 (rounds 1-5 are on file as `review-round-1..5.md`).
**Sources**: `specs/MEMORY-008-handoff-resume/{proposal,tasks,verification,features}.md`; `git diff e1c5754c604c45fb8d236694bf7a2a916e80cfde...HEAD` restricted to the sixteen commits that touch the spec folder (`git log --oneline e1c5754..HEAD -- specs/MEMORY-008-handoff-resume`), i.e. `cli/internal/mem/{resume,session_start_resume,thread,thread_store,shellword}.go`, `cli/internal/cmd/mem_resume.go`, their tests and goldens, `harness/enforced/pr-stewardship.md`, `harness/skills/{handoff,catchup}/SKILL.md`, `session-start-config.json`.

The launcher's base resolves to an 811-commit range (2,566 files, 179k insertions) that includes unrelated main history, exactly as round 1 flagged. I did not substitute a base; I reviewed the spec's own commits and treated the rest of the range as unauditable rather than reviewed. That limit is recorded under Scope and not charged to the change.

**Verification performed in this session:**
- `cd cli && go vet ./internal/mem/... ./internal/cmd/` → exit 0, no output.
- `cd cli && go test ./internal/mem/... ./internal/cmd/` (whole packages, unfiltered) → `ok …/internal/mem 0.576s`, `ok …/internal/cmd 21.822s`.
- Mutation, reverted (`ShellWord` returning its input unquoted, `go vet`-clean): `go test -run Quote` turns **`TestSessionStartResumeQuotesTheOneLiner`** and **`TestMemResumeQuotesTheCandidateLine`** red — round 5's mutation claim is accurate, not asserted.
- `ShellWord` behaviour observed directly: `"-x"`→`-x`, `"--thread=1"`→`'--thread=1'`, `"x;echo"`→`'x;echo'`, `"a b"`→`'a b'`, `"plain/key"`→`plain/key`.
- `dotf spec digest MEMORY-008-handoff-resume` → contract digests `features.json 358ebb49…`, `proposal.md 9e3de542…`, `tasks.md a83b8a48…`, byte-identical to `review-request.json`'s `contract_digests`. **Not stale.**
- `./scripts/compile-harness.sh --check` → exit 0, `[check] OK: no harness drift`.
- AC5 command (f5): `grep -c 'dotf mem resume' harness/skills/handoff/SKILL.md` → 2; `awk '/^### Step 1/,/^### Step 2/' harness/skills/catchup/SKILL.md | grep -c 'dotf mem resume'` → 1. Exit 0.
- AC6 command (f6, first two legs): `grep -rl 'dotf mem resume' harness/enforced/` → `harness/enforced/pr-stewardship.md`; `compile-harness.sh --check` exit 0. The `bats --filter` leg was **not** run (time); UNVERIFIED.
- `session-start-config.json` declares `"handoff_resume": {"enabled": true, …}`, and `tests/session-start-config.bats:30` counts it — round 5's #2915 follow-up landed.
- Mutation probe of my own (temp test, removed afterwards; working tree left clean): a thread carrying a canonical label **and** its alias loses the alias field's content — see M-1.

### Spec and task alignment

- AC1-AC6 are all ticked in `proposal.md`, all six have a `features.json` entry with a non-vacuous command, and `verification.md` carries per-slice named tests plus mutation tables. This is the strongest evidence trail of the six rounds; the round-5 dispositions (F-1 quoting, the #2915 declaration) are visibly landed, not claimed.
- `tasks.md` has one box left open — "Independent review before archiving" — which is this run. Correct state.
- **Mismatch found (M-1):** the proposal's own migration premise ("The labels sessions already improvise … are read as aliases, so no existing block needs migrating by hand") is only half true at render time. `ParseThread` reads both the canonical label and the alias into two fields, but `renderContext`/`renderPrompt` render only the *first* field per label and drop the rest with no warning. A block that has adopted `**Verify:**` without deleting an older `**Verify at start:**` therefore resumes with the older Verify commands silently missing — in both formats.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Major | REAL | correctness / data loss | **M-1** A thread carrying a canonical label *and* its alias loses the alias field entirely. `ParseThread` keeps both (e.g. two fields labelled `Verify` / `Awaiting Manu`), but `renderContext` selects fields via `th.Get(label)` + `isFirstField` and skips every later field with a first-field label; `renderPrompt` does the same with three `Get` calls. No warning is printed and neither field's absence is visible to the next session. Reproduced in-session with a throwaway test: input `**Verify:** \`go test ./a\`` + `**Verify at start:** \`gh pr view 1\`` renders only the first; `**Awaiting Manu:** - one` + `**Judgment calls left open:** - two` renders only `- one`, in both `context` and `prompt`. Same class as round 2's Major (a `Verify` field's content dropped), fixed then for a *within-field* case and left open for the *duplicate-field* case. | `cli/internal/mem/resume.go` (`renderContext`, `renderPrompt`, `isFirstField`); `cli/internal/mem/thread.go:24-27` (aliases); probe output in this review's session | UNTESTED (`TestResumeRendersVerifyAndAwaiting`, `TestParseThreadReadsTheImprovisedLabelsAsTheirFields` each use a *single* spelling only; `grep` for the alias strings finds only `thread.go` and `thread_test.go:53`) | code + tests |
| Minor | THEORETICAL | injection-adjacent rendering | **m-1** `ShellWord`'s plain charset (`^[A-Za-z0-9_./@+-]+$`) accepts a leading `-`, so a thread key such as `-x` renders bare — observed: `ShellWord("-x")` returns `-x` — and the printed `dotf mem resume --thread -x` is then read by cobra as a flag, not a key, so the candidate line cannot be run as printed. Reachable: `threadFileSafe` also accepts a leading `-`, and a `### thread: -x` heading in a MEMORY.md this session did not write needs no writer at all. Not an execution risk (nothing is invoked; the command errors), but the quote-or-not decision should not be left to a charset that admits flag syntax. | `cli/internal/mem/shellword.go`; `cli/internal/mem/thread_store.go:39`; probe output above | UNTESTED | code + tests |
| Minor | THEORETICAL | observability | **m-2** `renderContext` silently omits a `Next action` that is present-but-blank, while `prompt` says "none recorded: ask what to work on". `handoff-write` warns at write time (`ThreadWarnings`), but a block written by another tool or an older release reaches the session with no next action and the context render says nothing about it. | `cli/internal/mem/resume.go:70-83`; `thread.go:ThreadWarnings` | UNTESTED | code |
| Question | — | verification | **q-1** `features.json` states are all `pending`, so `dotf spec archive`'s features leg reads six unpassed criteria on a branch whose six commands exit 0. Declined in round 4 as harness-owned, and still pending at archive time; whoever archives should decide whether the harness owns flipping them or the archive is expected to pass with pending states. | `features.json` `state` fields; `dotf spec digest` output above | n/a | spec (features.json — harness-owned) |
| Question | — | live behaviour | **q-2** Carried from rounds 2-5 and still open, not re-litigated here: a live `source=compact` SessionStart payload has never been observed end to end. The goldens and `normaliseSource` are sound; the matcher firing on a real compaction is the unproven link. | `verification.md` (round 2/3/4/5 tables); `session_start_resume_test.go` | `TestSessionStartResumeBySource` (unit only) | n/a (observation, then a ticket if it does not fire) |
| Question | — | scope | **q-3** The launcher's base yields 811 commits / 2,566 files, of which 16 commits touch this spec. Unrelated-code review did not happen and could not within the cap; this review's verdict is about the spec's commits. The base resolution is the harness's, not this change's, so it is recorded and not charged. | `git log --oneline e1c5754..HEAD \| wc -l` → 811; `-- specs/MEMORY-008-handoff-resume` → 16 | n/a | harness (launcher base resolution) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | C | Acceptance criteria hold on the fixtures, but a reproduced data-loss path (M-1) drops fields on an input the proposal explicitly says exists. |
| Verification | B | Named tests, golden files and reverted-mutation tables per slice, plus a fresh digest match; the gap that M-1 exposes has no named test, and one f6 leg was not run this round. |
| Scope | B | The spec's sixteen commits are coherent and on-topic; the 811-commit stated range makes overall scope unauditable (q-3) and three unrelated `harness/skills` records were rewritten by `--refresh` (declared, #1502). |
| Reliability | C | A silent, warning-free omission on a plausible input; error paths elsewhere (absent memory, unknown format, another agent's thread, disabled injector) are handled and tested. |
| Maintainability | B | `ShellWord`/`ShellQuote` are single-sourced in `package mem`, the retired read-only allowlist (`readOnlyCommand`, lesson 513) is gone rather than patched, functions sit well under 40 lines and CC 10; the duplicate-label filtering is the one piece whose rule is implicit. |
| Handoff-readiness | B | `verification.md`, ADR-056 and lessons 507/508/513 record the builds and the decisions; round 6's dispositions are not yet written into `verification.md`. |

### Verdict

FAIL

One **REAL Major** (M-1, reproduced) that is **UNTESTED**, so it cannot be closed by claim alone. Rubric minimum is C (Correctness, Reliability) with no D, which alone would have been PASS WITH GAPS; the severity axis escalates past that.

### Recommended next steps

All of these are outside the contract set (`proposal.md`, `tasks.md`, `features.json`) — the digests above are current and must stay so, or this verdict goes stale. Disposition each in `verification.md` (applied / ticketed / declined with a reason), as rounds 1-5 did.

- **M-1 (code + tests):** choose one rule and make it total. Either render *every* field in body order under its own label (so two `Verify` fields both appear, each under the note), or collapse the duplicates into one field at parse time — joining canonical-before-alias — and warn via `ThreadWarnings` that two spellings of one label were merged. Add a named regression test carrying both `**Verify:**` and `**Verify at start:**` (and both `Awaiting Manu` spellings) through `RenderResume` in both formats; the current probe is the red form of it. Then re-review.
- **m-1 (code + tests):** refuse a plain word that begins with `-` in `ShellWord`, so a flag-shaped key is quoted; add a case to `TestMemResumeQuotesTheCandidateLine`.
- **m-2 (code):** render an explicit "no next action recorded" line in `context` as `prompt` does, or leave it and record the decline — it is the one field whose absence the next session cannot see.
- **q-1 (spec, harness-owned):** state in `verification.md` who flips `features.json` states before `dotf spec archive`, so the archive does not discover six `pending` legs on a green branch.
- **q-2:** keep the live-compaction observation ticket open; do not treat the unit golden as proof the hook fires.
- **Archive:** not advisable in this state. The minimum that flips FAIL to PASS is M-1 fixed with its named test, the new digests captured, and one more round.
