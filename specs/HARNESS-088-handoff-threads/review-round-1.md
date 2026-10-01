---
spec: "HARNESS-088-handoff-threads"
verdict: "FAIL"
reviewed_sha: "e3a9b4d17c4a3a3b07fd70a331ed97d772519cbe"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: spec `HARNESS-088-handoff-threads` — the thread-scoped handoff writer.
**Sources**: `specs/HARNESS-088-handoff-threads/{proposal,tasks,verification}.md` +
`features.json`; `git diff 0df4bf8d59a012b226ec214ecbf3640e6ab47c7a...HEAD`;
`cli/internal/mem/{handoff,thread,identity,session_end}.go`,
`cli/internal/cmd/mem_handoff.go`, `harness/skills/handoff/SKILL.md`,
`tests/guard-no-session-handoff.bats`.

### Diff-scope caveat (stated, not a finding)

The launcher's base `0df4bf8d` is **279 commits behind HEAD**, so `git diff base...HEAD`
is 1,188 files / ~94k insertions and includes a month of unrelated `main` work (other specs,
ADRs, secrets-blob removals, Windows setup, CI). The spec-relevant surface is 14 files:
`cli/internal/mem/*` (handoff/thread/identity/session_end + tests), `cli/internal/cmd/mem_handoff.go`,
`harness/skills/handoff/SKILL.md`, `tests/guard-no-session-handoff.bats`. I reviewed that surface
by reading and running it. The remaining ~93k lines are **UNVERIFIED** here and out of this
spec's scope.

### Spec and task alignment

- All of `tasks.md`'s Implementation and Closing boxes are ticked or carry a stated reason; AC8 is
  declined **with** a tracker (`#1881`), which is the correct form for a dropped criterion.
- `features.json` f1–f5 all name tests that exist; I ran each `verification` command verbatim — all
  exit 0 (`go test -run` with a non-matching name also exits 0, so I confirmed by name, not by exit
  code, that each names a real test).
- AC1–AC7 each map to a named, passing test (verified with `go test ./internal/mem/ -run '…' -v`).
  AC4 is genuine: `TestWrittenThreadsStayInsideTheArchivedBlock` calls the real `extractHandoffBlock`.
- `go test ./...` in `cli/`: all packages ok. `golangci-lint run ./...`: 0 issues.
  `go build ./...` and `go vet ./internal/mem/`: clean. `bats tests/compile-harness.bats`: no failures.
- Mutation check of the core span logic (remove `threadSpan`'s next-marked-thread scan, so a
  replacement absorbs the following thread): caught by
  `TestWriteThreadSurvivesASubheadingInsideAThreadBody` — the boundary property is genuinely guarded.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | concurrency / data loss | Two concurrent `dotf mem handoff-write` runs lose one thread silently: the command reads the whole file, computes, then renames it over the original — a read-modify-write with no lock, compare-and-swap or re-read. The last writer wins and erases the other's thread, exiting 0 and printing `wrote …`. This is exactly the invisible last-writer-wins failure the spec exists to remove, returned in a millisecond window. The proposal declares a lock file a non-goal, but no mitigation (retry/CAS) and no documented residual window exist either. | Reproduced with the built binary: 2 writers, 40 rounds, launched together → thread A lost in **6/40**; control, writers 200 ms apart → **0/40**. Earlier, 4 writers × 60 rounds → 58/240 lost. Sequential runs are clean, so the loss is the write window, not the harness. | UNTESTED — no named concurrent-writer test exists (`grep concurrent/goroutine/parallel` in `cli/internal/mem/*_test.go` finds only prose). A regression test would be e.g. `TestConcurrentWritesDoNotLoseAThread` (N goroutines over `WriteThreadAs`, or N `dotf` processes). | code + tests |
| Minor | REAL | input validation / idempotency | An explicit `--thread` key containing a space is not idempotent once `--agent` is set: `writeThread` accepts spaces (it rejects only empty/newline keys), but `isThreadHeading` matches a stamped heading by its first space-delimited token, so it never finds `### thread: my work (writer: claude)` and appends a duplicate block on every run. | `printf … \| dotf mem handoff-write --memory f.md --thread "my work" --agent claude` twice → `grep -c '### thread: my work'` = **2**. (Without `--agent`, exact-match makes it idempotent.) | UNTESTED | code + tests |
| Question | SPECULATIVE | process | `writeThread` lifts an un-threaded legacy block and re-appends it as `legacy-<date>`; `journalWriter` attributes a writer by scanning hyphen-split journal words for a known agent name, asserting "no project name contains one". A project actually named `pi`/`codex`/`gemini` would be misattributed and its block forked rather than replaced. | Code read of `journalAgents`/`journalWriter`; no occurrence observed. | UNTESTED | code (or spec: state the invariant as a checked assumption) |

No defect was found in `identity.go`, `SessionEnd`'s `O_EXCL` fallback (incl. removal of a
half-written record), or the `--dry-run`/stderr reporting paths.

**Where each fix lands.** The Major is **code + tests**, not a contract edit: the contract set
(`proposal.md`/`tasks.md`/`features.json`) is what the staleness gate watches, so a passing verdict
cannot co-exist with a request to edit it. The Minor is likewise code + tests.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All stated ACs pass and negative paths are tested, but AC1's "every other byte-identical" guarantee fails for concurrent writers (reproduced). |
| Verification       | B | Reproducible commands and outputs, and the spec even records a prior false-negative mutation test; the concurrency path is absent from the evidence table. |
| Scope              | B | The spec change itself matches the proposal; it also carries the documented SessionEnd/identity reworks. The launcher's base, not the change, makes the surrounding diff enormous. |
| Reliability        | C | Atomic rename and `O_EXCL` are good, but the unguarded read-modify-write silently loses data under the multi-worktree usage the spec is built around. |
| Maintainability    | B | Clear naming, functions within limits, WHY-comments; the pervasive essay-length comments are heavy but justified by the defect history. |
| Handoff-readiness  | A | `verification.md` is unusually honest (records its own false-negative mutation, fixes a dead `-run` name); AC8 decline and debt are ticketed (#1881, #1882). |

### Verdict

**FAIL** — one **REAL Major** (silent lost update under concurrent handoff writes) that is
**UNTESTED**. The rubric's two C's would independently put this at PASS-WITH-GAPS; the REAL Major
forces FAIL.

### Recommended next steps

- **Close the lost-update window** (code + tests). Cheapest correct options: an `O_EXCL`/`flock`
  guard around read→rename, or re-read + compare-and-swap + bounded retry, or `renameat2`-style
  atomic exchange. Add a named regression test (`TestConcurrentWritesDoNotLoseAThread`) that runs
  N concurrent writers and asserts every thread survives; a test that merely runs the writers
  sequentially proves nothing here.
- **Either document the residual window** in the handoff skill and an ADR if a lock is deliberately
  declined, **or** fix it. A silent permanent residual is not acceptable for the class of loss this
  spec was opened to remove; "the writer removes the cause" is the same reasoning that already failed
  once (the proposal's own Why).
- **Validate the thread key** (code + tests): reject or sanitize a key containing whitespace in
  `writeThread`, or make `isThreadHeading` match the key before the stamp robustly.
- **Disposition the `journalWriter` heuristic** (code, or an ADR recording the assumed invariant with
  a check).
- After the code/tests change, run a fresh adversarial review: the Major is outside the contract set,
  so the contract files must stay untouched to keep any subsequent verdict valid.
