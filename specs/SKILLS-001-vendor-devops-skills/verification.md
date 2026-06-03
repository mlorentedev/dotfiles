---
id: "SKILLS-001-vendor-devops-skills"
type: verification
status: draft
created: "2026-06-03"
template_version: "1.0"
---

# SKILLS-001 — Verification

| AC | How verified | Evidence |
|---|---|---|
| AC1 | 5 vault `SKILL.md` exist with name+description; `--check` validates frontmatter against the schema. | ls + --check |
| AC2 | `--refresh` produced `harness/skills/{terraform,helm,mcp-builder,golang-pro,async-python-patterns,vault-sync}`. | git status (6 new dirs) |
| AC3 | `compile-harness.sh --check` → exit 0, "no harness drift". | command output |
| AC4 | `harness/skills/ATTRIBUTION.md` lists all 5 sources; each SKILL.md ends with a source/license footer. | grep |
| AC5 | Audit recorded in proposal (3 dropped dupes + terraform 3→1). | proposal.md |
| AC6 | Vault `master` commit SHA (5 sources) + dotfiles PR (records+attribution+spec) closing #158. | git log / gh pr |
| AC7 | Epic #162 #158 checkbox `[x]` after merge. | gh issue view |

## License due-diligence (per source, read at vendor time 2026-06-03)
- terraform — antonbabenko/terraform-skill — Apache-2.0 (LICENSE read).
- helm — laurigates/claude-plugins — MIT (LICENSE read).
- mcp-builder — anthropics/skills — Apache-2.0 (per-skill LICENSE.txt read).
- golang-pro — jeffallan/claude-skills — MIT (LICENSE read).
- async-python-patterns — wshobson/agents — MIT (LICENSE read).
All permit redistribution with attribution; ATTRIBUTION.md + per-skill footer preserve the notice.

## Anti-scope confirmation
Upstream `references/` deep-dives NOT copied (linked upstream). No `manifest.json` change (skills auto-discovered). Windows deploy parity deferred to the batched Windows session (records are OS-agnostic markdown).
