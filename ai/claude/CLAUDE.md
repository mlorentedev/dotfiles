# CLAUDE.md

> **First, read `AGENTS.md` at the repo root** — canonical SSOT for all agents (Standing Orders, Decision Hierarchy, Neural Hive, MCP, Operational Rules). This file (`~/.claude/CLAUDE.md` after deploy) holds only Claude Code-specific tooling extensions.
>
> If `AGENTS.md` is missing from the current repo, default to canonical version at `$DOTFILES_REPO_DIR/AGENTS.md` (resolved via `machine.json` per ADR-025).

## Auto-Maintenance Rules

- **Session Handoff:** Run `/handoff` (`00_meta/skills/handoff/SKILL.md`) at the end of non-trivial sessions. Replaces `## Session Handoff` continuity block in `MEMORY.md`.
- **Auto-Crystallize:** If context includes `CRYSTALLIZE NEEDED`, run `/crystallize` before user tasks.
- **Auto-Archive:** If context reports >60d cold memory files, move to `memory/archive/` and update `MEMORY.md`.

## Claude Code Tooling Notes

* **Overrides of harness defaults (generated).** Sourced from the vault via `scripts/compile-harness.sh` — edit the vault pattern + re-run setup, not here:
<!-- BEGIN HARNESS GENERATED (sha256:ea171c3de1a715ff) — SSOT: vault 00_meta/patterns; edit there + re-run setup, do NOT edit between markers -->
- **No AI attribution** in git history or GitHub messages (commits, PRs, issues).
- No `Co-Authored-By` trailers referencing AI agents.
- No bot-style emojis or "Generated with" footers.
- All artifacts must appear human-authored.
- **English only** in git/GitHub artifacts: commit messages, branch names, PR/issue titles and bodies, and code comments. Conversation with the user may be in any language; the durable record is English.
- **No internal phase/milestone references** in branch names, commit messages, or PR titles.
  - Bad: `feat/phase-3.1-scaffold`, `chore: scaffold repo (Phase 3.1)`
  - Good: `feat/scaffold-pyhydra3d`, `chore: scaffold PyHydra3D repository`
- Phase/milestone tracking belongs in the bitácora GitHub Project (issues + board), not in git history or the vault (per ADR-018).
- **Auto-merge is forbidden in every repository.** Never run `gh pr merge --auto`, never enable "Auto-merge" in the GitHub UI, and keep the repo setting `allow_auto_merge=false`. Auto-merge lands a PR the instant CI goes green — bypassing the human review gate in §1.
- Every PR merges deliberately, after a human has reviewed it and CI is green (squash or rebase per §4, diff verified per §5). Merge is a supervised action, never a queued automatic one. An agent merges only when the user has authorized merging that specific PR.

> Injected verbatim into every agent's instructions (harness `enforced` id `definition-of-done`) and executed by the `verification-before-completion` skill. It **binds** existing standing orders to the moment of closing; it does not restate them.

Working code is not a finished change. Before saying done, each of these is true:

1. **Debt** — every defect noticed along the way is fixed in scope or filed as a ticket with its root cause. A mention in conversation is not an exit.
2. **Knowledge** — what was learned is written where it belongs, this session: build/operate detail in the repo (docs/lessons/, docs/adr/), cross-project insight in the store.
3. **Board** — the ticket matches reality: picked up when you start, blocked when blocked, closed with the change that closed it.
4. **Review** — an open PR is not finished work. Its checks and its reviewer comments are triaged, and each comment is applied, ticketed, or declined with a reason.
5. **Evidence** — no completion claim without the command output that proves it, produced in this session.

Any of the five may be skipped, but only as a stated decision naming which one and why. Silence is not a skip.

> Injected verbatim into every agent's instructions (harness `enforced` id `pr-stewardship`). It elaborates Definition of Done §4 — "an open PR is not finished work" — into what that item leaves implicit: what you still owe a PR after you push it, and what does not count as having been reviewed.

**What binds is the disposition, not the waiting.** Before the change is called done, the PR's checks and its reviewer output are dispositioned — each one applied, ticketed, or declined with a reason. *How* you learn they arrived is not prescribed: a project that already tells you when to look back — the human notifies, a hook fires — has met this, and its instruction wins. Absent such a signal the default mechanism is to stay: the window closes at the first of an actionable reviewer comment or ten minutes after the checks settle, and pushing a fix reopens it, because the reviewer re-reviews. Leaving with nothing dispositioned hands the next session a change nobody read.

**"Hand the PR over; don't watch CI" is this rule's escape being exercised, not a contradiction of it.** Where a project names the signal, its instruction wins — and that rule names one: *the human reviews the PR and reports a red build*. So in a repository carrying it the timed window never opens, and what that rule forbids is the watch loop, never the disposition. Read in that order the two are one instruction: don't sit and watch, and don't leave the reviewer's output unread.

**A comment is not a review, and green checks are not the end of one.** Both halves have been observed failing here. On one PR every check went green and the reviewer then posted four Major findings. On another, checks went green and the reviewer posted *"review limit reached — we couldn't start this review"*: a comment arrived, and nobody looked. **A notice that no review ran leaves the PR unreviewed.** Tell the two apart by content, never by author — a review names files, lines, or claims; a notice talks about the review itself. Proceeding on an unreviewed PR is allowed; proceeding silently is not. "Merged unreviewed, reviewer quota exhausted" is a disclosure; saying nothing is a claim of review that never happened.

**A change that closes a spec gets an independent adversarial review before it archives.** The trigger is the archive gate and nothing wider — not every PR that touches a spec folder. It names an obligation that already binds mechanically, so the only question is whether you meet it deliberately or discover it as a refusal: the spec gate declines to merge a PR closing a spec's issue without archiving it, `spec archive` declines without a passing review, and the reviewer pool declines one signed by the wrong model. The reviewer must not be the implementer; that independence is the entire value.
<!-- END HARNESS GENERATED -->
* **Skills:** Auto-loaded via slash commands from `~/.claude/skills/<skill>/SKILL.md` (deployed by `compile-harness.sh --deploy`).
* **TaskCreate / TaskUpdate / TaskList:** Use for non-trivial work (≥3 steps). Mark `in_progress` before start, `completed` on finish.
* **AskUserQuestion:** Use for branching decisions (2-4 options), marking "(Recommended)" on preferred.
* **MEMORY.md:** Auto-loads at session start (capped at 200 lines). Index-only; link to topic files.

## Project Memory Hierarchy & Models

1. **Resolution Order:** `~/.claude/CLAUDE.md` $\rightarrow$ `<repo>/AGENTS.md` (authoritative SSOT) $\rightarrow$ `<repo>/.claude/CLAUDE.md` $\rightarrow$ `<project-hash>/memory/MEMORY.md`.
2. **Model Tier:** Top: `claude-opus-4-7` (architecture/debug) | Mid: `claude-sonnet-4-6` (default/refactor) | Low: `claude-haiku-4-5-20251001` (lookups).

