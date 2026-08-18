---
id: "CLI-040-dotf-search-and-suggest"
type: spec
status: active
created: "2026-08-18"
issue: "mlorentedev/dotfiles#1066"
tags: [spec, proposal, search, harness]
template_version: "1.0"
---

# CLI-040 — Fast Knowledge Search & Dynamic Harness Suggest

## Why

Navigating the cross-project knowledge vault, patterns, and skills currently requires manually browsing directory trees or spawning external MCP queries. Providing native, sub-millisecond local search and task-based skill/pattern suggestions directly inside the Go `dotf` CLI enables both human operators and autonomous agents to locate relevant doctrine, troubleshooting runbooks, and skills instantly with zero cloud latency and no extra dependencies.

## What

1. **`dotf harness suggest [--prompt "text"] [paths...] [--diff] [--json]`**:
   - Analyzes a task description (prompt), touched files, or unified git diff against trigger rules.
   - Outputs matching pattern IDs and recommended skill names.
2. **`dotf search <query...> [--type all|pattern|skill|lesson|spec|doc] [-n limit] [--json] [--dir path]`**:
   - Parses frontmatter metadata, titles, and body content across the knowledge vault.
   - Employs token-weighted ranking (exact ID > title > keywords/tags > description > body).
   - Returns ranked matches with paths, metadata, scores, and contextual text snippets.
3. Also wires `dotf vault search` as a subcommand under `dotf vault`.

## Out of scope

- Embedding-based vector database backends (pure lexical and frontmatter indexing provides deterministic, zero-dependency sub-millisecond queries).
- Modifying or patching vault files during search.

## Risks / open questions

- **Vault location resolution**: Ensured via `vault.ResolveVault()` using the standard machine.json / `$VAULT_PATH` cascade with fallback to the local repo.
- **Large corpus walk time**: Directory pruning skips `.git`, `node_modules`, and hidden folders for instant scans.

## Acceptance criteria

- [x] `dotf harness suggest` resolves patterns and skills from prompt strings, file arguments, and diffs.
- [x] `dotf search` indexes frontmatter, keywords, and markdown body with relevance scoring.
- [x] `--type` filter restricts results to specific item classifications (`pattern`, `skill`, `lesson`, `spec`, `doc`).
- [x] `--json` emits structured machine-readable JSON for both commands.
- [x] Complete Go unit tests (`cli/internal/search/`, `cli/internal/harness/`) and Bats integration suites pass.

## References

- Issue: [mlorentedev/dotfiles#1066](https://github.com/mlorentedev/dotfiles/issues/1066)
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md`, `00_meta/patterns/pattern-knowledge-placement.md`
