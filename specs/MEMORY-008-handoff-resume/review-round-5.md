---
spec: "MEMORY-008-handoff-resume"
verdict: "FAIL"
reviewed_sha: "8e0b61bcdf08538e02724852f07dbe78d60ee2c6"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-30"
---

## Adversarial review

**Scope**: MEMORY-008-handoff-resume (round 5; launcher base `e1c5754c…`, whole change `git diff e1c5754c604c45fb8d236694bf7a2a916e80cfde...HEAD`)
**Sources**: `specs/MEMORY-008-handoff-resume/{proposal,tasks,verification,features}.md|json`, review rounds 1–4, `review-request.json`; implementation in `cli/internal/mem/{resume,thread,thread_store,handoff,session_start_resume}.go`, `cli/internal/cmd/mem_resume.go`, harness skill records, `docs/lessons/lesson-513-…`, `docs/adr/adr-056-…`

### Spec and task alignment

- **Contract integrity**: `dotf spec digest MEMORY-008-handoff-resume` matches all three `contract_digests` in `review-request.json` exactly. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder. `proposal.md` AC1–AC6 are ticked; `tasks.md`'s Closing box "Independent review before archiving" is correctly unticked (that is this review).
- **AC1–AC6**: each has a named test and each of the six `features.json` verification commands was re-run fresh this session — all exit 0 (f1–f6). `go vet ./...` exit 0; `go test ./...` in `cli/` exit 0; `./scripts/compile-harness.sh --check` exit 0 ("no harness drift"); `bats tests/compile-harness.bats --filter '…under 8000 characters'` passes; `bats tests/spec-gate-archive.bats` — 35 tests, 0 failures.
- **Round-4 late fix judged against the earlier rounds**: decision B landed completely where it was promised — `readOnlyCommand`, `codeRunners`, `writesFile` and `shellControl` are gone from `cli/` (grep); `Verify` renders as written under exactly one fixed note (`verifyNote`), asserted by `TestResumeRendersVerifyAndAwaiting` (count == 1) and `TestResumeKeepsVerifyProseBesideTheCommands` (note + field, `(unverified)` never present); the cmd `Long` help, the catchup skill step 1, the proposal risk bullet and lesson 513 all reworded to the note, no surface still promises a read-only mark. Round-4 F-2 quoting landed in `listCandidates` (`shellWord`/`shellQuote`), tested by `TestMemResumeQuotesTheCandidateLine` with a hostile key `x;echo${IFS}pwned`. Round-4 F-5 reworded AC1 to name the source path. Goldens re-pinned for both formats and three sources.
- **Test-deletion audit (round-4 commit `8e0b61bc`)**: `TestReadOnlyCommand` (≈24 cases) was deleted with the feature it guarded. Retention bar: (1) it asserted which `Verify` commands render marked; (2) remaining proof *shown* — this session mutated `verifyNote` to `""` and `TestResumeRendersVerifyAndAwaiting` + `TestResumeKeepsVerifyProseBesideTheCommands` went red, then reverted (tree clean); removing quoting in `shellWord` likewise turned `TestMemResumeQuotesTheCandidateLine` red, then reverted; (3) no seam unlocked — the production helper was deleted with the test; (4) history: created `1fd9bfac` (#2880), hardened `2e8777c6`/`4f19d07d`/`dee3fe4f`, deleted `8e0b61bc` (#2990) with the feature, recorded in lesson 513 (amended in the same commit); (5) focused command: `cd cli && go test ./internal/mem -run 'TestResumeKeepsVerifyProseBesideTheCommands|TestResumeRendersVerifyAndAwaiting'`. **Bar satisfied — no finding.**
- **Base-range note**: the launcher's base gives a 2552-file range (all of `main` since release 0.58.0). This spec's own change is the 14 commits `40e576fa…8e0b61bc` touching the spec folder, the `mem`/`cmd` packages, skill records, lesson 513 and ADR-056; I reviewed that surface in full within the given range. The rest belongs to other PRs/specs (round 1 recorded the same base-range quirk as not this change's contract).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | injection / session-start one-liner (AC3 surface) | Round-4 F-2's fix is incomplete: `sessionResume`'s one-line reply for `source=resume`/`fork`/unknown interpolates the thread key **raw** into a backticked, copyable command — `` [resume] Thread <key> has a handoff; `dotf mem resume --thread <key>` renders it. `` — the same "hostile key ends up in a shell word the operator or next session runs" class that `#2944` fixed only in `listCandidates`. Commit `f38a294c` touched `cli/internal/cmd/mem_resume.go` alone; no round examined this line. | Reproduced end-to-end at function level in this session (throwaway test, deleted after running): `ThreadKey("/tmp/copied-tree/proj-wt-a;id")` → `"wt-a;id"` — the non-git `-wt-` fallback applies no sanitization, and `sanitizeThread` maps only `/ \ space :` so `;` survives the git path too (`git check-ref-format 'refs/heads/x;echo-pwned'` and `'refs/heads/x$(id)'` both exit 0 — git accepts them in branch names); `sessionResume("resume", mem, "wt-a;id", "claude")` printed `` `dotf mem resume --thread wt-a;id` `` with the `;id` unquoted. Full chain needs a hostile branch/`-wt-` directory name *and* a matching `MEMORY.md` heading — each hop code-verified, the whole chain never observed live. | UNTESTED — `TestSessionStartResumeBySource` pins goldens only for the plain key `wt-x`; nothing asserts the one-liner's key is one shell word | code + tests (`cli/internal/mem/session_start_resume.go`; share/duplicate the `shellWord` rule from `cli/internal/cmd/mem_resume.go`, and add a hostile-key case to the resume/fork golden path) |
| Minor | REAL | spec artifacts (`verification.md`, outside the contract set) | Slice-3 evidence still presents the retired read-only marker as current behaviour: *"Verify commands are listed one per line, and one outside the read-only allowlist is marked unverified (`TestResumeRendersVerifyAndAwaiting`). The allowlist matches on a word boundary … (`TestReadOnlyCommand`)"* — but `TestReadOnlyCommand` and the allowlist were deleted at HEAD (#2990), and the round-4 disposition table lower in the same file records the retirement: the file contradicts itself. | `grep -rn 'TestReadOnlyCommand' cli/` → no matches; `git show 8e0b61bc --stat` shows `-func TestReadOnlyCommand`; `verification.md` lines 84, 94–95 | `TestResumeRendersVerifyAndAwaiting` exists but now asserts the note, not a mark — the stale paragraph's refutation is the grep itself | spec (`verification.md` — editable without invalidating this review) |
| Question | UNVERIFIED | SessionStart matcher (carried since round 2) | A live Claude `source=compact`/`clear` payload firing the `matcher: "*"` SessionStart group has still never been observed; the golden fixtures prove the renderer, not the hook firing. | Carried disposition in `verification.md`; no live observation possible in this session. Not blocking: a hook that does not fire gets its own ticket. | n/a | n/a (keep open; observe the next real compaction) |
| Minor | carried, ticketed | config surface | `handoff_resume` is not declared in `session-start-config.json` (bats counts 11 injectors while the code has a 12th on by default). | `gh issue view 2915` → **OPEN** ("Declare the handoff_resume injector in session-start-config.json"), verified this session. | n/a | code, tracked by #2915 — not blocking |

UNVERIFIED: byte-for-byte freshness of the `handoff`/`catchup` skill records against their vault source (`verification.md` claims `--refresh` output from vault commit `75d90a9a`); the vault is outside this repository's scope for this review. The in-repo proxies do pass: `features f5` grep and `compile-harness.sh --check`.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All six ACs met with named passing tests and real negative paths (ownership, ambiguity, prose retention, quoting); one residual unhardened interpolation of an already-fixed class (finding 1). |
| Verification       | A | Six feature commands re-run fresh exit 0; per-source goldens; two mutations killed live this session plus four prior-round mutation tables; live-run probes recorded in `verification.md`. |
| Scope              | B | The spec's 14 commits match the proposal exactly (fixes #2990/#2944 stay on the spec's own surfaces); the launcher's base range carries unrelated main history, recorded, not creep by this change. |
| Reliability        | B | Missing memory injects nothing; unknown source gets one line; unknown format errors; over-budget writes fail loudly; atomic writes under lock; `go vet` and `go test ./...` clean. |
| Maintainability    | B | Small functions, no dead code left by the marker deletion (grep confirms all four helpers gone), tests colocated and descriptively named. |
| Handoff-readiness  | B | Proposal/tasks/features internally consistent and round-4 dispositions recorded; `verification.md` carries one self-contradicting paragraph (finding 2); lesson 513 + ADR-056 written; carried tickets filed. |

### Verdict

**FAIL** — one **REAL Major** (unquoted thread key in the session-start one-liner, reproduced; same class as the round-4 fix that was applied everywhere *except* here). The rubric is all B (no C, no D), but the severity × reality rule governs independently: a REAL Major stands until fixed and re-reviewed. The two minors/the question are tracked and do not move the verdict.

### Recommended next steps

1. **code + tests (finding 1):** apply the round-4 one-shell-word rule to `sessionResume`'s one-liner — the `shellWord` helper must move into (or be shared with) `package mem` — plus a named test with a hostile key (`wt-x;echo`) on the `resume`/`fork` golden path that goes red if the quoting is removed. This is outside the contract set, so it does not stale this review.
2. **spec (finding 2):** rewrite `verification.md`'s slice-3 paragraph (lines 84, 94–95) to describe the note-not-marker behaviour and drop the `TestReadOnlyCommand` references as current. Also outside the contract set; do it alongside the fix.
3. **Re-review (round 6)** after 1 lands (`dotf spec review MEMORY-008-handoff-resume`). The contract files were not touched by this verdict, so their digests remain valid for the next round.
4. Carried dispositions stay as recorded: #2915 open; observe a live `source=compact` payload.

**`dotf spec archive` is NOT advisable in the current state**: the gate refuses a signed `FAIL` ("a FAIL is resolved by its findings: apply them in a follow-up, then re-review", `cli/internal/spec/review.go`) and would also have refused the provisional `PENDING` this run wrote first. Minimum to flip to PASS: item 1 with its named red-green test, item 2, then a fresh round.
