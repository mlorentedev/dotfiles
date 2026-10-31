---
spec: "MEMORY-008-handoff-resume"
verdict: "FAIL"
reviewed_sha: "302a427bbaacba77dab4a7c21dbf579233115dff"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-21"
---
## Adversarial review

**Scope**: MEMORY-008-handoff-resume (slices 0–5, issue #1689)
**Sources**: `specs/MEMORY-008-handoff-resume/{proposal,tasks,verification,features}.md`; the spec's own commits inside
`git diff e1c5754c...HEAD` — `9603e9ac`/`8ce4bcd4`/`221e0daf` (slice 0), `ffe5c44b` +`6436c716` (slice 1),
`2ed09c3f` (slice 2), `1fd9bfac` + `2cfa06ea` (slice 3), `1753d520` (slice 4), `302a427b` (slice 5).
Code read: `cli/internal/mem/{thread,thread_store,resume,session_start_resume,session_start_adapter,handoff}.go`,
`cli/internal/cmd/{mem,mem_handoff,mem_resume}.go`, `harness/enforced/pr-stewardship.md`, golden fixtures under
`cli/internal/mem/testdata/`.

**Note on the stated base.** `e1c5754c` is `chore(main): release 0.58.0`, the release in effect when the spec was
written (2026-09-25). It is an ancestor of HEAD, but the range carries **750 commits / 2,433 files**, because the
five slice PRs landed on `main` across a month of unrelated work. It is therefore not a bounded diff of this change.
This review reads the nine commits above (the spec's own production and test diff) plus the tree at `302a427b`.
That is a launcher/base-resolution defect, not an implementation defect; recorded as a finding so the range is not
silently treated as "the change".

### Contract and digest check

`dotf spec digest MEMORY-008-handoff-resume` returns exactly the three digests recorded in `review-request.json`
(`features.json 358ebb49…`, `proposal.md 297a6356…`, `tasks.md 97e4977c…`). The review is against the current
contract; the uncommitted `tasks.md` change in the tree is checkbox ticks only, which the digest ignores.

### Spec and task alignment

- **AC1** — verified. `RenderResume` reads no clock, no cwd, no env: two runs of `dotf mem resume` on the same file
  were byte-identical (`cmp` clean), and `TestResumeIsAPureRenderOfTheThreadFile` pins both formats against
  `testdata/resume/{context,prompt}.golden`. An unknown `--format` exits 1 with a named message (run, not read).
- **AC2** — **partially met**. `Verify` and `Awaiting Manu` both render, and an ambiguous directory lists candidates
  (`TestResumeListsCandidatesWhenTheThreadIsAmbiguous`, `TestMemResumeListsCandidates`; live run in this checkout
  printed ten candidate threads and exit 0). Two gaps: the read-only allowlist marks execution-capable commands as
  safe (F-A), and the thread-ownership rule applied on read is weaker than the one applied on write (F-B).
- **AC3** — verified. Five goldens, one per `source`; `TestSessionStartResumeBySource` also asserts the compacted
  `ClaudeContext` *begins* with `## Resume: thread wt-x`. `startup` getting the full block is a recorded decision
  that widens AC3's letter (AC3 names only `clear`/`compact`); tasks.md and verification.md agree, so it is spec
  drift accepted in writing, not a silent divergence.
- **AC4** — verified and non-vacuous: `TestAMemoryShapedLikeDotfilesStaysUnderBudgetWithTheNewFields` fails itself
  if the inline fixture is *not* past 25,000 B ("the test proves nothing"), then requires the stored form under it.
  The real-file probe figures are in verification.md; the real file is not committed (repo is public) — sound.
- **AC5/AC6** — verified at HEAD: `f5` and `f6` both exit 0; `harness/enforced/pr-stewardship.md:11` carries the
  sentence, `bats tests/compile-harness.bats` = 87 ok / 0 fail, and the sentence reached a live non-hook harness —
  it is in this reviewer's own injected doctrine. Independent of the repo: the vault-side skill edits (commit
  `75d90a9a`) are outside this checkout and **UNVERIFIED**.
- Prerequisite slice 0 (#1606/#1620/#1651) landed as `9603e9ac`/`8ce4bcd4`/`221e0daf` before slice 1, as proposed.
- Every slice's production diff is inside the ~300 executable-line cap once comments are stripped (254 for
  `2ed09c3f`, 284 for `1fd9bfac`, 69 for `1753d520`); each PR carries a `## Knowledge` section (checked on #2852,
  #2880, #2881, #2885, #2908), and ADR-056 plus lessons 507/508 exist on this tree.
- No test was deleted. The four removed assertions (`2ed09c3f`, `2cfa06ea`) were re-pointed, not dropped: the
  `**Trap:**` case is still asserted in `TestParseThreadKeepsAnUnknownLabel` and the budget message in
  `TestTheBudgetRefusalSuggestsOnlyDatedThreadsAndHowToArchiveEach`.

### Verification run in this session

`go build ./...` exit 0, `go vet ./...` exit 0, `go test ./internal/mem ./internal/cmd -count=1` both `ok`,
`bats tests/compile-harness.bats` 87/87, and all six `features.json` verification commands exit 0 (f1–f6 run
individually, output above the line). The implementer's evidence claims hold.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | security (`Verify` allowlist) | The read-only allowlist renders execution-capable commands as safe, with no `unverified` marker. `bats` is allowlisted bare, so `bats /tmp/evil.bats` runs arbitrary shell from any path — the same "takes any path" class the code excludes `cat`/`grep` for. `go test -exec=…` runs each test binary through an arbitrary program, and `go test --outputdir=…` writes anywhere; neither is caught by `writesFile` (its regex `^--output(=.*)?$` does not match `--outputdir`). The proposal names `Verify:` as this spec's injection surface, so the mitigation leaks at its stated boundary. | Reproduced at HEAD: `dotf mem resume --memory … --thread feat-x --format context` printed ``- `go test -exec=/tmp/evil-shim ./...` `` and ``- `bats /tmp/evil.bats` `` unmarked; `go test -exec=/tmp/m8rev/evil-shim ./internal/mem` then returned `fork/exec /tmp/m8rev/evil-shim: no such file or directory`, i.e. the flag was honored and the path executed. | `TestReadOnlyCommand` exists but does not cover any of the three (none of its 24 cases); UNTESTED for these inputs | code (`readOnlyPrefixes`, `writesFile`) + tests (extend `TestReadOnlyCommand`, re-pin `testdata/resume/prompt.golden`) |
| Major | REAL | ownership (`SelectThread`) | The read path applies a weaker ownership rule than the write path, contradicting its own doc comment ("the ownership rule `WriteThreadAs` and `session-end` already apply"). `SelectThread` reads only the heading stamp (`threadStamp`); `threadOwner`/`threadWriter` additionally attribute an unstamped thread by its journal filename. So claude is handed pi's thread when the stamp is absent and ownership lives only in the Journal line — the common shape for blocks written before the stamp existed. | Reproduced at HEAD: fixture `### thread: feat-y` with `**Journal:** sessions/2026-10-20-dotfiles-pi-feat-y.md`, then `dotf mem resume --thread feat-y --agent claude` printed "Resume thread feat-y … Next action: pi's step, not yours." Mutation proof that nothing pins it: replacing `threadStamp` with `threadWriter` in `SelectThread` left `go test ./internal/mem ./internal/cmd -count=1` fully green (reverted; tree clean). | UNTESTED — `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` covers only stamped/unstamped-without-journal cases; the surviving mutation shows no test distinguishes the two rules | code (`cli/internal/mem/resume.go`) + tests (a named case for a journal-owned unstamped thread) |
| Minor | REAL | CLI default / doctrine | With no `--agent`, `SelectThread` returns the key even when it is stamped `(writer: claude)`, and the compact doctrine tells non-hook harnesses to run the bare command and "follow its Next action". A pi or copilot session at start is therefore pointed at claude's next action. | Reproduced: `dotf mem resume --memory … --thread feat-x` (no `--agent`) rendered claude's thread; with `--agent pi` the same command correctly listed candidates. `harness/enforced/pr-stewardship.md:11` is the bare form. | UNTESTED — the only empty-agent case in `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` is `{"feat-b", "", "feat-b"}`, an unstamped thread | code (resolve the agent from the harness, or list when no agent is known) — the alternative is a doctrine wording change, which is generated from vault, not from the contract set |
| Minor | REAL | render correctness | The pointer `resume` prints is reconstructed as `filepath.Dir(memoryPath)/MEMORY.md`, not the file actually read (`ThreadText` is only handed the directory). With `--memory` naming anything else, the next session is told to read a path that does not exist. | Reproduced twice: `--memory /tmp/m8rev/memory2.md` printed `Its full handoff is /tmp/m8rev/MEMORY.md`; `--memory /tmp/m8rev/z.md` printed `Source: /tmp/m8rev/MEMORY.md`. | `TestThreadTextForKey` pins the reconstruction itself (`source != filepath.Join(dir, "MEMORY.md")` is its failure condition), so the wrong-path case has no test that can see it | code (`ThreadText` takes/returns the memory path) + tests (update `TestThreadTextForKey`, re-pin the `resume` goldens which substitute `filepath.Dir(memory)`) |
| Minor | REAL | Definition of Done (debt) | verification.md slice 4 promises a "Follow-up ticket" for declaring `handoff_resume` in `session-start-config.json` (the bats there counts 11 injectors); no such issue exists. A noted item left verbal is Standing Order #7 / DoD #1. | `gh search issues "handoff_resume" --repo mlorentedev/dotfiles` → only #690 and #1689; `"injector"` search → #690 (dead-config bug, unrelated) and nothing for this. | n/a | vault/forge (file the issue) — not code |
| Minor | REAL | spec artifacts (stale) | verification.md's top Evidence list still has AC5 and AC6 unchecked while both criteria are implemented, ticked in tasks.md Closing, and pass f5/f6 — the file contradicts the tree it documents. | Ran f5 and f6 here: both exit 0; `harness/enforced/pr-stewardship.md:11` carries the sentence. | n/a | verification.md (outside the contract set, so safe to fix without invalidating this review) |
| Minor | SPECULATIVE | scope / tooling | The launcher's stated base produces a 750-commit range for a five-slice spec, which invites a review of nothing (the failure mode SDD-042 removed). Recording it here because a future round may read the range literally. | `git diff --stat e1c5754c...HEAD` → 2,433 files, +164,599; `git log --oneline … \| wc -l` → 750. | n/a | spec tooling (not this change's contract files) |
| Minor | REAL | render cosmetics | A label whose list has one item collapses onto the label line (`**Awaiting Manu:** - A or B`), because the leading blank of the parsed value is trimmed away and `writeField` then picks the space separator. Two or more items render as a block. Cosmetic, but it is the field that tells the session what to ask the owner. | Observed in the same live runs above; `testdata/resume/prompt.golden` shows the multi-item form rendering correctly. | UNTESTED | code (`writeField`) + tests — or accept and pin it in the golden |
| Question | THEORETICAL | production wiring | The injected SessionStart hook group is emitted with `matcher: ""` → `"*"` (`cli/internal/harness/bind.go:323`). AC3 only lands if Claude fires the hook on `clear`/`compact` as well as `startup`. The matcher default predates this spec and is not its change, and I could not confirm Claude's matcher semantics from inside this repo. | `harness/manifest.json:257-266` (bind emits `mem session-start` on SessionStart), `bind.go:323-327`; the goldens prove the Go side only. | UNVERIFIED (needs one live `source=compact` hook payload from a real Claude session) | n/a — confirm, then close |

### Rubric-independent notes (strengths that bear on risk)

- The two named risks in the proposal were both answered with artifacts, not prose: the envelope is pinned by
  `TestClaudeEnvelope…`-family goldens, and the doctrine-budget risk was measured (7,574 → 7,691 against 8,000)
  rather than asserted. The 117-character growth against a cap the proposal guessed at 7,798 is exactly the kind of
  stale figure this gate exists to catch, and the implementer caught it themselves.
- Slice 2's write order (thread file first, then MEMORY.md), the mode preservation, and the
  read→compute→rename lock all close failure modes that would have lost threads silently.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC1/AC3/AC4 hold under execution; AC2 is met on its letter but two reproduced defects sit inside it (allowlist leak, weaker read-path ownership). |
| Verification       | B | Every slice carries named tests, mutation tables and measured figures, and f1–f6 + `go test` + 87 bats all re-ran green here; but the surviving mutation shows the ownership claim is unpinned. |
| Scope              | B | The nine commits match the proposal; slice 0 landed first as scoped; the vault-health exemption in `2ed09c3f` is a consequence of the new files, not creep. Contract files untouched by this review. |
| Reliability        | B | Atomic replace with mode retention, lock across read→compute→rename, loud budget refusal, fail-open to the entry when a thread file is unreadable, one line rather than a guess for unknown `source`. |
| Maintainability    | B | `thread.go` 150 / `resume.go` 230 / `thread_store.go` 304 lines, small functions, no dead code, comments say WHY; `CheckMemoryBudget` is the one function reaching toward the length limit. |
| Handoff-readiness  | C | ADR-056 and lessons 507/508 exist and every PR names its Knowledge line, but a promised ticket was never filed and verification.md contradicts the tree on AC5/AC6. |

Aggregation: two C, no D → PASS WITH GAPS at minimum from the rubric. The severity×reality path is more severe:
two **REAL** Majors → **FAIL**. The more severe of the two governs.

### Verdict
FAIL

### Recommended next steps

Minimum set to flip this to PASS (all of it lands outside the contract set — code, tests, goldens,
`verification.md`, and a GitHub issue — so none of it invalidates a re-review of this same contract):

1. **F-A, the allowlist leak.** Decide what "read-only" means for `go test` and `bats`: either drop bare `bats` and
   bare `go test` to a form that carries no executable-or-writing argument, or extend `writesFile` into a general
   "argument that names a program or an output path" reject (`-exec`, `--outputdir`, `-o`, `--output`, plus a path
   argument for `bats`). Add the three reproduced inputs to `TestReadOnlyCommand` as red cases first, then re-pin
   `testdata/resume/prompt.golden`. This is the mitigation the proposal promises for its own stated injection surface.
2. **F-B, ownership symmetry.** Make `SelectThread` attribute an unstamped thread the way `threadOwner` does (journal
   fallback), and pin it with a named test — the mutation that swapped in `threadWriter` stayed green, so today no
   test can tell which rule the read path uses. Decide F-C in the same pass: either resolve the agent when the caller
   omitted `--agent`, or have the no-agent path list candidates; the bare `dotf mem resume` the doctrine teaches is
   the common invocation for pi, copilot and codex.
3. **F-D.** Thread the real memory path through `ThreadText` and return it, instead of reconstructing `Dir(memoryPath)/MEMORY.md`; `TestThreadTextForKey` currently asserts the reconstruction, so it moves with the fix, and the `session_start_resume` goldens substitute `filepath.Dir(memory)` and will need re-pinning.
4. **F-E/F-F.** File the `session-start-config.json` / `handoff_resume` ticket now (not "later"), and tick AC5/AC6 in
   `verification.md` so the evidence list matches the tree. Leave `proposal.md`, `tasks.md` and `features.json`
   untouched by that edit — a contract edit here costs another full round.
5. Re-review after 1–2 land (`dotf spec review MEMORY-008-handoff-resume`), since a fix to `readOnlyCommand` interacts
   with the goldens and `SelectThread` interacts with the slice-4 injectors; judging them separately is exactly what
   a per-round delta would miss.
6. Out of reach from this checkout, and to be confirmed by the implementer or a follow-up: the vault-side skill
   commits (`75d90a9a`) behind AC5, and one live Claude `source=compact` hook payload for the `matcher: "*"` question.
