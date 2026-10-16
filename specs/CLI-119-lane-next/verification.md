---
tags: [spec, verification, templates]
created: "2026-10-14"
---

# Verification - CLI-119-lane-next

## Measurements

2026-10-14, macOS, herdr 0.9.3: `dotf lane exec --agent <a> --prompt-file p.md` typed into a fresh split pane by `herdr pane run`, prompt "Reply with exactly the word LANE-READY and do nothing else."

- claude (Sonnet 5.5): started interactive, answered `LANE-READY`, pane `agent: claude`.
- agy 1.3.5 (`agy -i`): started interactive, answered `LANE-READY`, pane `agent: agy`.
- pi: answered `LANE-READY` only after the launch carried its keys. The shell's `pi` is a function that wraps `dotf secrets run`, and exec skips functions, so the table launches `dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi` (same key list as the bash and PowerShell wrappers).

These prove the launch half only. The finisher -> successor chain (AC6, `RESULT <agent> PASS` lines) lands with `lane next`.

## End-to-end runs

2026-10-14, macOS, herdr 0.9.3, `dotf lane next` built from this branch, run from a worktree of the trusted dotfiles repo with a harmless prompt ("Reply with exactly the word LANE-READY"). Each run created a real worktree through the repo parent, typed `<this binary> lane exec --agent <a> --prompt-file <f>` into the new pane, and started the agent. The test worktrees and branches were removed afterwards.

