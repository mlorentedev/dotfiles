---
spec: "MEMORY-008-handoff-resume"
verdict: "FAIL"
reviewed_sha: "2e8777c62e7ec24c5780fcafd40fc5207cd96699"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-22"
---
## Adversarial review

**Scope**: MEMORY-008-handoff-resume (slices 0–5, issue #1689), round 2 after the FAIL at `302a427b`, judged as the whole change and not as the delta since round 1. Every round-1 fix was re-tested against the current tree, since a late fix can undo an early one.
**Sources**: `specs/MEMORY-008-handoff-resume/{proposal,tasks,verification,features}.md`, `review-round-1.md`, `review-request.json`; `dotf spec digest MEMORY-008-handoff-resume`; code read: `cli/internal/mem/{resume,thread,thread_store,session_start_resume,session_start_config,handoff}.go`, `cli/internal/cmd/{mem,mem_resume}.go`, `cli/internal/mem/testdata/{resume,session_start_resume}/*`; executed: the six `features.json` commands, `go build ./...`, `go test ./internal/mem/... ./internal/cmd/...`, a throwaway render probe, and two reverted mutations of `resume.go`.

**Note on the stated base.** `e1c5754c` is `chore(main): release 0.58.0` (2026-09-25), the release in effect when the spec was written. It is an ancestor of HEAD, but the range carries **754 commits / 2,444 files** (`git rev-list --count e1c5754c..HEAD` = 754) because the five slice PRs landed on `main` across a month of unrelated work. It is therefore not a bounded diff of this change; this review reads the change through the spec's file set and its own commits inside that range. That is a launcher/base-resolution defect, not this change's, and round 1 already recorded it. It is repeated so the range is not silently read as "the change".

### Contract and digest check

`dotf spec digest MEMORY-008-handoff-resume` returns exactly the three digests in `review-request.json` (`features.json 358ebb49…`, `proposal.md 297a6356…`, `tasks.md 97e4977c…`), re-checked after this review was written. `review-request.json`'s modification in the tree is the launcher's (reviewer + `reviewed_sha`), not mine. `proposal.md`, `tasks.md` and `features.json` are unmodified by this run.

### Spec and task alignment

- **AC1** — met as AC1 words it, and undermined in the field AC1 governs. `RenderResume(text, source, format)` reads no clock, cwd or env; an unknown format is refused by name; `TestResumeIsAPureRenderOfTheThreadFile` pins both formats against `testdata/resume/{context,prompt}.golden` and renders each twice; two live CLI runs on the same file were byte-identical (`cmp` clean), though the pair I ran exercised the ambiguous-caller path, not a thread render. Purity holds — but the pure function is not faithful: see the Major finding.
- **AC2** — verified, and all four round-1 fixes are real. `Verify` renders one command per line and `Awaiting Manu` as a list; an ambiguous caller lists every thread key and picks none. Round-1 F-A is closed by an **allowlist** (`codeRunners`) plus `insideCheckout`, which is the right shape for the class of bug it was. F-B, F-C, F-D and F-G are all present in the code and guarded by tests — I confirmed two of them by mutation rather than by trusting `verification.md` (below).
- **AC3** — verified as written (golden fixture per `source`, byte-exact; `TestSessionStartResumeBySource` also asserts the compacted `ClaudeContext` *begins* with the block; `TestSessionStartResumeInjectsNothingThatIsNotThisAgents` covers both negatives). `startup` also getting the full block is a broadening recorded as a decision, not drift.
- **AC4** — verified: `TestAMemoryShapedLikeDotfilesStaysUnderBudgetWithTheNewFields` and `TestHandoffWriteRefusesAWriteThatCrossesTheByteBudget` pass (f4); the refusal names what to archive; the "may keep or shrink an already-over file" allowance is a recorded decision (ADR-056 §7).
- **AC5 / AC6** — verified by command, not by reading assertions: f5 and f6 both exit 0.
- **Test traceability** — all 43 `Test…` names cited in `verification.md` exist as `func Test…(` in the tree (checked mechanically, zero missing). No ticked box stands on a test that is not there.
- **Build/tests** — `go build ./...` clean; `go test ./internal/mem/... ./internal/cmd/...` → `ok …mem`, `ok …cmd`, 0 failures. All six `features.json` commands run independently: exit 0 each.

**Mutations I ran myself** (each reverted; `git diff` clean afterwards):

| Mutation applied to `resume.go` | Result |
|---|---|
| `SelectThread` ignores `threadWriter` (ownership from the heading stamp only — round-1 F-B) | **killed**: `TestResumeListsCandidatesWhenTheThreadIsAmbiguous` red, naming `feat-j`, `feat-a`, `master@msi` |
| `runnerArgsSafe` accepts every argument (round-1 F-A) | **killed**: `TestReadOnlyCommand` red on all ten escape hatches, including `go test -exec=`, `-toolexec`, `-cpuprofile`, `bats /tmp/evil.bats`, `bats ../outside.bats`, `shellcheck ~/x.sh` |

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | rendering fidelity | `verifyLines` renders only the backticked commands in a `Verify` field and **silently discards every other word in it** whenever the field also holds at least one command. A caveat an author wrote beside the commands ("do not run this on mba") vanishes from `--format prompt`, from `--format context`, and therefore from the block a `compact`/`clear` session is injected with — the one field whose content tells the next session what to execute. This contradicts the package's own stated invariant for the same text (`ParseThread`'s "No line of the body is dropped", enforced by `TestParseThreadDropsNoLine`): the parser keeps it, the renderer deletes it, and no artifact says so. | reproduced with a throwaway test, since removed: a `Verify` of "do NOT run this on mba. \`go test ./...\` is fine, but \`rm -rf /tmp/x\` only after review." rendered as exactly the two commands, both caveats gone. The committed fixture shows the same mechanism benignly: its Verify's "…and…" is absent from both goldens. | `TestResumeRendersVerifyAndAwaiting` covers commands and the unverified mark; **UNTESTED** for prose retention | code + tests (both outside the contract set) |
| Minor | REAL | config SSOT | The `handoff_resume` injector is implemented and default-on but not declared in `session-start-config.json`, whose own guard (`tests/session-start-config.bats`: "has the 11 injector entries with enabled flags") now undercounts the real injector set at 12. Toggles are read as a map with no allowlist, so disabling it works — but the declared SSOT cannot discover it and its count assertion no longer describes the code. Already tracked as #2915 in `verification.md`. | `grep -rn handoff_resume` → `session_start_resume.go` + its own test only; no config entry, no bats entry | `TestSessionStartResumeCanBeTurnedOff` proves the flag; the config-inventory guard does not cover it → **UNTESTED** at the layer that drifted | code/config + tests (outside the contract set) |
| Minor | REAL | verification artifact | `verification.md` carries an empty `## Slice 4 evidence` / `## Slice 5 evidence` pair (lines 102, 104) *before* the filled ones (lines 124, 158), so a reader who stops at the first pair reads slices 4 and 5 as unevidenced. The filled sections exist and are strong; only their position is wrong. | `grep -n '^## Slice [45] evidence'` → 102, 104 (empty) and 124, 158 (filled) | n/a (artifact) | spec artifact — `verification.md`, outside the staleness set, freely editable |
| Question / assumption | THEORETICAL | session hook wiring | `SessionStart`'s matcher `*` firing on `compact`/`clear` is still unobserved against a live payload (round 1 raised it; `verification.md` still says it "is still to be observed"). The goldens prove the code path and the envelope, not that Claude invokes the hook with these `source` values. AC3 is satisfied as written — it asks for golden fixtures, and the proposal declares the envelope as its own risk. | round-1 record; `verification.md` slice 4; no live transcript in the tree | UNTESTED | spec/observation — record it in `verification.md` on the next real `/compact`; if it does not fire, the hook half needs a ticket |
| Question / assumption | THEORETICAL | source normalisation | `sessionResume` matches `source` verbatim, so a payload spelling `Compact` or ` clear` falls to the one-line pointer. Failure mode is quiet degradation, not a wrong context; no evidence any harness sends a non-lowercase source. | `cli/internal/mem/session_start_resume.go` switch | UNTESTED | code (lowercase/trim the parsed source) — optional hardening |
| Minor | SPECULATIVE | context size | Nothing caps the rendered resume, and the render is the whole thread file, so a large thread file is injected in full on every `compact`. The 25,000 B budget bounds `MEMORY.md`, not the per-thread file. Not observed; thread files are operator-written and the recorded probe found nothing near problematic. | code read of `RenderResume`/`ThreadText`; probe figures in `verification.md` slice 2 | UNTESTED | — surface only; do not gate |
| Minor | REAL | spec artifacts (contract set) | `proposal.md`'s acceptance criteria are all still `[ ]` while `verification.md` ticks every one, and `proposal.md` calls the derived entry "about 350 B" where the designed worst case is 450 B (160 + 24 + 72 + count + pointer). Neither is a behaviour defect, and **neither may be edited to satisfy this review**: an edit to the contract set would invalidate any verdict on it. | `proposal.md` AC block; `thread_store.go` cap constants and their 450 B comment | n/a | contract set — carry to the re-review or the archive PR, never a post-review edit |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Every AC is met on the paths its tests exercise and the four round-1 Real findings are closed at the root, but a REAL Major remains: the render silently deletes author-written text from `Verify`, the field that gates what the next session runs. |
| Verification       | A | Reproduced independently: six feature commands exit 0, `go build`/`go test` clean, 43/43 cited tests exist, per-slice mutation tables name the test each mutation turned red, and my own two mutations reproduced their claimed kills. |
| Scope              | B | The spec's file set maps 1:1 to the tasks with no unrelated edits inside `cli/internal/mem`; the graded change is read through those files because the launcher's base yields a 754-commit, 2,444-file range. |
| Reliability        | B | Absent memory, unreadable file, unknown format, ambiguous thread and over-budget write all fail closed with a named message; the write is atomic and preserves the replaced file's mode. The silent render loss is graded under Correctness, not a crash path. |
| Maintainability    | B | Render, parse, store and select are small single-purpose functions with the label set defined once in `ThreadLabels`; the allowlist/`insideCheckout` split carries the most logic and is commented with the reason, not the mechanic. |
| Handoff-readiness  | B | Per-slice evidence, ADR-056, lessons 507/508/513 and follow-up #2915 are recorded; the duplicated empty slice headings and the unticked promotion candidates are the loose ends. |

