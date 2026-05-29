---
id: "dotfiles-adr-001-skill-based-ai-workflow"
type: adr
adr: "001"
title: Custom Skills Over BMAD Framework
tags: [adr, dotfiles]
status: accepted
created: "2026-02-21"
owner: manu
---

# ADR-001: Custom Skills Over BMAD Framework

## Context

The dotfiles project needed a complete product development pipeline for AI-assisted workflows. Two approaches were evaluated:

1. **BMAD Method (v6):** Full agile development framework with 5 subdirectories (commands, config, skills, templates, utils), module-based skill organization (bmb, bmm, cis, core), and interdependent helper files. Provides PM, Scrum Master, and Developer skills with commands like `/prd`, `/sprint-planning`, `/create-story`.

2. **Custom Skills:** Lightweight `SKILL.md` files in `ai/skills/*/SKILL.md` format, each self-contained with YAML frontmatter (`name`, `description`) and markdown instructions. Deployed via glob loop in setup scripts.

### Key findings from BMAD evaluation:

- BMAD skills require `helpers.md` and a project config system — they cannot be cherry-picked as standalone files
- BMAD's module structure (bmm, bmb, cis, core) conflicts with the flat per-skill directory pattern already established
- The gap BMAD does not fill: converting PRD output into `gh issue create` calls (the GitHub Issues bridge)
- BMAD adds significant weight (5 subdirectories) for functionality that can be achieved with 3 focused skills

## Decision

Build custom skills (`prd`, `qa-plan`, `prd-to-issues`) in the existing `ai/skills/*/SKILL.md` format rather than adopting the BMAD framework.

## Consequences

### Positive

- **Zero dependencies:** Each skill is a single markdown file with optional `references/` assets
- **Composable:** Skills chain naturally (`brainstorming` → `prd` → `prd-to-issues` → `writing-plans`)
- **Portable:** Same skills deploy to Claude Code (`~/.claude/skills/`) and Gemini CLI (`~/.gemini/prompts/`)
- **Reproducible:** Setup scripts auto-discover new skill directories via glob loop — no script changes needed
- **Consistent:** Follows the same pattern as the 18 existing skills

### Negative

- **No multi-agent orchestration:** BMAD provides coordinated PM/Dev/QA agent personas; custom skills are single-agent
- **Manual composition:** Users must invoke skills in sequence rather than having a framework chain them
- **No project config system:** Each skill reads project state independently (PRD location, repo info)

### Mitigations

- The `claude-mem` plugin provides `make-plan` + `do` commands for multi-phase orchestration when needed
- Skill descriptions include references to related skills (e.g., `prd` suggests `prd-to-issues` as next step)
