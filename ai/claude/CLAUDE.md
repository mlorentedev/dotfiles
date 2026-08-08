# CLAUDE.md

> **First, read `AGENTS.md` at the repo root** — it is the canonical SSOT for behaviour rules across all agents (Standing Orders, Decision Hierarchy, Neural Hive, MCP usage, Operational Rules). This file (`~/.claude/CLAUDE.md` after deploy) contains only Claude Code-specific extensions on top of `AGENTS.md`.
>
> If `AGENTS.md` is missing from the current repo, default to the canonical version at `$DOTFILES_REPO_DIR/AGENTS.md` (resolved via `machine.json` per ADR-025; falls back to `~/Projects/Workspace/dotfiles/AGENTS.md`).

## Auto-Maintenance Rules

Self-maintaining knowledge across sessions. Zero manual intervention required.

### Session Handoff

Session handoff is a **cross-agent `/handoff` skill** — SSOT at `00_meta/skills/handoff/SKILL.md`, with an always-on trigger in `AGENTS.md`. At the END of any non-trivial session, run it. For Claude, the continuity block it replaces in place (`## Session Handoff`, the **last** section — after the stable index, for KV-cache prefix stability) lives in this project's auto-memory `MEMORY.md`. The skill is the source of truth for the full checklist (continuity block + vault hygiene + repo/worktree state + artifact summary + next action) — don't duplicate it here.

### Auto-Crystallize

If session start context includes `CRYSTALLIZE NEEDED`, run `/crystallize` BEFORE any user task. Inform briefly: "Auto-crystallizing (N days stale)."

### Auto-Archive Cold Memories

If session start context reports memory files needing archive (>60 days cold):

1. Create `memory/archive/` if needed.
2. Move flagged files there.
3. Remove their entries from `MEMORY.md`.
4. Inform briefly: "Archived N cold memory files."

## Claude Code Tooling Notes

* **Overrides of harness defaults (generated).** Sourced from the vault via `scripts/compile-harness.sh` — edit the vault pattern + re-run setup, not here:
<!-- BEGIN HARNESS GENERATED (sha256:9bbc453bc3f4cd17) — SSOT: vault 00_meta/patterns; edit there + re-run setup, do NOT edit between markers -->
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
2. **Knowledge** — what was learned is written where it belongs, this session: build/operate detail in the repo (`docs/lessons.md`, `docs/adr/`), cross-project insight in the store.
3. **Board** — the ticket matches reality: picked up when you start, blocked when blocked, closed with the change that closed it.
4. **Review** — an open PR is not finished work. Its checks and its reviewer comments are triaged, and each comment is applied, ticketed, or declined with a reason.
5. **Evidence** — no completion claim without the command output that proves it, produced in this session.

Any of the five may be skipped, but only as a stated decision naming which one and why. Silence is not a skip.
<!-- END HARNESS GENERATED -->
* **Skills.** `~/.claude/skills/<skill>/SKILL.md` auto-load via slash commands. Skill auto-loading is a Claude Code feature, not portable. Skill **content** is portable: the harness render path (`harness/skills/<name>/` with `targets[]`, deployed offline by `compile-harness.sh --deploy` — ADR-021) emits each skill as an OpenCode command at `~/.config/opencode/commands/<name>.md`. (AI-012 shipped the original transform in PR #43; the standalone `skills-to-opencode.sh` was since retired.)
* **TaskCreate / TaskUpdate / TaskList.** Use for non-trivial multi-step work (≥3 distinct steps). Mark `in_progress` BEFORE starting; mark `completed` immediately on finish. Don't batch updates.
* **AskUserQuestion.** Use for branching decisions with 2-4 mutually exclusive options. Always include "(Recommended)" on the preferred option.
* **MEMORY.md.** Auto-loads at session start (capped at 200 lines). Index-only — never write memory content directly here; use linked memory files.

## Project Memory Hierarchy

Claude reads memory in this order at session start:

1. **Global:** `~/.claude/CLAUDE.md` (this file — deployed from `ai/claude/CLAUDE.md` in dotfiles).
2. **Project root:** `<repo>/AGENTS.md` (canonical SSOT — read FIRST per the pointer at the top).
3. **Project-specific:** `<repo>/.claude/CLAUDE.md` (optional, repo-specific overrides).
4. **Auto memory:** `~/.claude/projects/<project-hash>/memory/MEMORY.md` (cross-session continuity).

If both `CLAUDE.md` and `AGENTS.md` exist in a repo, `AGENTS.md` is authoritative for behavioural rules; `CLAUDE.md` overlays Claude-specific tooling notes on top.

## Model Tier (per AGENTS.md "Model Selection")

- **Top:** `claude-opus-4-7` — hard debug / architecture / root-cause / Socratic Guardrail triggers
- **Mid:** `claude-sonnet-4-6` — mechanical refactor / docs / single-file fixes / test scaffolding
- **Low:** `claude-haiku-4-5-20251001` — syntax lookups / quick questions

Subagent declaration: `model: opus|sonnet|haiku` in frontmatter. Main session: `/model` slash command.
