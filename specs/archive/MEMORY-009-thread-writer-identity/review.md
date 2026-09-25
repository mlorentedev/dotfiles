---
spec: "MEMORY-009-thread-writer-identity"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "d4b77425e5ddc6be5cceb0e372ca983b44b557c2"
reviewer: "nan/glm5.3-flash"
date: "2026-09-25"
---

## Adversarial review

**Scope**: MEMORY-009-thread-writer-identity (#1690), diff `29947d1a9976656c60e28a7e9b665d23981d2be2...HEAD` (d4b7742)
**Sources**: `specs/MEMORY-009-thread-writer-identity/{proposal,tasks,verification,features.json}`; `git diff 29947d1a...HEAD` restricted per `verification.md` §Review scope to `cli/internal/mem/handoff.go`, `cli/internal/cmd/mem_handoff.go`, their tests, `harness/skills/handoff/SKILL.md`, and the spec folder.

### Spec and task alignment

- Slice 1 (commit `f6dc6f6`, merged as #1711/`6436c71`) implements the stamp + fork; slice 2 (`ffc1442`) makes the handoff skill pass `--agent` after 0.59.0 shipped the flag. Both slices are present at HEAD; all six slice-1 tasks and the slice-2 task carry diff evidence.
- The launcher's base spans commits belonging to other specs (#1714, #1715, #1716, #1721, #1724, the 0.59.0 release). I confirmed by `git show ffe5c44` that the `mem.ThreadWarnings` wiring inside `mem_handoff.go` is #1716's (MEMORY-008), not this spec's, and judged it out of scope as `verification.md` declares. No other spec's code was attributed to this one.
- AC-by-AC, each re-verified fresh in this session (`go build`, `go vet`, `go test ./internal/mem/... ./internal/cmd/... -count=1` all ok; `golangci-lint run ./internal/mem/... ./internal/cmd/...` → 0 issues; all five `features.json` commands re-run, exit 0):
  - **AC1** — `TestAnotherAgentsStampedBlockIsForkedNotReplaced` (stamped) and `TestAnUnstampedBlockIsAttributedByItsJournalLine` (journal attribution, incl. other-project paths, thread-suffixed names, markdown links). Mutation-verified by me: disabling the fork branch made both red; removing journal attribution made the second red; both reverted, tree clean.
  - **AC2** — `TestMemHandoffWriteAnnouncesAFork` asserts stderr names `"master@msi"`, pi, claude and `"master@msi+claude"`, and that stdout names the thread actually written. Re-run fresh: PASS.
  - **AC3** — `TestTheSameAgentRewritesItsOwnBlockInPlace` (own stamp and own journal) and `TestABlockWithNoKnownWriterIsReplacedAndStamped` (no journal line; journal naming no agent). Re-run fresh: PASS.
  - **AC4** — `WriteThread` delegates to `writeThread` with an empty stamp, so no-agent output is identical by construction, and `TestAWriteWithNoAgentIsUnchanged` pins it across four documents × three keys against `WriteThread`'s own output; `TestMemHandoffWriteWithoutAgentSaysWhatItAlwaysSaid` pins the command's words (empty stderr, exact stdout line). The only other change on the no-agent path, `replaceMemoryFile`, is a behavior-preserving extraction that additionally fixes a real defect (temp file 0600 narrowing MEMORY.md's mode — now `os.Chmod`ed to the original perms, with `defer os.Remove` cleanup).
  - **AC5** — `features.json` f5 grep re-run fresh: PASS. The SKILL.md diff shows `--agent <agent>` on the `handoff-write` line plus the fork explanation, and fixes the adjacent `dotf mem thread --agent claude` hardcoding, which was feeding this spec's journal heuristic wrong attributions.
- Negative/abuse cases checked: writer-name validation (`TestWriteThreadAsRejectsAWriterTheStampCannotCarry`: uppercase, spaces, `)`, newline — the stamp is read back from the heading, so one-word lowercase is enforced, and no injection path into the heading exists); fork-of-fork (`TestAnotherAgentsStampedBlockIsForkedNotReplaced` second write replaces the fork in place, no cascade); idempotent stamped rewrite (`again.Changed` false); legacy-block migration interaction (probed, see findings).
- `dotf mem thread` derives keys as `<branch>@<host>`; `threadHeadingKey` cuts at the first space and `isThreadHeading` matches the head exactly, so `master@msi` cannot match a `master@msi+claude` heading — the fork cannot be confused with the base key, nor a branch-suffixed heading misread.
- Issue #1690 verified OPEN via `gh issue view` (matches tasks.md's setup claim). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the contract files.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | tests | The combined path "legacy un-threaded block migrates **and** the base key holds another agent's block (fork)" has no named test, though both features exist and interact inside `writeThread` | I probed it live (throwaway test, deleted): doc with legacy block + `### thread: master@msi (writer: pi)`; result was correct — fork `master@msi+claude` kept `pi`, legacy block re-appended as `legacy-2026-09-05`, pi's block byte-intact. Correct today, but unguarded against regression | UNTESTED | tests (a named case in `cli/internal/mem/handoff_writer_test.go`) |
| Minor | THEORETICAL | skill | SKILL.md names the agent vocabulary as `claude, pi, agy, copilot, opencode, codex` but omits `antigravity` (and `gemini`), both in `journalAgents` and in the proposal's own risk section — an antigravity session following the skill is not told it may name itself | `harness/skills/handoff/SKILL.md` diff vs proposal.md "Risks" (known agents list) | UNTESTED | vault (skill source; regenerate `harness/skills/handoff/SKILL.md`) |
| Minor | SPECULATIVE | parsing | A hand-edited or foreign stamp with non-lowercase writer, e.g. `(writer: Claude)`, is read back as a distinct writer, so `claude` forks from what is semantically its own block. Cannot arise from the validated write path (`writerName` rejects it); requires out-of-band editing of MEMORY.md | Probe: `WriteThreadAs` on a doc stamped `(writer: Claude)` with self `claude` → fork `master@msi+claude` keeping "Claude". Cost is one extra thread, never a loss | UNTESTED | — (surface only; do not gate) |
| Minor | SPECULATIVE | parsing | `journalWriter`'s first-known-agent-word heuristic could misattribute when a project word before the agent equals an agent name (e.g. a project literally named `pi-*`). The proposal declares this accepted: cost is one extra fork thread, never an overwrite | proposal.md §Risks, bullet "Reading the writer from `Journal:` is a heuristic"; code `journalAgents` map | `TestAnUnstampedBlockIsAttributedByItsJournalLine` covers the accepted journal shapes | — (declared accepted risk in the contract; no action) |
| Question | THEORETICAL | verification | `verification.md`'s "Live dry run against the vault" could not be independently reproduced: no `knowledge` checkout / `10_projects/knowledge/memory/MEMORY.md` exists on this machine. The mechanism it demonstrates (fork keeps antigravity's block byte-identical; no-`--agent` clobbers as before) is covered by the named unit tests, so nothing hangs on the unreproduced step | `ls ~/Projects/knowledge/memory/MEMORY.md` → absent in this session | `TestAnotherAgentsStampedBlockIsForkedNotReplaced` covers the mechanism | — (note in `verification.md` if desired) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All five ACs verified fresh including negative paths (unknown writer, invalid agent names, fork idempotence); no observed defect across probes and mutations |
| Verification       | A | Reproducible commands per feature in `features.json` (all re-run exit 0 here), a mutation table I independently reproduced red-green, and a live dry-run record (not reproducible on this machine — see Question) |
| Scope              | B | Named-file diff matches the proposal exactly; the only out-of-spec delta in those files is the incidental `requires:`/`generated_sha` frontmatter refresh in SKILL.md from the vault record refresh, and the shared base's other-spec commits are explicitly fenced off in `verification.md` |
| Reliability        | A | Atomic temp-file write preserved and improved (mode retention, deferred cleanup), fork notice printed even when the fork content is unchanged (the one surprising outcome), no new error paths left unhandled |
| Maintainability    | A | `WriteThreadAs`/`threadWriter`/`journalWriter` are small single-purpose functions with comments explaining WHY; `WriteThread`'s signature and behavior unchanged, so no caller churn |
| Handoff-readiness  | B | Decisions during implementation are recorded with reasons in `verification.md`, promotions explicitly dispositioned (none), archive checklist pending the review this file provides |

### Verdict

PASS WITH GAPS — no Blockers, no REAL Majors; four Minor findings (two THEORETICAL, two SPECULATIVE), each with a disposition here; one un-reproducible verification step that no criterion depends on.

### Recommended next steps

Route by set: **the contract set is closed** — the items below are for the implementer to disposition in `verification.md` (applied / ticketed / declined with a reason) or to carry into a follow-up ticket; none blocks the archive.

- **tests** — add a named test for the legacy-migration-plus-fork combination in `cli/internal/mem/handoff_writer_test.go` (the probe in the findings table is the seed case). Disposition in `verification.md` or attach to the follow-up.
- **vault** — add `antigravity` (and `gemini`) to the handoff skill's agent vocabulary next time the record is refreshed, so every agent `journalAgents` knows is one the skill tells to name itself.
- **verification.md** — note that the live dry run was performed on the implementer's machine and is not reproducible where the `knowledge` checkout is absent; the named unit tests carry the mechanism.

**Archive**: `dotf spec archive` is advisable once the dispositions above are recorded in `verification.md`; no code change or re-review is required by this verdict.
