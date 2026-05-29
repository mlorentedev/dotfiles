---
id: adr-008-skills-ecosystem-overhaul
type: adr
status: active
created: "2026-03-29"
---

# ADR-008: Skills Ecosystem Overhaul

> **Supersession proposed by [ADR-013](adr-013-agent-artifact-deploy-engine.md)** (status: `proposed` — not yet in effect). ADR-013 subsumes the skills ecosystem into a manifest-driven agent-artifact deploy engine. This ADR remains `accepted` until ADR-013 is accepted.

## Status: Accepted

## Date: 2026-03-29

## Context

The dotfiles repo serves as the central deployment hub for AI agent skills across all projects and machines (Linux + Windows). Over 6 months of organic growth, the skills library accumulated to 23 skills with several problems:

- **Redundancy:** `skill-creator` (356 lines) and `writing-skills` (655 lines) both taught skill creation
- **Dead weight:** 4 skills (`brainstorming`, `prd`, `qa-plan`, `doc`) had no evidence of invocation in session memory
- **CSO violations:** 10 of 23 skill descriptions started with what the skill does instead of when to use it, causing Claude to summarize instead of reading the full body
- **Missing capabilities:** No vault structural maintenance skill, no project maturity auditing, no automated weekly maintenance
- **Behavioral rules in skills:** `using-superpowers` encoded a behavioral rule as a skill, which only fires when explicitly invoked -- defeating its purpose
- **No sync:** Setup scripts copied skills to targets but never cleaned up removed skills, causing stale skills to persist on deployed machines

## Decision

### Reduction: 23 to 17 skills (-26%)

**Deleted (7):** `using-superpowers`, `refactor`, `backlog`, `brainstorming`, `prd`, `qa-plan`, `doc`
- `using-superpowers` moved to CLAUDE.md behavioral rules
- `refactor` and `backlog` too thin (37-38 lines), already covered by CLAUDE.md Code Quality Rules
- `brainstorming`, `prd`, `qa-plan`, `doc` had zero usage evidence in 2 months of session memory

**Merged (2 into 1):** `skill-creator` + `writing-skills` into `creating-skills` (~200 lines from 1,011)

**Created (3):**
- `project-maturation` -- stack-aware maturity audit and improvement cycle
- `vault-doctor` -- vault structural maintenance (links, frontmatter, orphans)
- `creating-skills` -- merged skill creation guide with TDD testing methodology

**Modified (2):**
- `insights` -- added vault structural health tier, quick/full modes, decision persistence check
- `writing-plans` -- compressed TDD duplication, added pipeline cross-references

### CSO Audit

All 17 skill descriptions now start with "Use when..." and contain only triggering conditions, never workflow summaries. This follows the empirical finding that Claude follows description summaries as shortcuts instead of reading the full skill body.

### Standing Orders

6 non-negotiable behavioral rules added to the top of both CLAUDE.md and GEMINI.md:
1. Automate, don't instruct (stack-appropriate: scripts, IaC, Makefiles, Python CLIs, CI pipelines)
2. SSOT (code in git, knowledge in vault)
3. Vault hygiene (in-session, not "later")
4. Clean as you go
5. Consult patterns before architectural decisions (37 patterns catalogued)
6. Enterprise-grade or nothing (no hacks, proven patterns, scalable)

### Deployment: Sync Instead of Copy

Setup scripts (`setup-linux.sh`, `setup-windows.ps1`) now perform a mandatory sync:
1. Remove skill directories/prompts in target that don't exist in source
2. Copy current skills

This prevents stale skills from persisting on deployed machines after removal from the repo.

### Weekly Maintenance Automation

Three-layer approach:
- **System cron/Task Scheduler** (permanent): `vault-maintenance-weekly.sh` / `.ps1` runs Sundays 10:07 AM, executes `knowledge-crystallize.sh --all` + `vault-health.sh`, sends desktop notification
- **Claude Code CronCreate** (session bonus): Durable trigger for in-session reminders (7-day auto-expiry)
- **SessionStart hook** (passive safety net): Already warns when MEMORY.md is stale or vault health fails

## Cross-Platform Considerations

- All scripts have Linux (.sh) and Windows (.ps1) parity
- Notification: `notify-send` (Linux) / `System.Windows.Forms.NotifyIcon` (Windows)
- Scheduling: `crontab` (Linux) / `Register-ScheduledTask` (Windows)
- Skill sync: Both platforms use delete-then-copy pattern
- No non-ASCII characters in .ps1 files (PSScriptAnalyzer BOM requirement)

## Consequences

### Positive
- ~35% reduction in token cost per session (fewer skill descriptions loaded)
- Better auto-detection via CSO-optimized descriptions
- Vault maintenance now has dedicated tooling (insights + vault-doctor)
- Standing Orders ensure behavioral rules are seen first, not buried
- Weekly automation prevents knowledge drift

### Negative
- Gemini CLI doesn't support skill auto-triggering (prompts are reference-only)
- CronCreate doesn't persist durably -- system cron is the real backbone
- 4 deleted skills (brainstorming, prd, qa-plan, doc) cannot be recovered from deployed machines without re-running setup

### Neutral
- `audit` and `using-git-worktrees` were not in the original analysis but were kept as-is (both justified)
- Total final count: 17 skills (14 kept + 3 new)
