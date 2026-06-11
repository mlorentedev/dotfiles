# CLAUDE.md

> **First, read `AGENTS.md` at the repo root** — it is the canonical SSOT for behaviour rules across all agents (Standing Orders, Decision Hierarchy, Neural Hive, MCP usage, Operational Rules). This file (`~/.claude/CLAUDE.md` after deploy) contains only Claude Code-specific extensions on top of `AGENTS.md`.
>
> If `AGENTS.md` is missing from the current repo, default to the canonical version at `~/Projects/dotfiles/AGENTS.md` (Linux/macOS) or `%USERPROFILE%\Projects\dotfiles\AGENTS.md` (Windows).

## Auto-Maintenance Rules

Self-maintaining knowledge across sessions. Zero manual intervention required.

### Session Handoff

Session handoff is a **cross-agent `/handoff` skill** — SSOT at `00_meta/skills/handoff/SKILL.md`, with an always-on trigger in `AGENTS.md`. At the END of any non-trivial session, run it. For Claude, the continuity block it overwrites (`## Session Handoff`, the first section after the H1) lives in this project's auto-memory `MEMORY.md`. The skill is the source of truth for the full checklist (continuity block + vault hygiene + repo/worktree state + artifact summary + next action) — don't duplicate it here.

### Auto-Crystallize

If session start context includes `CRYSTALLIZE NEEDED`, run `/crystallize` BEFORE any user task. Inform briefly: "Auto-crystallizing (N days stale)."

### Auto-Archive Cold Memories

If session start context reports memory files needing archive (>60 days cold):

1. Create `memory/archive/` if needed.
2. Move flagged files there.
3. Remove their entries from `MEMORY.md`.
4. Inform briefly: "Archived N cold memory files."

## Claude-Only MCP: claude-mem (Conversation Memory)

**Active by default in every session.** Captures observations automatically via session hooks — conversation flow → claude-mem, crystallized knowledge → vault. Never duplicate across both.

* `/mem-search "query"` — find solutions from past sessions.
* `/timeline-report`, `/knowledge-agent`, `/how-it-works` — narrative history, topic brains, self-explanation.
* **Do NOT** write strategic decisions, lessons, or ADRs to claude-mem — those go to vault via `capture_lesson` / `vault_write`.
* Default `worker` runtime blocks manual writes (`observation_add`, `memory_add`); hook capture works regardless. Set `CLAUDE_MEM_RUNTIME=server-beta` in `~/.claude/settings.json` to enable manual writes.

For the full dual-memory protocol, query `00_meta/patterns/pattern-dual-memory.md`.

## Claude Code Tooling Notes

* **Overrides of harness defaults (generated).** Sourced from the vault via `scripts/compile-harness.sh` — edit the vault pattern + re-run setup, not here:
<!-- BEGIN HARNESS GENERATED (sha256:e9c8d9d67d9ce58f) — SSOT: vault 00_meta/patterns; edit there + re-run setup, do NOT edit between markers -->
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
<!-- END HARNESS GENERATED -->
* **Skills.** `~/.claude/skills/<skill>/SKILL.md` auto-load via slash commands. Skill auto-loading is a Claude Code feature, not portable. Skill **content** is portable: `AI-012-opencode-commands-port` mechanically transforms each skill into an OpenCode command in `ai/opencode/commands/*.md`.
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
