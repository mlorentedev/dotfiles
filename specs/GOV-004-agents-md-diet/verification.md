---
id: "GOV-004-agents-md-diet"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — GOV-004-agents-md-diet

## Pre-move check (move-not-delete)

Confirmed each target pattern exists and holds the content before deleting the
AGENTS.md duplicate:

| AGENTS.md section | Target pattern | Covered? |
|---|---|---|
| Technical Standards (6 langs) | `pattern-language-standards.md` (75L) | yes — Python/Go/TS/Java/Astro/Matlab all present |
| Architecture Patterns | `pattern-architecture.md` (34L) | yes — microservices + monolith trees |
| Vault Structure / Frontmatter | `pattern-ai-protocol.md` (82L) | yes — frontmatter law |
| Neural Hive phases | `pattern-workflow-protocol.md` (223L) | yes — more detailed than here |
| MCP bullet bodies | `pattern-mcp-*.md` (41-64L each) | yes — all four exist |

## Post-edit evidence

| Check | Result |
|---|---|
| Line count `wc -l AGENTS.md` | **237** (from 487, -51%) |
| HARNESS block sha guard `bats tests/harness-generated-sha.bats` | **2/2 pass**; sha still `e9c8d9d67d9ce58f` |
| `scripts/compile-harness.sh --check` | **OK: no harness drift** |
| Rule inventory (Standing Orders 1-9, Discipline Gate criteria, Operational corrections) | all present (grep-verified) |
| Pattern pointers | all include `pattern-` prefix; no dangling ref |

Landed at 237, not the ~160 the issue targets: the remainder is genuine
behavioural content (Standing Orders, Decision Hierarchy, Language Boundary,
Knowledge Placement, the SDD Discipline Gate) that would cost a rule to cut. The
token-heavy reference tables (6 language tables, MCP bodies, dir trees, phase
lists) — the actual per-turn cost — are gone.

## CI (post-push)

- [ ] `spec-gate` green (GOV-004 spec present; large removal is a substantive spec touch).
- [ ] `test` green (harness-generated-sha + docs-drift).
- [ ] `lint` / `integration` unaffected.
