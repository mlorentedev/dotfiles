---
id: enrich-us-skill
type: skill
status: active
created: "2026-05-14"
owner: manu
name: enrich-us
description: Take a backlog item or pasted user story and rewrite it into an implementation-ready form (fields, endpoints, files-to-modify, DoD, non-functional requirements). Triggers on /enrich-us, "enrich SDD-014", "rewrite this user story", "make this ticket implementation-ready". Single-shot rewrite (NOT Socratic — that is /spec fill's job). Outputs `## Original` + `## Enhanced` sections. Ported and adapted from LIDR-academy/lidr-specboot.
allowed-tools: [Bash, Read, Grep, mcp__hive__vault_query, mcp__hive__vault_search, mcp__hive__vault_patch]
---

# Enrich User Story / Backlog Item

> Take a backlog item or pasted user story and rewrite it into an implementation-ready form: complete fields, endpoints, files-to-modify, DoD, non-functional requirements. Output two sections — `## Original` and `## Enhanced` — so the user can paste the enhanced version into `specs/<feature-id>/proposal.md` (Why/What) or back into the GitHub issue.
>
> **Origin:** ported from [LIDR-academy/lidr-specboot](https://github.com/LIDR-academy/lidr-specboot/blob/main/ai-specs/skills/enrich-us/SKILL.md) (`enrich-us`, MIT). Adapted: Jira mode removed; task tracking via bitácora GitHub Project (ADR-018); technical context resolved from `00_meta/patterns/`; backlog-item lookup added.

## When to use

- `/enrich-us <backlog-id-or-pasted-story>` explicitly.
- "Enrich SDD-014" / "rewrite this user story" / "make this ticket implementation-ready".
- Before invoking `/spec fill <feature-id>` when the input from the GitHub issue or backlog is too thin to drive Socratic Q1–Q6 — enrich first, then fill.

## When NOT to use

- Trivial changes (typo, comment-only, mechanical rename) per `pattern-spec-driven-development` Skip rules.
- The backlog item is already detailed enough to fill a proposal directly. Heuristic: if it already names files-to-modify + DoD + acceptance criteria, do not enrich.
- The change is exploratory / spike. Spikes do not benefit from upfront enrichment.

## Inputs

Two modes, auto-detected:

1. **Backlog-id mode.** Input matches `^[A-Z]+-\d+(-[a-z0-9-]+)?$` (e.g. `SDD-014`, `AI-001-ollama-public`). Resolve the original:
   - Read the GitHub issue via `gh issue view <id> --json title,body` (bitácora is the SSOT per ADR-018).
   - If not found, ask the user to paste the story.

2. **Pasted-story mode.** Input is multi-line markdown. Use as-is.

## Instructions

Act as a product expert with technical fluency in the project's stack. Read at most 2-3 relevant patterns from `$VAULT_PATH/00_meta/patterns/` based on the story's domain (e.g. for an HTTP endpoint task, read `language-standards`, `testing-standards`, `observability`). Do not exhaust the catalog — the goal is enrichment, not citation.

Validate that the story includes ALL of:

1. **Full functionality description** — what changes, observable from outside.
2. **Comprehensive list of fields / data shapes / migrations** if data state is touched.
3. **Endpoints affected** — method, URL, request/response schema. Only if the change is HTTP/API.
4. **Files/modules to modify** — concrete paths, aligned with the repo's architecture (consult `pattern-architecture` and `pattern-project-structure` patterns).
5. **Definition of done** — implementation + delivery steps (build, test, deploy, docs).
6. **Tests** — what test scaffolds need updates (unit, integration, contract). Reference `testing-standards`.
7. **Non-functional requirements** — security, performance, observability, error handling. Reference `secrets-security`, `observability` when applicable.

If the story is missing ANY of the above relevant to its domain, produce an enhanced version. Use the **same voice and constraint level** as the original (do not invent scope; do not soften clear directives).

## Output

Always two top-level sections, in this exact order:

```markdown
## Original

<verbatim copy of the input, code-fenced if multi-line>

## Enhanced

<the rewritten user story, structured with the relevant items from the Validation list above>
```

If `--write-back` is passed AND the input was backlog-id mode, add a comment to the GitHub issue:

- Append `<!-- enriched 2026-MM-DD -->` as an issue comment via `gh issue comment`.
- Do NOT replace the issue body itself — the body stays terse; the enhanced version belongs in `specs/<feature-id>/proposal.md` once the user creates the spec.

## Edge cases

- **Ambiguous input** (short reference without ID format, no body): ask whether to resolve from a GitHub issue or to paste the full story. Do not guess.
- **Backlog item is already done (`[x]`):** still allowed; enrichment becomes a post-hoc spec for audit. Note this in the Enhanced section header.
- **No matching pattern found in the catalog:** proceed without pattern citation; do not fabricate a pattern name. Flag the gap to the user as a possible new pattern candidate.

## Anti-patterns

- Do not turn enrichment into Socratic Q&A — that is `/spec fill`'s job. `enrich-us` is a single-shot rewrite.
- Do not change the user's intent — enrich for clarity, not for redirection.
- Do not paste pattern bodies into the Enhanced output — link/reference them by name only.