### Verdict
FAIL

One **REAL Major**, **UNTESTED**, which the codebase contradicts itself on (`ParseThread` promises no line is dropped; the renderer drops them) and which I reproduced. A REAL Major forces FAIL regardless of the rubric; the rubric independently floors at PASS WITH GAPS (one C, no D). The severity path is the more severe of the two.

### Recommended next steps

Minimum set that flips this to PASS — **code and tests only; no contract edit is needed or permitted**, since the invariant the render breaks is already documented in `thread.go`:

1. Make `verifyLines` keep the field's non-command text as written, either in place (prose line, then the command list) or as a trailing "as written" block, in both formats. If the team decides prose in `Verify` is genuinely unsupported, then the renderer must say so where the author will see it — a warning on `handoff-write` for a `Verify` that mixes prose and commands — rather than dropping it silently. Either way the choice is recorded in `verification.md`.
2. A named regression test, red on today's code and green after the fix (the red-green is the evidence, per the evidence rule): e.g. `TestResumeKeepsVerifyProseBesideTheCommands`, asserting the caveat survives in `context` and `prompt`.
3. Nothing else. The remaining items are dispositioned below and do not gate the archive.

For the implementer to disposition in `verification.md` (outside the staleness set, freely editable) — applied, ticketed, or declined with a reason:

- **`handoff_resume` (Minor, REAL)** — #2915 already tracks it; when it lands, make the bats injector guard derive its count from the code or enumerate the new entry, instead of asserting a stale 11.
- **`verification.md` duplicate headings (Minor, REAL)** — delete the empty 102/104 pair or move the filled sections under it, before archiving.
- **Live `source=compact` payload (Question)** — observe on the next real compaction and record the transcript; if the matcher does not fire, ticket the hook half of AC3.
- **Source normalisation, render size (THEORETICAL / SPECULATIVE)** — carry into #2915 or leave as noted risks; neither gates this change.
- **Contract-set notes** — the unticked AC boxes in `proposal.md` and the "about 350 B" wording are for the archive PR to reconcile. After this FAIL is fixed, the next review round reads the contract as it then stands.

### Advisability of `dotf spec archive`

**Not advisable yet.** This review is fresh and describes the current tree, but it is a FAIL: archiving now would either be refused by the gate or would archive a change whose renderer silently discards part of the record it exists to carry. The fix above is small and needs one more review round — which is the mechanism, not an inconvenience.
