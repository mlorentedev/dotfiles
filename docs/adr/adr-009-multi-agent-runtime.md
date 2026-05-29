---
id: adr-009-multi-agent-runtime
type: adr
status: proposed
created: "2026-05-15"
---

# ADR-009: AGENTS.md as Single Source of Truth for Cross-Agent System Prompt

## Status: Proposed

## Date: 2026-05-15

## Context

The dotfiles repo originally shipped system-prompt-style instructions for four coding agents:

- `ai/claude/CLAUDE.md` (~350 lines): canonical, full Standing Orders, decision hierarchy, MCP usage rules.
- `ai/gemini/GEMINI.md` (~280 lines): manual port of CLAUDE.md, with intentional divergences (tool subset, ASCII-only constraints).
- `.github/copilot-instructions.md` (~44 lines): condensed, Copilot-flavoured restatement.
- `ai/aider/aider.conf.yml` (~30 lines): config-only — no behavioural rules, just provider/model selection. **Sunset in AI-011 PR2 (2026-05-16); replaced by OpenCode.**

As of AI-011, the active set is three agent files (Claude, Gemini, Copilot) plus OpenCode reading `AGENTS.md` natively.

These four files have drifted three times in the last 6 months (Engineering Discipline rules added 2026-03-25, claude-mem MCP rules added 2026-05-08, hive-first vault access added 2026-03-12). Every drift created an active incident: the agent without the rule produced output that violated it. Manually keeping four files in lockstep is the failure mode the SSOT principle (Standing Order 2) exists to prevent.

In parallel, the `agents.md` standard (https://agents.md) is being adopted by sst/opencode, Cursor, Codex, and others. OpenCode (the runtime this ADR onboards) reads `AGENTS.md` at repo root as its system prompt by default. The standard is intentionally schema-free Markdown — any content works, but agents converge on it as a *known location*.

This ADR addresses both signals: drift between four agent-specific files, and the emergence of a portable system-prompt convention.

## Decision

Adopt `AGENTS.md` at the repo root as the **single source of truth** for the cross-agent system prompt. Migrate the substantive content (Standing Orders, decision hierarchy, Operational Rules, MCP usage rules) into `AGENTS.md`. Reduce the four per-agent files to thin pointers (≤ 50 lines each) that contain only:

1. A header explaining what the file is and which agent reads it.
2. A short, agent-specific note about any behavioural quirks (e.g., Gemini's ASCII-only string constraints, Copilot's limited tool-use vocabulary).
3. A delegation line: "Behavioural rules and standing orders live in [`AGENTS.md`](../../AGENTS.md). Read that file first."

Skills (`ai/skills/*/SKILL.md`) remain Claude-only at the runtime level — the skill auto-loading mechanism is a Claude Code feature, not portable. However, skill **content** is portable: a companion spec (`AI-012-opencode-commands-port`) defines a script that mechanically transforms each skill into an equivalent OpenCode command in `ai/opencode/commands/*.md`. The transformation runs in CI as a `--check` gate so the two locations cannot drift.

`init-repo-agents.sh/.ps1` (already implemented) is the canonical generator: it pulls the SDD snippet from the vault and writes `AGENTS.md`. Future content additions to `AGENTS.md` (beyond the SDD snippet) live in the vault under `00_meta/templates/agents-*-section.md` and are composed by the same generator.

## Alternatives Considered

1. **Keep four parallel files and accept drift.** Rejected: drift incidents are increasing, not decreasing. Standing Order 2 (SSOT) explicitly forbids this pattern.
2. **Make `CLAUDE.md` the SSOT and have other agents read it via symlinks or include directives.** Rejected: Gemini/Copilot/OpenCode each look for their own conventional filename; none of them follow symlinks reliably across OSes, and `include` directives are not part of the markdown contract any of them honour.
3. **Generate the per-agent files from a master template.** Considered. Mechanically equivalent to this decision but more complex (build step before agents can read instructions). Deferred: if `AGENTS.md` adoption stalls and we end up needing per-agent variants again, this is the obvious next step.

## Consequences

### Positive

- One file to update when a standing order changes (no drift surface).
- OpenCode, Cursor, Codex, and any future `agents.md`-aware tool gets the full instruction set for free.
- Lower cognitive load for contributors: one canonical place to read the rules.
- Skills-as-OpenCode-commands (AI-012) becomes the first proof that the broader workflow set is agent-agnostic — a concrete portability test.

### Negative

- Claude Code does not auto-load `AGENTS.md` (it loads `CLAUDE.md`). The Claude entry point file therefore still exists as a pointer, but its body is intentionally thin. Mitigation: the pointer line ("Read AGENTS.md first") relies on Claude actually following it. Empirical observation in 2026-05 onboarding sessions confirms it does, but this is a soft contract — if Claude regresses, the per-agent file can be re-populated as a fallback.
- Copilot Chat's support for `AGENTS.md` is not officially documented (as of 2026-05-15). The plan is to validate empirically in spec `AI-013-copilot-instructions-refresh`. If Copilot ignores `AGENTS.md`, the pointer file remains the contract surface and accepts that limitation — not a regression vs. today.
- One additional file at repo root. Acceptable: it is the convention, not noise.

### Neutral

- ~~`aider.conf.yml` is unaffected — it is config, not instructions.~~ **Aider sunset in AI-011 PR2 (2026-05-16); replaced by OpenCode + Go subscription.**
- The vault Neural Hive protocol does not change. `AGENTS.md` references the vault the same way `CLAUDE.md` did before this ADR.

## Implementation Plan

Tracked via specs in `<repo>/specs/`:

1. `AI-011-opencode-bootstrap` runs `init-repo-agents.ps1` for the first time, producing the initial `AGENTS.md` (currently a stub with the SDD snippet only).
2. Migrate Standing Orders, decision hierarchy, and MCP usage rules from `CLAUDE.md` into `AGENTS.md` (handled inside `AI-013-copilot-instructions-refresh`).
3. Shrink `CLAUDE.md`, `GEMINI.md`, `copilot-instructions.md` to pointer-style files in the same PR.
4. Smoke-test each agent loads `AGENTS.md` (Claude reads it via the CLAUDE.md pointer, OpenCode reads it natively, Copilot validated empirically, Gemini via GEMINI.md pointer).
5. Promote ADR status from Proposed → Accepted once all four agents are confirmed to honour the new layout. Until then, this ADR is Proposed and the implementation should not merge to `main`.

## References

<!-- Provenance: project roadmap (theme 4 "Multi-Agent Runtime") and sprint backlog live in the maintainer's cross-project knowledge store. Not linked here to preserve repo->store independence (knowledge-placement directionality invariant). -->
- [ADR-001](adr-001-skill-based-ai-workflow.md) — the original "skills as a Claude convention" decision being partially superseded here
- [ADR-008](adr-008-skills-ecosystem-overhaul.md) — current 17-skill catalogue (the input to AI-012's port)
- External: https://agents.md (standard), https://opencode.ai (runtime adopting it)
