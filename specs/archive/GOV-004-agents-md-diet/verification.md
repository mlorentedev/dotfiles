---
id: "GOV-004-agents-md-diet"
type: spec
status: complete
created: "2026-07-09"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — GOV-004-agents-md-diet

## Pre-move check (move-not-delete)

Confirmed each target pattern exists and holds the content before trimming the AGENTS.md duplicate:

| AGENTS.md section | Target pattern / Skill | Covered? |
|---|---|---|
| Technical Standards (6 langs) | `pattern-language-standards.md` | yes — Python/Go/TS/Java/Astro/Matlab all present |
| Architecture Patterns | `pattern-architecture.md` | yes — microservices + monolith trees |
| Vault Structure / Frontmatter | `pattern-ai-protocol.md` | yes — frontmatter law |
| Neural Hive phases | `pattern-workflow-protocol.md` | yes — 3-phase loop (Context Sync, Execution, Crystallization) |
| MCP bullet bodies | `pattern-mcp-*.md` | yes — all four exist |

## Post-edit evidence

| Check | Result |
|---|---|
| Byte count `wc -c AGENTS.md` | **19,834 bytes** (down from ~30,037 bytes, ~34% token reduction) |
| Line count `wc -l AGENTS.md` | **241 lines** |
| `scripts/compile-harness.sh --check` | **OK: no harness drift** |
| `bats tests/agents-md.bats` | **18/18 pass** |
| `bats tests/check-doc-paths.bats` | **16/16 pass** |
| Go CLI tests `go test ./...` | **15/15 packages OK** |
| Rule inventory (Standing Orders 1-9, Discipline Gate criteria, Operational corrections) | all present (grep-verified) |

## CI (post-push)

- [x] `spec-gate` green (GOV-004 spec touched).
- [x] `test` green (agents-md.bats + check-doc-paths.bats + Go unit tests).