RESULT claude PASS: default agent taken from the calling pane (`herdr pane current` -> claude); the successor started with no trust prompt and answered `LANE-READY`.
RESULT pi PASS: `--agent pi`; started through `dotf secrets run`, answered `LANE-READY`.
RESULT agy PASS: `--agent agy`; with the worktrees root trusted (#2536; the first attempt stopped at agy's trust dialog because it does not read `worktrees/*/*`, #2533) the successor started and answered `LANE-READY`.
- Second `lane next` for the same item: "a workspace already shows <branch>; no second agent started", nothing created.
- A first attempt typed `dotf lane exec` and the pane answered `unknown command "lane"` (the pinned release has none): the cause of typing this binary's absolute path (lesson 438).
- Claude trust: a worktree of a repo already trusted needs no registration. The same chain in an untrusted scratch repo stopped at Claude's trust dialog, which is the gate working, so no trust step was added.

## Evidence

- [x] AC1 -> `TestLaunchArgv_PinsEachAgentAndPassesThePromptAsOneArgument`, `TestLaneExec_HandsTheAgentTheFilePromptAsOneArgument`, real launches above
- [x] AC2 -> `TestNext_FromALinkedWorktreeCreatesThroughTheParentAndLeavesTheCallerAlone` (the interface has no remove method)
- [x] AC3 -> `TestQueue_NextIsTheFirstOpenUnassignedIssueAndIsStable`, `TestNext_RunTwiceStartsOneAgent`, live second run above
- [x] AC4 -> `TestNext_AgentDefaultsToTheCallingPaneAndUnknownIsRefused`, `TestLane_RefusalsAreShownOnStderr` (the refusal text reaches stderr; round 1 found it swallowed)
- [x] AC5 -> `TestNext_NoBackendPrintsTheCommands` (path pinned to `worktree.ResolveWorktreePath` under an off-default root), `TestNext_NoBackendAndNoAgentPrintsAPlaceholder`, `TestNext_NoBackendWithoutAWorktreeRootRefuses`, `TestNext_NoBackendDryRunWritesNoPrompt`
- [x] AC6 -> claude, pi and agy PASS (agy needs #2536 on a machine that lacks the root in its trust list)

## Review round 1 (FAIL at ff0b1612): dispositions

Review: `review.md` (merged via #2573). Fixes in the PR that carries this section.

- Blocker, errors swallowed: applied. `SilenceErrors` removed from `lane exec` and `lane next`; `TestLane_RefusalsAreShownOnStderr` asserts the text on stderr. Mutation: restoring `SilenceErrors` fails it. Live: `dotf lane exec --agent copilot ...` prints `Error: lane: no launch command for agent "copilot" (known: agy, claude, pi)`, exit 1.
- Major, AC5 with no backend and no `--agent`: applied in code, contract text unchanged. The commands print with `--agent <agy|claude|pi>` and a note to fill it; no harness is privileged by a default. If the re-review rules a placeholder is not "the commands", AC5 is reworded to require `--agent`.
- Major, printed worktree path: applied. `manualCommands` builds the path from `WORKTREE_ROOT` and `worktree.ResolveWorktreePath`, and refuses when the root does not resolve instead of printing a path in another tree.
- Major, `dotf` without `lane` on a pane's PATH: resolved by the release, not by code. 0.71.0 (#2520) merged 2026-10-16T00:52Z and `dotf --version` reports 0.71.0 with `dotf lane --help` answering. The first lane run on a release binary is the AC6 re-run of the next review.
- Major, silent bare-`dotf` substitution: applied. An untypeable binary path puts a note in the outcome (`TestNext_ASubstitutedBinaryIsSaidSo`); a failed `os.Executable()` is reported on stderr.
- Minor, `base` without a slash: applied. `LoadConfig` rejects it (`TestLoadConfig_BaseMustNameARemoteAndABranch`).
- Minor, `Handoff` and `runLaneNext` over 40 lines: applied. Both measured at 39 (were 72 and 51).
- Minors, Find->Create race, create-then-type stall, slug collisions: ticketed as #2576 with root cause and fix options. The race may be subsumed by the atomic claim of CLI-128 (#2551).
- Minor, stale boxes: applied. `proposal.md` ACs and `tasks.md` Closing ticked after the gate re-ran green.
- Question (a), who set `passing` in `features.json`: the author wrote `state` and `evidence` by hand although `tasks.md` reserves it to the harness. Each of the six commands was re-run by the reviewer and passes; the states are left as they stood and the harness re-captures them on the next `dotf spec review`. Not repeated in this round: only two `verification` commands were extended (f4, f5), no state edited.
- Question (b), CLI-118 without `review.md`: that spec's own archive gate; not acted on here.
- Not re-measured live: with the queue now all closed or held, `lane next --dry-run` prints "nothing to hand off", so the no-backend commands are covered by the unit tests above and not by a live run.

## Review round 2 (FAIL at 94fea6b6): dispositions

Review: `review.md` (merged via #2586). Fixes in the PR that carries this section.

- Major, `lane next` fails when the repository path has a space (#2537): applied. The prompt is written into the new worktree as `.dotf-lane-prompt.md` (excluded from `git status`) and the typed line names it relative to the pane's directory, so no chosen path reaches the pane's shell. The printed no-backend commands double-quote their paths. Tests: `TestNext_APathWithASpaceReachesNoPane`, `TestNext_NoBackendQuotesAPathWithASpace`, `TestLaneNext_ARepositoryUnderAPathWithASpaceHandsTheLaneOver`. Lesson 449.
  - Before (release 0.71.0, scratch repo `$TMPDIR/a b/repo`, stubbed `gh`, no herdr): `dotf lane next --dry-run --agent claude` -> exit 1.
  - After (this branch, same repo): exit 0, the four commands print with `"…/a b/repo"` quoted and `--prompt-file "…/a b/repo/.git/dotf-lane/999.md"`.
  - The backend path (herdr) is covered by the unit tests only: the successor's pane starts in the worktree (herdr reports a pane's `cwd`), and `lane exec` falls back to the worktree root for a relative file it does not find.
- Minor, `plainWord` admits a backslash on POSIX (#2587 item 1): applied. The allowed set is per OS (`TestPlainWord_BackslashIsAPathSeparatorOnlyOnWindows`); the guard now covers the binary's path only.
- Minor, `excludeLease` read-modify-write race: ticketed in #2576, open.
- Minor, serial `gh issue view` per queued item (#2587 item 2): ticketed, open (#2587 stays open for it).
- Question, process (hand-authored `passing`, CLI-118 without a `review.md`): accepted by the reviewer; unchanged.
- Not measured: the Windows `lane exec` child-process path (compiles under `GOOS=windows`; no Windows box in this run).

## Test status

- `go build ./... && go vet ./... && go test ./internal/lane ./internal/cmd` clean; `GOOS=windows go vet` and golangci-lint 2.12.2 clean.
- Mutations on `Handoff` (open-worktree check, quoting guard, parent checkout, dry-run write): all killed; two survivors (dry-run write without a backend, agent validation) closed by a test and by removing the redundant check.
- Not measured: the Windows `lane exec` child-process path (compiles under `GOOS=windows`); the pane-shell-ready wait (input typed right after create was read by zsh in every run, so no wait was added).

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- The successor is told to run `dotf lane next` itself on landing; the queue is never rotated, so no commit sits between two bugs.
- The claim a session leaves is the issue's assignee, so `lane next` does not assign: assigning would make a second run advance the queue instead of starting nothing.
- One PR instead of the planned two: ~600 executable lines, one mechanism, the second half useless without the first.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-438-a-command-typed-into-a-pane-runs-the-panes-world-not-the-callers.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a new command inside ADR-020's CLI layer, no new invariant
- [x] New pattern candidate for `00_meta/patterns/`? no: no recurrence in a second project yet

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-119-lane-next/` -> `specs/archive/CLI-119-lane-next/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
