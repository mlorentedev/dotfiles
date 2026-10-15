---
id: "CLI-119-lane-next"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-10-14"
issue: "mlorentedev/dotfiles#2528"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
wip_override: "owner 2026-10-15: the lane successor is the fix for the stall that wip limits cause; take it as an exception (10 active, limit 10, 2026-10-14)"
---

# CLI-119-lane-next

> **Naming**: file lives at `<repo>/specs/CLI-119-lane-next/proposal.md`. `CLI-119-lane-next` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #2528: CLI-119: a fix lane has no successor: landing a PR ends the session and the next queued bug is opened by hand -->

A fix lane runs one bug per branch, worktree and PR, but nothing hands the lane to a successor: removing the finished worktree closes the pane and the session (#1653), so the rule became "land, hand off, stop", and the owner opens the next bug from herdr's sidebar and re-pastes a prompt. The queue lives only in prose. Measured 2026-10-15: after bug 1 of 4 the lane stopped; a hand-made `herdr worktree create` plus `herdr pane run wG:p1 'claude "$(cat prompt.md)"'` started the successor correctly, so the mechanism works and only the command is missing.

## What

Two commands under `dotf lane`, one implementation for every OS and every project.

- `dotf lane exec --agent <name> --prompt-file <path>`: replaces itself with the named agent's interactive CLI, the prompt passed as an argument by dotf, never through a shell. This is what runs inside the new pane, so no quoting rules of zsh, bash or PowerShell apply. Launch argv per agent comes from one table in Go: `claude <prompt>`, `pi <prompt>`, `agy -i <prompt>`. An agent not in the table is refused, never guessed; an agent is added only once a real launch of it has been observed.
- `dotf lane next [--agent <name>] [--dry-run]`: from any checkout or linked worktree of the repo, picks the next item of the repo's queue, creates its worktree off origin/main in the backend (herdr) from the repo's PARENT checkout (herdr refuses a linked worktree as source), renders the successor prompt from the repo's template, and runs `dotf lane exec` in the new pane. It then returns; it never enters the new worktree and never removes the caller's. `--agent` defaults to the agent of the calling pane (`herdr pane current`), so the successor is the same kind as the finisher.
- The repo declares the lane in `.dotf/lane.json`: `queue` (ordered issue numbers), `branch_prefix`, `base`, `template` (a path). Next item = the first queued issue that is OPEN and has no assignee; running it twice, or from two sessions, picks the same item or none, and an existing worktree for the branch is reused, never given a second agent. Nothing in the file is rotated, so there is no commit to make between bugs.
- No backend (herdr absent): `dotf lane next` prints the exact commands and exits 0 with a notice. Backends sit behind an interface so tmux can follow.

## Out of scope

- Agents other than claude, pi and agy (copilot, opencode, codex): the table takes an entry only after a real launch is observed; each is a one-line follow-up.
- An ordered GitHub label as the queue (the issue's option 2): a label has no order; `.dotf/lane.json` is the v1 queue and a board field can replace it later.
- Carrying session-local settings (`/model`, permission mode) into the successor: only flags written in the launch table are passed.
- Fixing #1653 (`dotf worktree done` removing the launch worktree) and CLI-118 (one worktree root): this command avoids the first by never removing anything.

## Risks / open questions

- herdr's `pane run` types the command into the pane's shell, so `dotf` must be on that shell's PATH and the command must be a single plain word list; the pane's first prompt may not be ready (needs a `wait-output` on the prompt, to measure).
- The agent CLIs change their flags: the table is the one place to fix, and a test pins the argv of each entry.
- Windows: `lane exec` cannot replace its process, so it runs the agent as a child with stdio attached; to be measured on the Windows box, not assumed.

## Acceptance criteria

- [ ] `dotf lane exec` launches claude, pi and agy with the prompt as one argv element, shown by an argv-recording fake in tests and by a real launch of each, recorded in verification.md.
- [ ] `dotf lane next` from a LINKED worktree creates the successor's worktree through the repo parent, runs `lane exec` in its pane and leaves the caller's worktree and branch untouched.
- [ ] The same `lane next` run twice picks the same issue and starts one agent.
- [ ] `--agent` defaults to the calling pane's agent; an unknown agent is refused with the list of known ones.
- [ ] With no backend it prints the commands and exits 0.
- [ ] An end-to-end run on this Mac chains finisher -> successor for each of claude, pi and agy.

## References

- Bitácora board: dotfiles#2528
- Related: #1653 (done removes the launch worktree), #2503 / CLI-118 (one worktree root)
- ADR-020 (new tooling in Go), ADR-043 (WIP limit; exception taken by the owner 2026-10-15)
