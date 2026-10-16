---
spec: "CLI-119-lane-next"
verdict: "PASS"
reviewed_sha: "ec982b38eb026c54e8154c1b1d2351557a7b21dd"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-15"
---

## Adversarial review

**Scope**: CLI-119-lane-next — `dotf lane exec` + `dotf lane next` (herdr backend, `.dotf/lane.json` queue, prompt template, lease), round 3.
**Sources**: `specs/CLI-119-lane-next/{proposal,tasks,verification}.md` + `features.json`; diff `cfffe79de47b31b326ae1222248f78414bd6e55a...ec982b38` — the whole change since the base the launcher resolved, not the delta since round 2. The lane commits in the window are `58909de7` (#2532), `f0c1b9e5` (#2555), `94fea6b6` (#2578), `ec982b38` (#2588, this round's fix) plus `dd1f5c78` (#2577, `dotf lane status`, which carries its own spec CLI-127-lane-reconcile); the rest of the 183-file window is other merged PRs (deploy manifest v6, worktree agent/adopt/migrate, CLI-118, harness bind, release 0.71.0) with their own specs, which I did **not** deep-review here — a stated scope limit, not a clean bill.

Everything below was run fresh at `ec982b38` on this Mac (dotf 0.71.0, herdr 0.9.3). Working tree left clean apart from the launcher's `review-request.json`.

**Gate re-ran green**: `go build ./...` exit 0 · `go vet ./...` exit 0 · `GOOS=windows go vet ./...` exit 0 · `go test ./... -count=1` → no failing package (lane/cmd/worktree `ok` ×3) · `golangci-lint` 2.12.2 on `./internal/lane/... ./internal/cmd/...` → `0 issues.` · all six `features.json` verification commands re-run exit 0 · normalized contract digests recomputed (`proposal.md` `7b14f9fc…`, `tasks.md` `d617197c…`, `features.json` `9efa3d59…`) match `review-request.json` exactly, so nothing in the contract set moved since launch · no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder.

**Round-2 REAL Major (spaced repo path, #2537) re-verified by reproduction**: scratch repo at `/private/tmp/lane-repro/a b/repo` (stubbed `gh`, stubbed `herdr` so the no-backend path is taken), built from this branch — `dotf lane next --dry-run --agent claude` → **exit 0**, the four commands print with `"/private/tmp/lane-repro/a b/repo"` and `--prompt-file "…/.git/dotf-lane/999.md"` double-quoted; non-dry-run → exit 0 with the prompt written to `.git/dotf-lane/999.md`. Round 2's before-state was exit 1 with the "prompt path has characters…" refusal. **Mutation check** (all three edited back afterwards, tree reverted clean): (a) make the typed line carry a chosen path → `TestNext_APathWithASpaceReachesNoPane` **FAIL**; (b) let POSIX `plainWord` admit a backslash → `TestPlainWord_BackslashIsAPathSeparatorOnlyOnWindows` **FAIL**; (c) stop quoting printed paths in `humanPath` → `TestNext_NoBackendQuotesAPathWithASpace` **FAIL**. Tickets: **#2537 CLOSED**, #2576 OPEN, #2587 OPEN (all three read live).

### Spec and task alignment

- **AC1** (argv as one element, unknown agent refused): named tests `TestLaunchArgv_PinsEachAgentAndPassesThePromptAsOneArgument`, `TestLaneExec_HandsTheAgentTheFilePromptAsOneArgument`; three `RESULT <agent> PASS` lines present; re-ran green.
- **AC2** (linked worktree → create through parent, caller untouched): `TestNext_FromALinkedWorktreeCreatesThroughTheParentAndLeavesTheCallerAlone` plus the new `TestNext_APathWithASpaceReachesNoPane` (now folded into f2's verification command); the Backend interface still has no remove method; re-ran green.
- **AC3** (run twice, one agent): `TestQueue_NextIsTheFirstOpenUnassignedIssueAndIsStable`, `TestNext_RunTwiceStartsOneAgent`; live second-run evidence in verification.md; re-ran green.
- **AC4** (default pane agent, refusal listed): `TestNext_AgentDefaultsToTheCallingPaneAndUnknownIsRefused`, `TestLane_RefusalsAreShownOnStderr`; re-ran green.
- **AC5** (no backend → commands, exit 0): `TestNext_NoBackendPrintsTheCommands`, `TestNext_NoBackendAndNoAgentPrintsAPlaceholder` — and my live spaced-path run above exits 0. The `$`-path edge case is finding 2 below.
- **AC6** (real chain per agent): prose evidence (`RESULT` lines, grep-checked by f6); a live chain is not re-runnable from here, as disclosed in verification.md in rounds 1–2.
- `tasks.md` boxes match reality (Closing all `[x]`, and the gate items — type check, lint, tests — I re-ran myself). Round-2's three disposition groups (fix + tests + lesson 449; ticketed race/serial-`gh`; process questions) are all written into verification.md's round-2 section, and the two open tickets are open, verified live.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | input handling | `readPrompt`'s new fallback (`git rev-parse --show-toplevel` when a relative `--prompt-file` is not in the pane's cwd) has no named test; a pane standing in a different checkout that happens to hold a stale `.dotf-lane-prompt.md` would hand the successor that stale prompt instead of failing | code read of `cmd/lane.go` `readPrompt`; the only `show-toplevel` line in tests is the git stub inside `TestLaneNext_ARepositoryUnderAPathWithASpaceHandsTheLaneOver`, which exercises `lane next`, not `lane exec` | UNTESTED | tests (one fallback case: relative file absent in cwd, present at repo root; optionally also the wrong-root case) |
| Minor | REAL | spec-vs-code / AC5 | With no backend, a repo path containing `$`, backtick, `!`, `%` or `"` still exits **1** (`Error: lane: "…a$b/repo… has a character no shell quotes alike; move the repository to a path without…"`) while AC5's letter says "prints the commands and exits 0". I reproduced this live in a `$`-path scratch repo. Ruling: the refusal is the correct behavior — no quoting of those characters means the same across bash/zsh/PowerShell/cmd, and a printed command no shell reads alike is not "the commands" — and it is encoded deliberately in `TestNext_NoBackendQuotesAPathWithASpace` (the refusal loop). Not gating; the ruling should be on record | live repro this session (`/tmp/lane-repro/a$b/repo`, stubbed `gh`/`herdr`, exit 1) | `TestNext_NoBackendQuotesAPathWithASpace` (asserts the refusal) | verification.md disposition (record the ruling); no contract edit — the proposal's wording stays as reviewed |
| Minor | SPECULATIVE | injection (shell-adjacent, printed) | `manualCommands` prints `o.Config.Base` and its remote half unquoted; `LoadConfig` only requires both sides of the first `/` to be non-empty, so `base: "origin/main; x"` or a remote name with a space yields a pasted line that runs/reads differently. Mitigations: the input is the repo's own committed `.dotf/lane.json` (same trust as any checked-out repo file), the typed backend line never carries it, and the branch is a `[a-z0-9-]` slug from `Slug` | code read of `queue.go` (`LoadConfig`, `Remote`, `Slug`) vs `handoff.go` `manualCommands`; not reproduced | UNTESTED — `TestLoadConfig_BaseMustNameARemoteAndABranch` covers empty halves only | code + tests (constrain base/remote charset to a plain word, or decline with a reason in verification.md) |
| Question | — | process | `features.json` still carries hand-authored `"state": "passing"` with `evidence` dated 2026-10-14, and f2's `verification` command was extended in `ec982b38` without refreshing that string; accepted with disclosure in rounds 1–2. I re-ran all six commands at this sha: exit 0. Also carried: the ~600-executable-line single PR exceeds the ~300-line cap (declared in verification.md), and `specs/CLI-118-worktree-one-root/` in this window still has no `review.md` — that spec's own archive gate | read of the three contract files (digests match launch); six commands re-run | n/a | process (carry forward; do not edit the contract set on a passing verdict) |

Round-2's remaining minors stay as dispositioned: `excludeLease`/`excludeFile` read-modify-write race → #2576 OPEN; serial `gh issue view` → #2587 OPEN. The Windows `lane exec` child path remains compile-verified only (`GOOS=windows go vet` clean), and the herdr backend path for spaced repos is unit-test verified (my live repro exercised the no-backend path); both were disclosed as not measured in rounds 1–2 and remain so.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All six ACs verified against code, named tests, three killing mutations and live spaced-path repro; the untested `readPrompt` fallback and the `$`-path AC5 edge keep it from A |
| Verification | B | Full gate re-ran green here (build/vet/windows-vet/lint/full suite/six feature commands), contract digests match launch, mutations killed; AC6's chain and the Windows exec path remain prose/compile-only as disclosed |
| Scope | B | Lane commits match the proposal; `lane status` rides in under its own spec CLI-127; the wide window is the launcher's base spanning other merged specs (stated limit), and the one-PR line count is declared |
| Reliability | B | Errors loud and tested, refusals leave the backend untouched, round-2's hard failure is fixed and regression-tested; ticketed #2576 start race / create-then-type stall and the deliberate `$`-refusal class keep it from A |
| Maintainability | B | Clean seams (`Runner`/`Backend`/`IssueSource`), functions ≤40 lines, lint 0 issues; residue is the untested fallback branch and base/remote string knowledge split between `LoadConfig` and `manualCommands` |
| Handoff-readiness | B | Round-2 dispositions written with before/after evidence, #2537 closed, #2576/#2587 open and named, lessons 438/449 in `docs/lessons/`, promotion lines answered; features evidence strings dated before f2's extension is the only staleness |

Aggregation: all **B**, no C or D → the rubric alone reaches PASS; the severity path finds no Blocker and no REAL Major, so both paths agree.

### Verdict

**PASS**

Round 2 failed on one REAL Major: the command this spec ships could not run for any repository under a path with a space. That is fixed the right way — the typed line now carries no chosen path at all (prompt written into the successor's worktree under a fixed relative name), the printed commands double-quote their paths and refuse the characters no quoting fixes, and the guard is per-OS. I reproduced round 2's exact failing scenario as exit 0, killed all three fix-relevant mutations with their named tests, re-ran the whole gate green, and confirmed the contract set has not moved since launch. The three open findings are minors (one untested fallback, one deliberate AC5 edge I am putting my ruling on record for, one speculative config-in-paste concern), none of which blocks archive under the severity × reality rule.

### Recommended next steps

(All outside the contract set — for disposition in `verification.md` or a follow-up ticket; do not edit `proposal.md`, `tasks.md` or `features.json` on this verdict.)

1. **Tests**: add one named case for `readPrompt`'s repo-root fallback (relative file absent in cwd, present at the worktree root) so finding 1 stops being UNTESTED. Scope: tests; can ride on #2587's follow-up.
2. **Verification.md**: record the finding-2 ruling (the `$`/`!`/`%` refusal is the correct reading of AC5 — a command no shell reads alike is not "the commands") and note that f2's extended verification command was re-run at `ec982b38` by this review. Scope: verification.md.
3. **Ticket or fix**: constrain `base`/remote to a plain word in `LoadConfig` (or decline with a reason), closing finding 3. Scope: code + tests; a follow-up issue beside #2587 is enough.
4. **Optional**: let the harness re-capture `features.json` evidence on the next `dotf spec review` so the f2 string dates the command that actually ran.

**Is `dotf spec archive CLI-119-lane-next` advisable now? Yes.** The verdict is PASS, the contract digests match the launcher's record, and no `[AGENT-DRAFT]` tags remain; items 1–4 are tracked gaps, not archive gates.
