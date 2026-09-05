---
spec: "SEC-002-secrets-run-pty"
verdict: "FAIL"
reviewed_sha: "a636844981f0804395d9d5832532267879403c07"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-04"
---

## Adversarial review

**Scope**: SEC-002-secrets-run-pty
**Sources**: `specs/SEC-002-secrets-run-pty/{proposal,tasks,verification,features}.json`, plus the
implementation diff `git diff 362783b..HEAD` (merge-base `main`), commit `a636844`.

Reviewed as `nan/deepseek-v4-flash` (drawn from `harness/reviewer-pool.json`). I read the spec
triad first, then the implementation, then ran the build, vet (Linux + Windows), the test suite,
golangci-lint at the pinned 2.12.2, and a mutation that reverts `holdBack` to the old fixed-window
rule. All commands produced evidence below; the working tree is clean apart from this review and the
launcher's `review-request.json`.

### Spec and task alignment

- **Root cause is real and correctly diagnosed.** `exec.Cmd` passes a descriptor through only when
  the writer's dynamic type is `*os.File`; a `redactWriter` never is, so every child got an
  `os.Pipe`. The fd-level evidence in the proposal is credible and the fix direction (pty when the
  parent owns a terminal, pipe otherwise) is sound.
- **The second defect (fixed hold-back window) is the more important of the two.** I independently
  confirmed this: reverting `holdBack` to the old `maxSecretLen-1` rule makes
  `TestRedactWriter_ReleasesFrameWithNoSecretPrefixImmediately` and
  `TestRunChildPTY_RedactsASecretSplitAcrossWrites` go red, with the exact failure mode the proposal
  describes (a TUI frame's tail held dark). A pty alone would not have fixed the bug.
- **Tasks `[x]` claims map to diff evidence for the code.** `runChildPTY`, `secrets_child_windows.go`,
  the `holdBack` change, the `isTerminal` seam, and all four new tests are present in the diff.
- **One task is ticked without the diff evidence AC1 demands.** The `isTerminal` seam is added and
  two tests exercise the pty/pipe *bodies*, but no test exercises the seam itself. See Finding 1.
- **Out-of-scope items are correctly left untouched.** `cmd/spec.go:92`, the rc wrappers, `opencode`,
  and the `script(1)` stopgap are all outside the diff.
- **The dependency decision (creack/pty v1.1.24) is settled in tasks.md** and present in
  `cli/go.mod`/`go.sum`. `GOOS=windows go vet ./...` stays clean.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | tests / spec | AC1 says the tests must verify branch selection "by exercising the `isTerminal` seam in both directions"; the seam exists but no test exercises it. `isTerminal` is referenced only at the call site (`secrets.go:512`) and its definition (`:635`), never in a test. The two AC1 tests (`TestRunChildPTY_ChildSeesATerminal`, `TestRunChild_ChildSeesAPipe`) call `runChildPTY`/`runChild` directly, bypassing the call-site branch condition. The linchpin of the fix — the one-line `if interactiveChildSupported() && isTerminal(os.Stdout.Fd())` selection — is unverified; an inverted condition would pass every test while silently reverting the fix. | `grep -rn isTerminal cli/internal/cmd/` returns only the two `secrets.go` hits; no `_test.go` references it. All SEC-002 tests pass without touching the seam. | UNTESTED (no named test exercises the `isTerminal` seam / call-site selection) | tests |
| Minor | REAL | spec / handoff | `features.json` has all 5 features at `state: "pending"` with empty `evidence`, even though `verification.md` documents the proof. This does not block `dotf spec archive` (I read `Archive`/`checkReviewGate`: neither validates feature state), but the machine-readable contract file is left incomplete, and the SDD gate's downstream consumers see every feature as unverified. | `cat specs/SEC-002-secrets-run-pty/features.json` — all `state: "pending"`, all `evidence: ""`; `verification.md` carries the proof that `features.json` does not. | UNTESTED (no test reads feature state; it is a contract-file completeness issue) | spec (populate `features.json` before archive) |
| Minor | THEORETICAL | redaction | A partial secret prefix at end-of-stream is emitted unredacted. When the child's output ends on a run that is a proper prefix of a secret and the child exits, `runChildPTY` returns, the call site calls `merged.Flush()`, and `Flush` replaces only whole secrets — the held prefix is written out as-is. Up to `maxSecretLen-1` bytes of a key's prefix can reach stdout. Not a regression (the old fixed-window flush had the same property), but it is a real redaction edge. | Code read of `redactWriter.Flush` + the `holdBack` path; not observed. `TestRedactWriter_HoldsBackATrailingSecretPrefix` completes the secret and never exercises the flush-of-partial-prefix case. | UNTESTED (no test for the partial-prefix-at-EOF flush) | code (accept as tradeoff, or drop the tail on flush of a non-whole prefix) |
| Minor | THEORETICAL | reliability | `runChildPTY` can hang if the child leaves a descendant holding the pty slave fd. `io.Copy(out, ptmx)` blocks until the master read yields EIO; EIO only arrives once every fd to the slave is closed. If a child daemonises or forks a long-lived process that inherits the slave, EIO never arrives, `c.Wait()` is never reached, and the command hangs without reaping the child. | Code read of `runChildPTY`: `io.Copy` runs before `c.Wait()` and there is no timeout or deadline. Not observed. | UNTESTED (no test with a forking/daemonising child) | code |
| Minor | REAL | behavior change | The pty path translates `\n` to `\r\n` (terminal ONLCR), so a non-TUI child run under a terminal now emits CRLF where the pipe path emitted LF. This is correct for a TUI (it matches running the child directly) and the proposal's decision #2 frames the pty path as "closer to the unwrapped behaviour", but the proposal never states the CRLF translation, and no test asserts the exact pty output (the redaction test checks only `strings.Contains`). | Observed in the mutation run: `TestRunChildPTY_RedactsASecretSplitAcrossWrites` failed with `got "mock-openrouter-test-token-val\r\n"` under the reverted rule. `grep` shows no mention of CRLF/ONLCR in `proposal.md` or `secrets_child_unix.go`. | `TestRunChildPTY_RedactsASecretSplitAcrossWrites` (asserts contains, not exact bytes) | spec (document) |
| Minor | SPECULATIVE | redaction | Redaction prefix-collision: if two injected secrets have a prefix relationship (e.g. `abc` and `abcdef`), `ReplaceAll` processes them in `injected` order, so the shorter secret's replacement can break detection of the longer one, leaking the longer secret's suffix. Astronomically unlikely with real random API keys, and pre-existing (the old code had the same `ReplaceAll` ordering), but it is a hole in the redaction logic this change touches. | Code read of `newRedactWriter`/`Write` (pairs processed in order); no reproduction. | UNTESTED | code (process pairs longest-first) — surface only, does not gate |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Behavior is correct and verified, but AC1 is only partially met (the `isTerminal` seam is never exercised) and the branch-selection path is unverified. |
| Verification       | B | Strong: mutation proof landed (I re-ran it), e2e binary evidence, reproducible `go test` commands — but the pty e2e is a manual harness, not a committed test, and `features.json` evidence is empty. |
| Scope              | A | Diff matches the proposal exactly; no creep; every out-of-scope item is untouched. |
| Reliability        | B | Signals, raw-mode restore (defer + handler), and EIO handling are sound; the grandchild-holds-pty hang and swallowed non-EIO copy error are partial gaps. |
| Maintainability    | B | Clear WHY comments, low cyclomatic complexity, functions mostly under threshold; `runChildPTY` is ~100 lines but structured and readable. |
| Handoff-readiness  | C | `features.json` left all-pending with no evidence; the lesson is flagged as a candidate but not written; proposal/tasks/verification otherwise complete. |

### Verdict
**FAIL**

One **REAL Major** (AC1's `isTerminal` seam never exercised; the call-site branch selection —
the linchpin of the fix — is untested) forces FAIL per the severity axis, independent of the rubric.
The rubric alone (Correctness C, Handoff-readiness C, no D) would be PASS-WITH-GAPS, but the more
severe path (a Major finding) governs. The SPECULATIVE prefix-collision finding is surfaced and does
not move the verdict.

### Recommended next steps (before archive)
1. **Add the missing seam test (the blocker).** Write a test that sets the `isTerminal` package
   variable to a stub (true and false) and invokes the call-site selection (or the command) to
   assert the pty path is taken when `isTerminal` returns true and the pipe path when it returns
   false. This is what AC1 names and what the seam exists for. Restore the variable in a
   `t.Cleanup`. Name it, e.g., `TestRunChild_SelectsPTYWhenParentHasATerminal` /
   `TestRunChild_SelectsPipeWhenParentHasNoTerminal`, and reference it in `features.json`.
2. **Populate `features.json`** with `state: "pass"` and the `evidence` commands, matching the
   `verification.md` table, before `dotf spec archive`.
3. **Write the lesson** flagged in `verification.md` (an `io.Writer` that is not an `*os.File`
   silently becomes a pipe) to `docs/lessons/`, since it is a cross-cutting gotcha that the
   `cmd/spec.go` sibling still carries.
4. **Minor / optional:** document the pty CRLF translation in the proposal's "Decisions" section;
   consider a non-hanging strategy (deadline or detach) for the grandchild-holds-pty case; and
   process redaction pairs longest-first to close the prefix-collision hole.

`dotf spec archive` is **not advisable in the current state**: it will (correctly) refuse on the
`review.md` verdict. Once the seam test is added, `features.json` populated, and this review re-run
against the new head, the verdict can flip to PASS. The code fix itself is sound and well-evidenced;
the FAIL is driven by the missing seam test, not by a defect in the pty or redaction logic.
