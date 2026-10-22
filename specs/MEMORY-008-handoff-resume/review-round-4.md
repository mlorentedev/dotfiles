---
spec: "MEMORY-008-handoff-resume"
verdict: "FAIL"
reviewed_sha: "dee3fe4f31aeccdd7abeddd7e3d6bbc9b40fb60a"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-22"
---

## Adversarial review

**Scope**: `MEMORY-008-handoff-resume` (fourth independent round; the third left this spec at `4f19d07d`)
**Sources**: `specs/MEMORY-008-handoff-resume/{proposal.md,tasks.md,verification.md,features.json}`;
diff `git diff e1c5754c604c45fb8d236694bf7a2a916e80cfde...HEAD` over the MEMORY-008 surface
(`cli/internal/mem/{resume,thread,thread_store,session_start_resume,handoff,session_end}.go`,
`cli/internal/cmd/{mem_resume,mem_handoff,mem}.go`, `harness/skills/{handoff,catchup}/SKILL.md`,
`harness/enforced/`). No PR was read; the review is of the tree at `dee3fe4`.

**Base caveat (observed, not this change's contract).** The base the launcher resolved,
`e1c5754c604c45fb8d236694bf7a2a916e80cfde`, is ~750 commits behind `HEAD`: the range touches
**2,459 files** and includes the archive moves of most of this repo's specs and unrelated
`harness/skills/` churn. I therefore reviewed the MEMORY-008 surface file by file rather than the
whole range, and could not judge unrelated files. Round 1 already raised this as "Speculative: the
launcher's base gives a 750-commit range"; it is still true.

### Spec and task alignment

- `contract_digests` in `review-request.json` match `dotf spec digest MEMORY-008-handoff-resume`
  byte for byte (all three). The contract had not changed before this round — verified, not assumed.
- All six feature commands in `features.json` were run fresh in this session and each exits 0: f1,
  f2, f3, f4 (`go test -run … -v`, two PASS lines), f5 (the two greps over the skill records) and
  f6 (`grep -rq`, `compile-harness.sh --check` → `[check] OK: no harness drift`, and the
  `bats … 'under 8000 characters'` case → `ok 1`). AC1–AC6 are each mapped to a named, failing-if-broken
  test or check; this part of the round-3 claims holds.
- `go vet ./...` is clean and `go test ./internal/mem/... ./internal/cmd/... -count=1` is green
  (`ok` both, 0.7 s / 21.9 s).
- `tasks.md` slice boxes 1–5 are all `[x]`, including the one that says *"a `Verify` command outside
  the read-only allowlist is marked unverified"*. Finding F-1 below is a `[x]` that the diff does not
  fully support.
- The budget refusal is ordered correctly, not by luck: `CheckMemoryBudget` is called **before**
  `publish`, and the thread file is written only inside `publish`, so a refused write leaves
  `MEMORY.md` and `threads/` untouched (no orphan thread file with no entry pointing at it).
  Verified by reading `cli/internal/cmd/mem_handoff.go:119-135,148+`.
- The knowledge the slices claim exists where they say it does: `docs/adr/adr-056-a-handoff-thread-is-stored-in-its-own-file.md`,
  `docs/lessons/lesson-507-…`, `lesson-508-…`, `lesson-513-a-read-only-allowlist-that-names-a-code-runner-is-not-read-only.md`.
  The doctrine bullet is in `harness/enforced/pr-stewardship.md` and names `dotf mem resume --agent <agent>`.
- The **Closing** block and `proposal.md`'s AC1–AC6 boxes are unticked while `verification.md` ticks
  all six. Rounds 2 and 3 carried this to the archive PR. It is a finding here too (F-3), routed to
  the contract set — safe to fix **now**, while this round is still FAIL, and expensive afterwards.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | security / guard completeness (AC2 area, slice-3 task) | Only **backticked** text is ever inspected. A `Verify:` field whose command is written as prose is rendered unmarked, under a heading with no note, even when the command is not read-only — so the resume presents an untrusted command as one of the "plain read-only" ones. `verifyText` marks only what `` inlineCode `` finds; a field with no backticks never reaches `readOnlyCommand`. Three separate texts promise the opposite: the command's own help (*"A Verify command outside a read-only allowlist, or one that chains, pipes or redirects, is marked unverified"*), `proposal.md`'s risk control, and the `catchup` skill's step 1 (*"one marked unverified was written by another session"*) — which tells the reading session that the **mark** is what distinguishes the two classes, and a prose command has no mark. | Fixture thread `feat-c` with `**Verify:** run go test ./... then cleanup with rm -rf /tmp/x<br>` and `dotf mem resume --thread feat-c` printed `First confirm the state it describes:` with **no** `(unverified)` marker and **no** note; the same field backticked does mark (see `TestResumeKeepsVerifyProseBesideTheCommands`). Reproduced in this session. Impact is bounded by the harness permission gate, which still applies to every command — that is why this is a marker-completeness defect and not code execution. | UNTESTED — the existing test's three cases are all backticked; no case asserts what happens to an unbackticked command, so nothing fails today | code (`cli/internal/mem/resume.go: verifyText`) + tests (add an unbackticked case to `TestResumeKeepsVerifyProseBesideTheCommands`, or a new `TestVerifyProseCommandOutsideTheAllowlistIsMarked`) |
| Major | REAL (rendering) / impact THEORETICAL (needs a human paste) | injection / path from untrusted file to a printed shell command | `listCandidates` prints its copy-paste lines by raw `%s` interpolation of the thread key and of `--agent`; only `--memory` is shell-quoted. `threadHeadingKey` takes the first space-delimited token of a heading with no character validation, so a `MEMORY.md` — the untrusted input class this spec names — can put metacharacters into the line the session tells the operator to run. | `--thread nope --agent "claude --x"` on a fixture whose heading is `### thread: x;echo${IFS}pwned` printed: `dotf mem resume --thread x;echo${IFS}pwned --memory '/tmp/m8probe/MEMORY.md' --agent claude --x`. Reproduced in this session. Nothing executed it. | `TestMemResumeListsCandidates` covers the listing, not metacharacters → UNTESTED for this path | code (`cli/internal/cmd/mem_resume.go: listCandidates` — `shellQuote` the key and the agent, or refuse a key outside `threadFileSafe`) + tests |
| Minor | REAL | handoff / contract bookkeeping | `proposal.md` AC1–AC6 and the `tasks.md` Closing block are unticked while `verification.md` claims all six; the archive checklist is likewise untouched. Round 2 and round 3 carried this and it has now survived three rounds. | `proposal.md` `- [ ] **AC1** … **AC6**`; `tasks.md` `## Closing` | UNTESTED (no gate checks it; n/a) | spec — **contract set**: apply in the same round as F-1/F-2, before the next review, or the forced re-review is spent twice |
| Minor | THEORETICAL | verification metadata | Every `features.json` entry is `"state": "pending"` although all six commands exit 0. The file says only the harness may set `passing`, so this may be correct-by-design; whether `dotf spec archive` requires `passing` cannot be observed from inside this repository. | `features.json`; the commands run above | n/a | harness / UNVERIFIED (gate lives outside this repo) |
| Minor | REAL (documented gate) | archive readiness | `verification.md`'s three **Promotion candidates** are still unanswered bullets. `dotf spec archive --help` documents a third pre-flight that reads exactly that section and refuses unless every line is answered `yes: <path>` or `no: <reason>`, with no flag to skip it. So the archive will refuse for a reason nothing in the current round names — and `verification.md` is outside the contract set, so it can be fixed without staling a review. | `dotf spec archive --help`, third pre-flight paragraph; `verification.md` "Promotion candidates" | UNTESTED (the refusal was not observed, to avoid moving the folder) | spec — `verification.md`, free to edit at any time |
| Minor | THEORETICAL | purity (AC1) | The `context` render embeds the thread file's absolute path (`Source: <path>`), so the output is a function of the thread file **and its location**; AC1 says "a pure function of the thread file". Harmless for equality across two runs, but the golden test passes an arbitrary `src`, so the property AC1 states is weaker in the code than in the criterion. | `dotf mem resume --format prompt` printed `Its full handoff is /tmp/m8probe/MEMORY.md` | `TestResumeIsAPureRenderOfTheThreadFile` (renders the same `source` twice) — does not exercise the difference | code or spec wording — one of the two should move, not both |

Tracked elsewhere, correctly dispositioned, **not** findings: `handoff_resume` is absent from
`session-start-config.json` and is filed as **#2915** (rounds 2, 3); the live `source=compact` payload
is still unobserved and is recorded as open in `verification.md`; the uncapped resume was declined
with a reason and the measured thread sizes support it.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Every AC has a passing named test, but the slice-3 guarantee under AC2 — "a `Verify` command outside the read-only allowlist is marked unverified" — does not hold for prose commands (F-1), and `[x]` is ticked anyway. |
| Verification       | A | Six reproducible commands, all re-run here and green; mutation tables per slice; live probes on a copy of the real file; the base-sha caveat disclosed rather than hidden. |
| Scope              | B | The changed surface stays inside the slice list; the 2,459-file / ~750-commit range comes from the launcher's base, not from this change, and the reference to it is out of my control. |
| Reliability        | B | Budget refusal, missing memory → empty injection, unknown format refused, unknown source normalised; the marker path is the one error path that silently says nothing where it should say something. |
| Maintainability    | B | Small pure functions, `resume.go` well inside the complexity ceilings, WHY comments that name the review that produced each rule; the duplicated shell-quoting in `mem_resume.go` (one call, one `+=`) is the seam that let F-2 through. |
| Handoff-readiness  | B | Decisions and lessons recorded in-session, `verification.md` is unusually complete, round history preserved; the contract set is stale against it (F-3). |

### Verdict

**FAIL** — two **REAL** Majors (F-1, F-2). Rubric has a C and no D, which would only force
PASS-WITH-GAPS; the severity axis forces FAIL. F-1 is the same defect class three earlier rounds
returned FAIL on (round 3 F-1/F-2: "rendered as verified" when it was not), reached this time through
the one path — prose rather than a backticked span — that no round has exercised.

### Recommended next steps

On this FAIL the contract set is open; a re-review follows the fixes.

1. **Code + tests — F-1:** make the unverified claim total. Simplest shape: if a `Verify` field's
   text contains no backticked span but does contain something `readOnlyCommand` would refuse (or
   simply: a field with no backticked span at all), mark the field unverified and set the note. Add
   the named test above with the reproduction command from this review.
2. **Code + tests — F-2:** `shellQuote` the thread key and `--agent` in `listCandidates` (the
   `--memory` already is), or validate the key against `threadFileSafe` before listing it. Add the
   metacharacter case to `TestMemResumeListsCandidates`.
3. **Spec (contract set, now while the verdict is FAIL) — F-3:** tick `proposal.md` AC1–AC6 and the
   `tasks.md` Closing block, and settle "about 350 B" against the 450 B worst case (carried since
   round 2). Doing this after a PASS would invalidate the verdict; doing it now costs nothing.
4. **Disclose in `verification.md` — F-4/F-5:** note that `features.json` states are harness-owned,
   and pick one of the two AC1 wordings (code renders the location, or the criterion says so).
5. **Answer the promotion candidates — F-6:** ADR-056 and lessons 507/508/513 already exist, so the
   three bullets can be answered `yes: docs/adr/…` / `yes: docs/lessons/…` and a `no: <reason>` for
   the pattern, which is a `verification.md`-only edit.
6. **Leave as-is:** #2915, the unobserved `source=compact` payload, and the declined uncapped-resume
   finding — all correctly dispositioned.

`dotf spec archive` is **not** advisable in this state: the verdict is FAIL and the gate would refuse
it in any case. The minimum set that flips this to PASS is (1) and (2) with their named tests, plus
(3) applied before the next review is launched; (5) is not required for the verdict but is required
before the archive can succeed.
