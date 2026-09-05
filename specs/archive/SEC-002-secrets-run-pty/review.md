---
spec: "SEC-002-secrets-run-pty"
verdict: "PASS"
reviewed_sha: "f7fee848e5f907d16e04e75f619075862edbeea3"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-04"
---

## Adversarial review

**Scope**: SEC-002-secrets-run-pty (round 2, against the new head `f7fee84`)
**Sources**: `specs/SEC-002-secrets-run-pty/{proposal,tasks,verification,features}.json`, plus the
implementation diff `git diff 5cc14ba..HEAD` (merge-base `main`), commits `a636844` + `f7fee84`.

Reviewed as `nan/deepseek-v4-flash` (drawn from `harness/reviewer-pool.json`). Round 1 reviewed
`a636844` and returned **FAIL** on one REAL Major (the `isTerminal` seam never exercised). This is the
round-2 re-run against `f7fee84`, which the implementer's `verification.md` "Adversarial review round
1 — dispositions" says applied five of six findings. I re-ran the build, vet (Linux + Windows), the
full test suite, golangci-lint at the pinned 2.12.2, and both mutation proofs. All evidence below was
produced in this session; the working tree is clean apart from this review and the launcher's
`review-request.json` / `review-transcript.jsonl`.

### Spec and task alignment

- **The round-1 Major is resolved.** `secrets_select_test.go` now carries
  `TestWantsInteractiveChild_FollowsTheTerminalSeam`, which sets the `isTerminal` package variable
  to stubs (true and false), drives `wantsInteractiveChild()` both ways, and additionally asserts the
  fd consulted is **stdout** (`os.Stdout.Fd()`), not stdin/stderr. I confirmed the test is live by
  inverting the condition: `return interactiveChildSupported() && !isTerminal(os.Stdout.Fd())` makes
  both assertions fail (`a terminal parent did NOT select the pty path` and `a non-terminal parent
  selected the pty path`). Restored. This is exactly the linchpin the round-1 review said had no test.
- **All seven acceptance criteria are met by named, reproducible tests.** AC1
  (`TestWantsInteractiveChild_FollowsTheTerminalSeam` + `TestRunChildPTY_ChildSeesATerminal` +
  `TestRunChild_ChildSeesAPipe`), AC2 (`TestRunChildPTY_RedactsASecretSplitAcrossWrites`), AC3
  (`TestRedactWriter_ReleasesFrameWithNoSecretPrefixImmediately` +
  `TestRedactWriter_HoldsBackATrailingSecretPrefix`), AC4 (`TestRunChildPTY_HonoursTheIntrospectionGuard`),
  AC5 (the pipe path regression test + e2e `pi -p` evidence), AC6 (`GOOS=windows go vet ./...` clean),
  AC7 (verification.md reports byte counts / exit codes from a scratchpad rebuild).
