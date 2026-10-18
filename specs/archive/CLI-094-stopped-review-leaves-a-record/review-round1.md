---
spec: "CLI-094-stopped-review-leaves-a-record"
verdict: "FAIL"
reviewed_sha: "eadb884d814e787e39696c6353552ad5737832bd"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-17"
---
## Adversarial review

**Scope**: CLI-094-stopped-review-leaves-a-record
**Sources**: specs/CLI-094-stopped-review-leaves-a-record/{proposal,tasks,verification}.md, git diff f13d0b606a6818ec617b2c5deff203c692f090e5...HEAD

### Spec and task alignment
- Implementation addresses both #1975 (wedged reviewer stopped before the 45m cap) and #2448 (write-first provisional review with refusal of `verdict: PENDING` at archive gate).
- Acceptance criteria 1 through 6 are implemented and matched in `features.json`.
- However, an adversarial integration analysis revealed a critical flaw in the transcript pipeline: in production (`dotf spec review`), `dotf spec deadline` runs piped into `dotf spec transcript-sink`. Because `SinkTranscript` in `cli/internal/spec/transcript.go` buffers writes in a `bufio.Writer` (4 KB) without flushing after non-incremental events, writes under 4 KB remain in user-space memory and are not committed to disk until EOF. When a tool call wedges, `LastToolCall` inspects the disk file and fails to extract the wedged tool call (returning empty string), and active runs with sparse output can trigger premature false stall terminations.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | stall-detection / transcript-sink | `SinkTranscript` buffers auditable event writes in `bufio.NewWriter(f)` (4 KB) without calling `w.Flush()` after non-incremental frames. In detached review mode (`dotf spec review`), `dotf spec deadline` runs piped into `transcript-sink`. Because writes remain buffered in memory until process exit, the file on disk remains 0 bytes (or an older multiple of 4 KB). When a tool call wedges, `LastToolCall` reads the file on disk and finds nothing (or an older call), failing to name the wedged tool call on stderr. Additionally, active runs whose output has not reached 4 KB can suffer false premature stall timeouts. | Live reproduction: running `go run ./cmd/dotf spec deadline 10s --stall-timeout 2s --transcript /tmp/t.jsonl -- sh -c '... | go run ./cmd/dotf spec transcript-sink /tmp/t.jsonl'` exited 125 but completely omitted `[review] last tool call recorded: bash: wedged_command`. With `w.Flush()` added after writing auditable frames, the exact same command properly detected the stall and printed the last tool call. | UNTESTED (existing tests `TestRunWithLimitsStopsARunnerWhoseTranscriptGoesSilent` and `TestSpecDeadlineStopsAStalledRunAndNamesTheLastCall` pre-populated the transcript via `os.WriteFile`, bypassing `transcript-sink`) | code (`cli/internal/spec/transcript.go`) + tests |
| Minor | REAL | review-record / StoppedRunRecord | `StoppedRunRecord` in `cli/internal/spec/stall.go` hardcodes `"review.md"` in all four message strings. When reviewing with `--second`, `slot.Review` is `review-second.md`. If a second signer run stalls or hits the cap, stderr misleadingly claims that `review.md` was not rewritten or is PENDING, even though `review.md` (the first signer's passing verdict) was untouched and `review-second.md` was the affected file. | Inspected `cli/internal/spec/stall.go` lines 152-166: `StoppedRunRecord` ignores `filepath.Base(reviewPath)` and returns string literals containing `"review.md"`. | UNTESTED | code (`cli/internal/spec/stall.go`) |
| Minor | THEORETICAL | stall-detection / LastToolCall | `LastToolCall` parses only `pi`'s JSON schema (`tool_execution_*` events with `args.command` or `args.path`). For `agy` pool members (`agy/gemini-3.1-pro-high`, `agy/gemini-3.8-flash-high`), `LastToolCall` returns `""`, so wedged `agy` reviews will report stall termination without naming the wedged tool call. | Code inspection of `cli/internal/spec/stall.go` lines 79-85 and 108. | UNTESTED | code (`cli/internal/spec/stall.go`) |
| Minor | REAL | spec-contract / proposal.md | In `proposal.md`, all acceptance criteria checkboxes under `## Acceptance criteria` (lines 46-51) are unticked (`- [ ]`), despite implementation and verification claims in `tasks.md`, `features.json`, and `verification.md`. | `specs/CLI-094-stopped-review-leaves-a-record/proposal.md` lines 46-51. | N/A | spec (`specs/CLI-094-stopped-review-leaves-a-record/proposal.md`) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Unbuffered disk writes in `SinkTranscript` prevent `LastToolCall` from naming the wedged call in the live tmux pipeline. |
| Verification       | B | Component tests are thorough but missed integration between `spec deadline` and `transcript-sink`. |
| Scope              | A | Diff matches proposal exactly; focused on launcher limits and provisional review contract. |
| Reliability        | C | Buffering in `SinkTranscript` risks premature stall timeouts and suppresses diagnostic output on wedge stops. |
| Maintainability    | A | Clean decomposition, CC ≤ 10, functions under 40 lines, strict typing, and cross-platform consideration. |
| Handoff-readiness  | B | Comprehensive verification and measurement notes; proposal checkboxes remain unticked. |

### Verdict
FAIL

`dotf spec archive` is **not advisable** in the current state and will refuse this FAIL verdict.

### Recommended next steps

#### Outside contract set (can be fixed directly):
1. **Fix transcript buffering (code)**: In `cli/internal/spec/transcript.go` (`SinkTranscript`), add `w.Flush()` immediately after writing each non-incremental auditable frame (`w.WriteString(line)`), ensuring real-time disk persistence for `statTranscript` and `LastToolCall`.
2. **Add pipeline regression test (tests)**: Add an integration test in `cli/internal/cmd/spec_deadline_unix_test.go` or `cli/internal/spec/` executing `spec deadline` concurrently with `spec transcript-sink`, verifying that a wedged command emitted to the sink is correctly captured and printed by `LastToolCall`.
3. **Fix review filename reporting (code)**: In `cli/internal/spec/stall.go` (`StoppedRunRecord`), use `filepath.Base(reviewPath)` instead of the hardcoded string `"review.md"`, so second signer runs (`review-second.md`) report their filename accurately.

#### Inside contract set (requires re-review after modification):
4. **Update proposal checkboxes (spec)**: Once the buffering defect is fixed and verified, update the acceptance criteria checkboxes in `proposal.md` to `[x]` and re-run adversarial review.
