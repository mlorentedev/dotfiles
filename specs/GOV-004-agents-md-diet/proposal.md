---
id: "GOV-004-agents-md-diet"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, proposal, agents-md, governance, ssot]
template_version: "1.0"
---

# GOV-004-agents-md-diet

Diet the root `AGENTS.md` from ~488 to ~160-180 lines by moving duplicated
reference material behind pointers to the patterns that already own it, keeping
every behavioural rule.

## Why

`AGENTS.md` is read as the system prompt by every agent on every turn. At ~488
lines it is ~3x GitHub's empirical ~150-line guidance (beyond which their
2,500-repo analysis measured +20-23% inference cost with no performance gain) —
paid on every turn of every agent. Per `pattern-agents-md-consistency`, the file
is authoritative for **behavioural rules, decision hierarchy, standing orders**;
pattern *bodies* (`00_meta/patterns/*.md`) own the detail. Much of the current
bulk is reference tables that already end with "for detail, query <pattern>.md"
— i.e. the pattern is the SSOT and AGENTS.md restates it. That duplication is
the fat to trim.

Verified before moving (move-not-delete): the target patterns exist and contain
the equivalent content — `pattern-language-standards.md` (all 6 language tables),
`pattern-architecture.md` (dir structures), `pattern-ai-protocol.md`
(frontmatter law), `pattern-workflow-protocol.md` (the phases, more detailed than
here). AGENTS.md's pointers currently omit the `pattern-` filename prefix; fixed
in passing.

## What

- **Move** (delete the duplicate, keep an accurate one-line pointer): Technical
  Standards tables (-> `pattern-language-standards.md`), Architecture Patterns
  dir trees (-> `pattern-architecture.md`), Vault Structure + Frontmatter Law
  (-> `pattern-ai-protocol.md`), the detailed Model-Selection tables + provider
  overlay list (-> the `ai/<agent>/` overlay files), the Neural Hive Phase step
  lists (-> `pattern-workflow-protocol.md`), the per-server MCP bullet bodies
  (-> `pattern-mcp-*.md`).
- **Merge**: Competence Retention Protocol and Response Protocol fold into
  Identity / Operating Mode (they restate the same Fast/Socratic/Debug modes).
- **Tighten (keep every rule)**: Standing Orders prose (rules stay, rationale
  moves to the cited pattern); dedupe Operational Rules "past corrections"
  against Standing Orders, keeping the unique ones.
- **Keep as-is**: Identity, Decision Hierarchy, Security HALT, Code Quality,
  Knowledge Placement, Language Boundary, the SDD Discipline Gate (CI-enforced),
  the Pattern Catalog table and the compact MCP discovery rows (both are
  discoverability indices — `pattern-agents-md-consistency` anti-pattern:
  removing them makes agents stop discovering the primitive), and the
  **HARNESS GENERATED block byte-for-byte** (compiled from the vault; the #728
  sha guard + `compile-harness.sh --check` enforce it).

## Out of scope

- Editing the vault patterns themselves (they already hold the content).
- The nested `cli/AGENTS.md` and the sha guard (shipped in #728).
- Per-agent overlay files (`ai/<agent>/`) — unchanged; the diet only points at
  them.

## Risks / open questions

- **Losing a rule in prose-tightening.** Mitigated: a rule-inventory diff check
  — every Standing Order, Discipline Gate criterion, and Operational correction
  present before must be present after (moved, not dropped).
- **Generated-block corruption.** Mitigated: edits never touch the marker
  region; `tests/harness-generated-sha.bats` + `compile-harness.sh --check` run
  post-edit and gate CI.

## Acceptance criteria

- [ ] AGENTS.md is <= ~200 lines (target ~160-180), no behavioural rule lost.
- [ ] Moved sections replaced by an accurate pointer to the (prefix-correct)
      pattern filename; no dangling "query X.md" where X lacks `pattern-`.
- [ ] The HARNESS GENERATED block is byte-identical (sha guard green,
      `compile-harness.sh --check` clean).
- [ ] Discipline Gate section unchanged in substance (CI depends on it).
- [ ] Pattern Catalog table and MCP discovery rows retained.

## References

- GH issue: [#673](https://github.com/mlorentedev/dotfiles/issues/673) (parts b+c in #728)
- Structural SSOT: `pattern-agents-md-consistency` (reader/authority map)
- Move targets: `pattern-language-standards`, `pattern-architecture`,
  `pattern-ai-protocol`, `pattern-workflow-protocol`, `pattern-mcp-*`
