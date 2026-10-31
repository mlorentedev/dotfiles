---
spec: "MEMORY-008-handoff-resume"
verdict: "PASS"
reviewed_sha: "b08c866593b4dcf36e3a431ab8078b1261a94fbd"
reviewer: "nan/glm5.3-flash"
date: "2026-10-31"
---

## Adversarial review

**Scope**: MEMORY-008-handoff-resume (round 7), diff `e1c5754c604c45fb8d236694bf7a2a916e80cfde...b08c866593b4dcf36e3a431ab8078b1261a94fbd`
**Sources**: `specs/MEMORY-008-handoff-resume/{proposal,tasks,verification}.md`, `features.json`, `review-request.json`; the spec's own surface (rounds 1–6 recorded in `verification.md` and `review-round-{1..6}.md`); code `cli/internal/mem/{thread,thread_store,resume,session_start_resume,shellword}.go`, `cli/internal/cmd/mem_resume.go`, `session-start-config.json`, `harness/skills/{handoff,catchup}/SKILL.md`, `harness/enforced/pr-stewardship.md`.

**On the diff range**: the launcher's base spans 811 commits / 2,581 files, so the raw range carries other specs' landed work (TOOL-017 archive, prreview gate, herdr fix). This is harness-owned and was recorded in rounds 1 and 6; as before, the review read the spec's actual surface directly, and the spec-relevant delta since round 6 (`0ecee6e3..HEAD`) is exactly #3004 (`Thread.All`) plus its records.