- **AC3 is genuinely proven, not asserted.** I reverted `holdBack` to the old fixed
  `maxSecretLen-1` window (anchor-asserted patch) and both hold-back tests went red with the exact
  failure the proposal describes (`got: "\x1b[2J\x1b["`, a frame's tail held dark). Restored. The
  mutation proof in `verification.md` is real.
- **Root cause and second defect are correctly diagnosed.** `exec.Cmd` passes a descriptor through
  only when the writer's dynamic type is `*os.File`; a `redactWriter` never is, so every child got an
  `os.Pipe`. The fixed hold-back window was the more important defect; a pty alone would not have
  fixed the rendering.
- **`features.json` deviation is justified by the template, not a silent no-op.** The implementer
  populated `evidence` for all eight features but kept `state: "pending"`, citing the template's
  pass-state gating. I read `cli/internal/spec/templates/tasks.md`:
  *"the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and
  capturing exit code 0, may set that terminal state."* So the round-1 recommendation to set
  `state: "pass"` was itself wrong; the implementer's decline is correct. `state: "pending"` with
  populated `evidence` is the correct contract state for an agent-written file.
- **Archive gate does not depend on feature state.** I read `checkReviewGate`
  (`cli/internal/spec/review.go:280`): it checks waiver, presence, `review.Spec == specID`,
  provenance, `review.Verdict.Blocks()`, staleness against `reviewed_sha`, and the reviewer pool. It
  never reads `features.json` state. So the pending state cannot block archive.
- **No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags** in `proposal.md`, `tasks.md`, `features.json`,
  or `verification.md`. (The only match in the folder is the launcher's own session transcript,
  `review-transcript.jsonl`, which is not a contract file.)
- **Out-of-scope items are untouched.** `cmd/spec.go:92`, the `.zshrc`/`.bashrc` wrappers, `opencode`,
  and the `script(1)` stopgap are all outside the diff. The `creack/pty v1.1.24` dependency is
  declared and pinned in `cli/go.mod`/`go.sum`; `GOOS=windows go vet ./...` stays clean.

### Findings

Round-1 findings and their dispositions, re-verified against the new head:

| Round-1 finding | Disposition | Status at `f7fee84` |
|---|---|---|
| Major, REAL — `isTerminal` seam never exercised | APPLIED | **Resolved.** `TestWantsInteractiveChild_FollowsTheTerminalSeam` exercises the seam both ways and asserts the fd is stdout; mutation-proven. |
| Minor, REAL — `features.json` all-pending, empty evidence | APPLIED, half-declined (state stays pending per template) | **Resolved.** All 8 features carry `evidence`; `state: "pending"` is the template-correct value for an agent-written file. |
| Minor, THEORETICAL — partial secret prefix flushed raw at EOF | APPLIED | **Resolved.** `Flush` replaces a proper-prefix tail with the secret's placeholder; `TestRedactWriter_DoesNotFlushAPartialSecretPrefixRaw` passes. |
| Minor, THEORETICAL — `runChildPTY` hang if grandchild holds pty slave | DECLINED, documented (proposal decision 7) | **Accepted limitation.** A deadline would truncate a legitimately long-running TUI, and the pipe path has the same property. Not a blocker. |
| Minor, REAL — pty CRLF translation undocumented | APPLIED as documentation (proposal decision 6) | **Resolved.** The proposal now states the `\n`→`\r\n` (ONLCR) translation. |
| Minor, SPECULATIVE — prefix-collision in `ReplaceAll` order | APPLIED (sort longest-first) | **Resolved for the same-buffer case.** `newRedactWriter` sorts pairs longest-first; `TestRedactWriter_HandlesOneSecretBeingAPrefixOfAnother` passes. |

Residual / new findings on the current head:

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | signals | The pty path registers only `SIGTERM`, not `SIGINT`, unlike the pipe path (`runChild` registers `os.Interrupt`). In raw mode this is correct (Ctrl-C becomes a `^C` byte via the pty's line discipline), but when **stdin is not a terminal** — so `term.MakeRaw` is skipped — the parent's controlling terminal still generates SIGINT, which the parent does not catch, so `^C` kills `dotf` and orphans the child (which then receives SIGHUP from the closed pty master). Narrow, but an inconsistency with the documented pipe-path behavior. | Code read of `runChildPTY` (registers `syscall.SIGTERM` only) vs `runChild` (registers `os.Interrupt, syscall.SIGTERM`). Not observed. | UNTESTED (no test exercises the pty path with a non-terminal stdin) | code (register `os.Interrupt` when raw mode is not set, or document as accepted) |
| Minor | THEORETICAL | reliability | A non-EIO error from `io.Copy(out, ptmx)` is silently swallowed (`_ = copyErr`). If writing to the redactWriter's target (parent stdout) fails — broken pipe, disk full — the failure is discarded and the child's exit code is returned, so the caller learns nothing about lost output. | Code read of `runChildPTY`'s `io.Copy` error handling. Not observed. | UNTESTED | code (surface a non-EIO copy error at least on stderr, or propagate) |
| Minor | THEORETICAL | redaction | Residual prefix-collision across writes: if one injected secret is a proper prefix of another and the longer one is written split exactly at the shorter one's length, the shorter (a complete secret) is replaced first and the longer one's suffix is emitted. Sorting longest-first fixes only the same-buffer case. Vanishingly unlikely (needs two prefix-related secrets AND an exact write boundary); pre-existing in spirit. | Code read of `Write`/`holdBack` + the sort; `TestRedactWriter_HandlesOneSecretBeingAPrefixOfAnother` covers only the single-write case. Not reproduced. | `TestRedactWriter_HandlesOneSecretBeingAPrefixOfAnother` (single-write only) | code (surface only; do not gate) |
| Minor | THEORETICAL | consistency | `wantsInteractiveChild()` reads `os.Stdout.Fd()`, while the redactWriter target is `cmd.OutOrStdout()`. These coincide in production but diverge if a caller sets `cmd.SetOut(...)`, so the branch decision and the output sink could disagree. | Code read of the call site in `newSecretsRunCmd`. | `TestWantsInteractiveChild_FollowsTheTerminalSeam` (asserts the fd is `os.Stdout.Fd()`, which is the code's behavior) | code (document, or consult `cmd.OutOrStdout()`) |

Documented, accepted tradeoffs (not findings, but stated for the record): the pty path merges
stdout/stderr onto one stream (proposal decision 2); redaction stays on always, with a length-shifting
placeholder that can corrupt a TUI's cursor arithmetic (decision 5); the grandchild-holds-pty hang is
accepted (decision 7); the pipe path keeps two redactWriters (pre-existing).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All acceptance criteria met and mutation-proven; minor negative-path gaps (SIGINT non-raw-mode edge, swallowed non-EIO copy error, divergent-prefix release) but no observed defect in the criteria paths. |
| Verification       | B | Go-level evidence fully reproducible (I re-ran build/vet/test/lint and both mutation proofs); the `pi` e2e is reported as byte counts but not independently reproducible without the model/pty harness. |
| Scope              | A | Diff matches the proposal exactly; every out-of-scope item untouched; the dependency is declared and pinned. |
| Reliability        | B | Raw-mode restore (defer + signal handler), EIO handling, and exit-code propagation are sound; the grandchild hang and swallowed non-EIO error are partial gaps. |
| Maintainability    | B | Clear WHY comments and low cyclomatic complexity; `runChildPTY` is ~100 lines, exceeding the <40-line threshold, but structured and readable. |
| Handoff-readiness  | B | Spec fully updated; lesson flagged as a candidate but not yet written (deferred); PR not yet opened. |

### Verdict
**PASS**

No Blocker or Major finding remains. The single round-1 Major (the untested `isTerminal` seam) is
resolved by a named, mutation-proven test. The rubric is all-B with no C or D, which mechanically
maps to PASS, and the severity axis has only Minor/THEORETICAL/SPECULATIVE findings. The residual
findings are real but do not rise to Major, and none of them moves the verdict. The `state: "pending"`
in `features.json` is template-correct and does not block archive.

### Recommended next steps (before archive)
1. **Surface the two Minor/THEORETICAL reliability gaps** if you want them closed: forward `SIGINT`
   on the pty path when raw mode is not set (or document the non-raw-mode behavior as accepted), and
   do not silently swallow a non-EIO `io.Copy` error.
2. **Write the flagged lesson** ("an `io.Writer` that is not an `*os.File` silently becomes a pipe")
   to `docs/lessons/` before archiving — it is a cross-cutting gotcha the `cmd/spec.go:92` sibling
   still carries, and it is currently deferred.
3. **Open the PR** referencing the spec folder (a closing task is still `[ ]`).
4. **Populate `state: "passing"` is NOT an agent action** — leave it for the harness after it runs
   `verification` and captures exit 0. The current `pending` + populated `evidence` is correct.

`dotf spec archive` is **advisable in the current state**: the verdict is PASS, the `reviewed_sha`
matches HEAD, the reviewer is in the pool, and the gate does not check feature state. The two
Minor/THEORETICAL reliability items are worth a follow-up but do not block archive.
