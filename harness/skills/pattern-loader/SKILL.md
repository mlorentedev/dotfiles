---
generated: true
generated_from: 00_meta/skills/pattern-loader/SKILL.md
generated_sha: 1b2c8b9e354b82cf
id: pattern-loader-skill
type: skill
status: active
created: '2026-05-31'
owner: manu
name: pattern-loader
description: Use when a task matches a known workflow pattern in vault/00_meta/patterns/.
  Searches the pattern catalog, loads the most relevant pattern, and applies it. Ensures
  agents always use the latest version of patterns without caching stale copies in
  memory.
keywords: [load pattern, pattern catalog, consult pattern, buscar patron]
paths: [00_meta/patterns/**]
---
# Pattern Loader

## Overview

When a task matches a known workflow pattern, load it from the vault pattern catalog instead of guessing. This ensures agents always use the latest version of patterns without caching stale copies in memory.

**Core principle:** Patterns live in the vault. Agents load them on demand. Never cache pattern content in long-term memory — always re-read from vault to stay current.

## When to use

- A task matches a known pattern name or category (e.g., "docker", "secrets", "git workflow", "spec")
- Manu says "use the [pattern-name] pattern"
- A workflow feels familiar but you're not sure of the exact steps
- Starting a session and you want to check which patterns are relevant

## When to skip

- The task is trivial and doesn't need a pattern
- You already know the exact steps from recent memory (e.g., just did this task 2 hours ago)
- No matching pattern exists in the catalog

## The protocol

### Step 1 — Search the pattern catalog

The categorized catalog is `00_meta/patterns/_index.md` (one line per pattern). To search by keyword, prefer Hive MCP — it is path-agnostic and always reads the live vault. `scope` names a vault zone (`meta`, `projects`, `work`, `agents`), never `patterns`; `type_filter` narrows the hits to pattern files:

```
vault_search(query="<keyword>", scope="meta", type_filter="pattern", ranked=True)
```

If Hive is unavailable, fall back to a file search against your local vault
clone. Resolve the path via: (1) `$VAULT_PATH` env var, (2) `dotf env path VAULT_PATH`,
(3) `~/.config/dotfiles/machine.json` `paths.VAULT_PATH`. **Never hardcode a literal path.**
If none resolve, ask the user for the vault path:

```bash
VAULT_PATH="${VAULT_PATH:-$(dotf env path VAULT_PATH 2>/dev/null || grep -o '"VAULT_PATH": *"[^"]*"' ~/.config/dotfiles/machine.json 2>/dev/null | cut -d'"' -f4)}"
grep -rli "<keyword>" "${VAULT_PATH}/00_meta/patterns/" 2>/dev/null
```

### Step 2 — Match and load

Read the most relevant pattern file(s). Each pattern has frontmatter:

```yaml
---
id: pattern-<name>
type: pattern
status: active
created: "<YYYY-MM-DD>"
tags: [<topic>, <topic>]
---
```

Use the `tags` and `id` to determine relevance. Read the full content.

### Step 3 — Apply the pattern

Follow the pattern's steps exactly. If the pattern has sections like:

- **Overview** — context and intent
- **When to use** — trigger conditions
- **Steps** — numbered procedure
- **Pitfalls** — common mistakes to avoid
- **Verification** — how to confirm success

Follow all sections in order. Do not skip pitfalls.

### Step 4 — Log which pattern was used

Add a brief note to the current session record:

```
Pattern applied: pattern-<name>.md
```

This is for audit trail — not for long-term memory.

### Step 5 — Never cache content

However often a pattern recurs, memory may hold its name (a pointer), never a copy of its content. Always re-read the pattern from the vault, so a change to it is picked up.

## Pattern categories

The categorized catalog is `00_meta/patterns/_index.md`. These filename clusters exist on disk today and make good search hints; when one drifts, the index and the directory listing win over this table:

| Cluster | What it covers | Examples |
|---------|---------------|----------|
| `pattern-git-*`, `pattern-github-*` | Git and GitHub workflow | pattern-git-workflow, pattern-github-branch-hygiene, pattern-github-repo-bootstrap |
| `pattern-secrets-*` | Secrets storage, use, rotation | pattern-secrets-security, pattern-secrets-rotation |
| `pattern-testing-*`, `pattern-*-testing` | Test conventions and suite strength | pattern-testing-standards, pattern-integration-testing, pattern-mutation-testing |
| `pattern-mcp-*` | MCP servers and tool use | pattern-mcp-tool-design, pattern-mcp-server-distribution, pattern-mcp-context7 |
| `pattern-spec-*`, `pattern-*-lifecycle` | Spec-Driven Development, change lifecycle | pattern-spec-driven-development, pattern-change-lifecycle, pattern-three-layer-proposal-lifecycle |
| `pattern-agent-*`, `pattern-cross-agent-*` | Agent workflows and pipelines | pattern-agent-orchestration, pattern-cross-agent-skill-pipeline |
| `pattern-docker-*`, `pattern-container-*` | Containers and image lifecycle | pattern-container-workflow, pattern-docker-tag-lifecycle |
| `pattern-shell-*` | Shell scripting | pattern-shell-standards, pattern-shell-advanced |
| `pattern-python-*` | Python CLIs and packaging | pattern-python-cli, pattern-python-pypi-pipeline |
| `pattern-verif*`, `pattern-detection-*` | Verification and enforcement | pattern-verification-fails-toward-unproven, pattern-verify-state-before-acting, pattern-detection-vs-enforcement |
| `pattern-knowledge-*`, `pattern-*memory*` | Where knowledge lives, memory | pattern-knowledge-placement, pattern-dual-memory, pattern-memory-consolidation |
| `pattern-release-*`, `pattern-version-*` | Releases and versioning | pattern-release-please-ci, pattern-version-single-source |

## Pitfalls

- **Don't guess the pattern name** — search first, match second
- **Don't cache patterns in long-term memory** — they change, cached versions become stale
- **Don't skip pitfalls** — they exist for a reason
- **Don't apply a pattern that doesn't fully match** — partial application causes bugs
- **Don't assume a pattern exists** — if no match, say so and proceed without it

## Verification

After applying a pattern:

1. Re-read the pattern's **Verification** section
2. Execute the verification steps
3. Report results to Manu
4. If verification fails, record it with `capture_lesson`

## References

- Pattern catalog: `vault/00_meta/patterns/`
- Pattern format: see any file in `00_meta/patterns/pattern-*.md`
- Related: `00_meta/skills/verification-before-completion/SKILL.md` (always verify after applying)