### Spec and task alignment
- All six acceptance criteria are ticked in `proposal.md` and map to named tests in `verification.md`; the six `features.json` verification commands were re-run in this session and all exit 0 (f1–f6 PASS).
- `dotf spec digest MEMORY-008-handoff-resume` matches the `contract_digests` in `review-request.json` on all three contract files — the review is fresh against the current contract.
- Tasks: every implementation box is `[x]`; the Closing box "Independent review before archiving" is correctly `[ ]` (this round fills it). No `[x]` was found without diff evidence.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in the spec's markdown files.
- Round-6 fix (#3004) verified in place: `Thread.All` joins every field under a label; both renderers read through it; `Get` keeps first-match semantics for `ThreadWarnings`, `deriveEntry`, and ownership. `TestResumeRendersEveryFieldOfALabel` covers both alias pairs in both formats and asserts the Verify note exactly once. Mutation check run in this session: `All` cut to first-match turns that test red (applied, observed FAIL, reverted; `cli/` tree clean).
- Round-4 decision B holds: no `readOnlyCommand` / `codeRunners` / `TestReadOnlyCommand` remnants anywhere under `cli/internal`; Verify renders as written under the fixed `verifyNote`.
- Round-5 quoting holds: exactly two `--thread` command sinks exist in non-test Go (`mem_resume.go:111` candidate line, `session_start_resume.go:44` one-liner), both through `ShellWord`; both quoted-line tests exist.
- #2915's fix is in: `session-start-config.json` declares `handoff_resume`, `tests/session-start-config.bats` counts it, and `TestCommittedConfigDeclaresTheInjectorsTheAdapterReads` pins the file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | derived index entry | `deriveEntry` counts `Awaiting Manu` via first-match `Get`, so a thread carrying the label and its alias (`Judgment calls left open`) writes "**Awaiting Manu:** 1, see the thread" while the thread holds 2 — the round-6 `All` fix reached the renders but not the slice-2 entry count. No data is lost (the thread file holds both; the render is correct); the summary undercounts and could understate pending decisions to a non-hook harness reading only `MEMORY.md`. Input shape proven parseable (`TestResumeRendersEveryFieldOfALabel` constructs it); never observed in a stored thread. | Probe run in this session (throwaway test, removed): `deriveEntry("**Awaiting Manu:**\n- one\n**Judgment calls left open:**\n- two\n", …)` → `**Awaiting Manu:** 1,` | UNTESTED — no test covers the entry count under alias duplication (`thread_store_test.go` has none) | code (`thread_store.go` `deriveEntry`: count via `All`), tests |
| Minor | SPECULATIVE | lint warnings | `ThreadWarnings` reads `Next action` via first-match `Get`: a thread with two `Next action` fields, first empty and second populated, would warn "no Next action" while one exists. No alias maps to `Next action`, so the round-6 shape cannot produce it; only a literal duplicate label could. Code read only; no repro, no observed instance. | `cli/internal/mem/thread.go` `ThreadWarnings` | UNTESTED | code (same `All`-based read) |
| Question | — | harness contract | A live `source=compact` SessionStart payload has never been observed; the `matcher: "*"` firing on compact/clear is argued from Claude's docs. Carried open by rounds 2–6 with a defined observation plan (the next real compaction on a release carrying slice 4). | recorded in `verification.md` rounds 2–6 | n/a | none this round; ticket if the hook does not fire |
| Question | — | process | Ticket states #3002, #3003 (open per round 6), #2915 (closed per round 5), #1689 could not be verified this run: GitHub API rate limit exhausted for every `gh issue view`. The round-6 record stands unverified here. | `gh issue view` → "API rate limit already exceeded" | n/a | none; re-check at archive time |
| Recorded | — | harness | The launcher base range (811 commits) is far wider than this spec's change; unchanged since rounds 1 and 6, harness-owned, not this spec's contract. | `git log e1c5754..HEAD --oneline \| wc -l` | n/a | vault/harness |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All six ACs verified by named tests, no observed defects; two minor edge gaps found (entry count under alias duplication, warnings under duplicate label), both outside observed inputs. |
| Verification       | A | `verification.md` carries reproducible commands, live probes, and per-round mutation tables; all six feature commands re-ran clean in this session and the round-6 mutation reproduced red. |
| Scope              | B | The spec's own slices are focused, but the stated diff range carries 811 commits of other work — harness-owned, recorded, still distorts any naive reading of the diff. |
| Reliability        | B | Atomic writes, locks, budget refusal that fails loudly, fail-open pointer reads, ambiguous-thread listing; the derived-entry count edge is the remaining unhandled path. |
| Maintainability    | B | Small functions, WHY-comments throughout; the `Get` (first-match) vs `All` (join) duality is a live trap that has now produced two defects (round 6 M-1 and the entry count) and deserves either a naming convention or consolidation. |
| Handoff-readiness  | A | Exhaustive round-by-round `verification.md`, lessons 507/508/513 and ADR-056 recorded, promotion candidates answered, open items ticketed (#3002, #3003) or carried with a plan. |

### Verdict
PASS

No Blockers, no REAL Majors, no rubric D. The two Minors are THEORETICAL/SPECULATIVE edges outside observed inputs and are tracked above; per the severity × reality rule they do not move the verdict below PASS. Six review rounds hardened this change against exactly the classes (allowlist false-safety, unquoted interpolation, alias-duplicated fields) a red-team pass should hunt, and every round's fix was confirmed present and mutation-tested, not merely asserted.

### Recommended next steps
For the implementer to disposition in `verification.md` (the contract set is closed under this PASS; none of these require editing `proposal.md`, `tasks.md` or `features.json`):

- **Apply (code + tests)**: make `deriveEntry` count `Awaiting Manu` through `Thread.All` and add a named regression test for the alias-duplicated entry count (the finding is otherwise UNTESTED). Same read for `ThreadWarnings`' `Next action` check.
- **Consider (code)**: consolidate or name the `Get`-vs-`All` contract so the next caller cannot silently take first-match — this duality caused round 6 M-1 and the entry-count gap.
- **Carry**: the live `source=compact` observation plan (rounds 2–6); re-verify #3002/#3003 states at archive time (rate-limited this run).
- **Harness**: the 811-commit base range remains worth fixing in the launcher so future rounds read a honest diff (recorded, not this spec's debt).
