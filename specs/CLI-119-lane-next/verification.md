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

- RESULT claude PASS: default agent taken from the calling pane (`herdr pane current` -> claude); the successor started with no trust prompt and answered `LANE-READY`.
- RESULT pi PASS: `--agent pi`; started through `dotf secrets run`, answered `LANE-READY`.
- RESULT agy BLOCKED: `--agent agy`; the pane ran agy but stopped at its trust dialog for the new worktree, so prompt delivery through the chain was not observed. Tracked in #2533 (the `worktrees/*/*` entry in `trustedWorkspaces` does not suppress it). A direct `lane exec --agent agy` in an already-trusted directory delivered the prompt and answered (Measurements above), so only the trust gate is open.
- Second `lane next` for the same item: "a workspace already shows <branch>; no second agent started", nothing created.
- A first attempt typed `dotf lane exec` and the pane answered `unknown command "lane"` (the pinned release has none): the cause of typing this binary's absolute path (lesson 438).
- Claude trust: a worktree of a repo already trusted needs no registration. The same chain in an untrusted scratch repo stopped at Claude's trust dialog, which is the gate working, so no trust step was added.

## Evidence

- [x] AC1 -> `TestLaunchArgv_PinsEachAgentAndPassesThePromptAsOneArgument`, `TestLaneExec_HandsTheAgentTheFilePromptAsOneArgument`, real launches above
- [x] AC2 -> `TestNext_FromALinkedWorktreeCreatesThroughTheParentAndLeavesTheCallerAlone` (the interface has no remove method)
- [x] AC3 -> `TestQueue_NextIsTheFirstOpenUnassignedIssueAndIsStable`, `TestNext_RunTwiceStartsOneAgent`, live second run above
- [x] AC4 -> `TestNext_AgentDefaultsToTheCallingPaneAndUnknownIsRefused`
- [x] AC5 -> `TestNext_NoBackendPrintsTheCommands`, `TestNext_NoBackendDryRunWritesNoPrompt`
- [ ] AC6 -> claude and pi PASS; agy BLOCKED on #2533

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
