---
spec: "GUARD-004-triage-loop-closure"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "12d0116bd0291bc020bf4cb0d0b8f5d24b2cace3"
reviewer: "nan/mimo-v2.5"
date: "2026-08-22"
---

## Adversarial review

**Scope**: GUARD-004-triage-loop-closure (PR #1101, commit 44c9417)
**Sources**: `specs/GUARD-004-triage-loop-closure/{proposal,tasks,verification}.md`, `git diff 44c9417^..44c9417`, full test suite run against HEAD

### Spec and task alignment

- **AC1 (skill records marker)**: Verified. `harness/skills/pr-review-triage/SKILL.md` step 7 posts the table under the registry's heading. Vault SSOT (`00_meta/skills/pr-review-triage/SKILL.md`) carries the source. Regenerated record matches. `grep -c "## Review triage"` confirms 2 occurrences in both.
- **AC2 (marker quoted from registry)**: Verified. Step 7 references `harness/review-attestation.json` by path and names `triage.marker`. The string is not restated inline.
- **AC3 (both assembly paths)**: Verified. `TestBriefSkipsTriageWhenProbeIsNil` proves the agnostic brief renders the section when a probe is wired. `TestClaudeContextCarriesTriageSection` proves the Claude adapter does. Both pass at HEAD.
- **AC4 (no registry → no section)**: Verified against `~/Projects/knowledge` (no reviewer registry). Zero `pr-triage` lines.
- **AC5 (present but unanswerable → loud)**: Verified with `gh` removed from PATH. Message reads `queue could not be computed: ... This is not an empty queue.` `TestTriageQueueSection/"a failure is reported, never rendered as empty"` asserts this.
- **AC6 (bounded)**: Verified. `Fetch` takes `context.Context`; session-start caller passes 5s deadline via `context.WithTimeout`.
- **AC7 (behaviour-preserving refactor)**: `dotf pr triage-queue` output confirmed unchanged before/after extraction. `prtriage.Fetch` is a clean extraction of the command's original logic.
- **AC8 (honest package comment)**: Verified. `session_start.go` now says "INTENDED to consume" and documents the measured ceiling. `mem/session_start.go` records the deploy-time staleness constraint for future consumers.
- **AC9 (all checks green)**: Verified against HEAD. `go build ./...` pass, `go vet ./...` pass, `GOOS=windows go vet ./...` pass, `go test ./...` pass (0 failures across 17 packages), `golangci-lint run` 0 issues (v2.12.2, matching pin), bats suite 1374 pass, 0 failures.
- **Doctrine floor**: `harness/enforced/pr-stewardship.md` carries the triage checkpoint instruction. `AGENTS.md` and `ai/claude/CLAUDE.md` carry it via `compile-harness.sh --refresh`. `.pr_agent.toml` includes both in `repo_context_files`, so PR-Agent reads the same doctrine.

All acceptance criteria are met. All tasks are ticked. The verification artifacts prove the criteria with reproducible commands and observed output.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | prtriage.Fetch | `--limit 100` silently truncates when a repository has >100 open PRs. No warning is emitted and no pagination is attempted. For this repo (3 open PRs) it is irrelevant; for a busy upstream fork or a monorepo with release branches, pending PRs beyond the 100th are dropped from the queue without notice. | `fetch.go:47` hardcodes `"--limit", "100"` | UNTESTED | code (add pagination or warn when len == 100) |
| Minor | THEORETICAL | mem assembly | No integration test exercises the error path through the agnostic brief assembly. `TestBriefSkipsTriageWhenProbeIsNil` tests nil and non-error probes; `TestTriageQueueSection` tests the renderer in isolation. The glue where `Brief()` calls `opts.TriageQueue()` and the error propagates into `triageQueue()` is untested at the assembly level. | `session_start.go:80-82` — `if opts.TriageQueue != nil { brief += triageQueue(opts.TriageQueue()) }` | UNTESTED | tests (add a case to `TestBriefSkipsTriageWhenProbeIsNil` where the probe returns an error) |
| Minor | THEORETICAL | mem assembly | Same gap for the Claude adapter. `TestClaudeContextCarriesTriageSection` verifies the section renders on success but does not test the error path. The error rendering is proven by `TestTriageQueueSection`, but the integration from `ClaudeContext()` through to the rendered string is not. | `session_start_adapter.go:62-64` | UNTESTED | tests (add a case to `TestClaudeContextCarriesTriageSection` where the probe returns an error inside a .git dir) |
| Minor | THEORETICAL | pr.go double-read | `dotf pr triage-queue` calls `Fetch` (which calls `LoadRegistry`) and then `Marker` (which calls `LoadRegistry` again). The registry is read from disk twice in the same command invocation. Both are short-lived, so this is a correctness-safe inefficiency, not a bug. | `pr.go:53-63` — `Fetch` then `Marker` | UNTESTED | code (pass `Registry` through rather than re-reading) |
| Minor | THEORETICAL | skill step 7 | No guard against posting duplicate `## Review triage` comments if the skill is run twice on the same PR. The queue reads the newest comment's timestamp, so a duplicate does not break the loop — it adds noise and could confuse a human reader comparing two identical tables. | `harness/skills/pr-review-triage/SKILL.md:137-162` — step 7 posts unconditionally | UNTESTED | spec (add "if a `## Review triage` comment already exists from this session, edit it rather than post a new one" to step 7) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All acceptance criteria verified, three-state rendering tested, error-wins-over-summary case pinned, empty-queue-is-silent case pinned |
| Verification       | B | Evidence proves each criterion with reproducible commands; agnostic error-path integration is a minor gap |
| Scope              | A | Diff matches proposal exactly: skill step 7 + session-start probe + doctrine floor + Fetch extraction. No unrelated changes. |
| Reliability        | B | Error paths handled, 5s deadline enforced, nil-safe throughout. The `--limit 100` truncation is a reliability edge case at scale. |
| Maintainability    | A | Clear naming, Fetch extraction eliminates duplication, package comment documents measured ceiling and constraints honestly |
| Handoff-readiness  | B | Spec, tasks, verification all present and accurate. Lesson capture deferred (not a gap per spec). The stale `session_start.go` comment from the prior revision is corrected. |

### Verdict

**PASS WITH GAPS**

No Blockers or Majors. Five Minor findings, all THEORETICAL and UNTESTED. The rubric produces all B or above (no C, no D), so the rubric path yields PASS. The severity path yields PASS-WITH-GAPS due to five Minor findings. The more restrictive path wins.

The change is correct, well-scoped, and the test suite pins the critical invariant (error ≠ empty queue). The gaps are in edge-case coverage (`--limit 100` truncation, error-path integration at the assembly layer) and in cosmetic resilience (duplicate triage comments). None blocks archive.

### Recommended next steps (before archive)

1. **Optional**: add error-path integration cases to `TestBriefSkipsTriageWhenProbeIsNil` and `TestClaudeContextCarriesTriageSection` — low effort, closes the minor testing gap.
2. **Optional**: address the `--limit 100` truncation (paginate or warn when `len(results) == 100`) — only matters if this repo or its downstreams approach 100 open PRs.
3. **Optional**: deduplicate `LoadRegistry` calls in `pr.go` by passing the `Registry` through — cosmetic, no correctness impact.
4. **Optional**: add a "skip if already triaged this session" guard to skill step 7 — cosmetic, no loop impact.

None of these are required for archive. The spec is implemented, verified, and the loop is closed.
