---
generated: true
generated_from: 00_meta/skills/read-all-adrs/SKILL.md
generated_sha: 9047ab3da31641f7
id: read-all-adrs-skill
type: skill
status: active
created: "2026-07-07"
owner: manu
name: read-all-adrs
description: Force a full read of every ADR in a repo before starting architecture-affecting work, producing a short decision inventory as an explicit completion artifact. Triggers on /read-all-adrs, "read all the ADRs", "give me the ADR map for this repo", "what decisions have already been made here", and as a mandatory pre-step before architecture-session, large refactors, or any change the Discipline Gate flags as warranting a Socratic Guardrail pause. Adapted from davidondrej/skills' read-all-adrs, rewritten with a real completion artifact instead of a bare "read them" instruction.
allowed-tools: [Read, Grep, Glob]
---

# Read All ADRs — full ingestion before architecture work

> Agents under time pressure sample ADRs (grep for a keyword, read the top hit) instead of reading the corpus, and re-propose something already decided-and-rejected, or miss a supersession banner and build on a dead decision. This skill forces full ingestion and produces a durable inventory so the reading actually happened — a bare "I read them" claim is not verifiable; a decision-map table is.

## When to use

- Before `architecture-session` (Phase B's "Regla del 3" reference audit needs this as input).
- Before any refactor the Discipline Gate flags as warranting a Socratic Guardrail pause (schema design, concurrency, breaking changes).
- When picking up a repo cold, or returning after a long gap — "what has already been decided here?"
- When the user asks directly for an ADR map / decision inventory.

## When to skip

- A one-file, low-risk change with no architectural surface — don't gate trivial work behind a full ADR sweep.
- The repo has no `docs/adr/` (or equivalent) — nothing to read; say so and stop.

## Procedure

1. **Locate the corpus.** Glob `docs/adr/**/*.md` (or the repo's documented equivalent — check `docs/README.md` / `AGENTS.md` for the actual path first, don't assume). Include audit-style records that live alongside ADRs (e.g. `audit-NNN-*.md`) if the repo's convention co-locates them, per that repo's own placement decision.
2. **Read every file — no sampling.** Full read (or full `view` in range chunks for large files), not a keyword grep. A grep hit tells you a file mentions a term; it does not tell you the file is irrelevant when it doesn't.
3. **Build the inventory** as you go — one row per ADR:

   | ID | Title | Status | One-line decision | Superseded by / relates to |
   |---|---|---|---|---|

   Pull `status` from frontmatter (`accepted`/`proposed`/`superseded`/`active`), not from assumption — a stale `status: proposed` on a fully-shipped decision is itself a finding worth flagging (see step 4).
4. **Flag drift while reading, don't silently pass over it:** a `status` that contradicts the body, a missing supersession banner where a later ADR clearly overrides an earlier one, or two ADRs making contradictory claims about the same fact (the exact class of finding a docs audit looks for). Surface these explicitly in the inventory's last column or a short "Flags" section — don't fix them unprompted unless the fix is trivial and directly in scope.
5. **State completion explicitly**: "Read N/N ADRs in `<repo>`" with the inventory table, before proceeding to the architecture work that triggered this skill. The inventory IS the completion artifact — if you can't produce it, you haven't actually read the corpus.

## Anti-patterns

- **Grep-and-call-it-read** — searching for a keyword across ADRs and treating the hits as "having read the ADRs". This skill exists specifically to close that gap.
- **Silent supersession** — noticing ADR-004 contradicts ADR-012 and saying nothing because it's out of scope for the current task. Flag it (a one-line note is enough); fixing it is a separate, ownable decision.
- **Inventory without action** — producing the table and then not actually using it to inform the architecture work that triggered the read. The point is input to a decision, not a checkbox.

## References

- Adapted from `davidondrej/skills/read-all-adrs` (external skills audit, 2026-07-07) — the original is a bare "read them all" instruction with no completion artifact; this version adds the inventory table and drift-flagging as the verifiable output.
- [[00_meta/skills/architecture-session/SKILL|architecture-session]] — Phase B's reference audit consumes this skill's inventory.
- [[pattern-knowledge-placement]] — where ADRs live (repo `docs/adr/`, project-bound, never the vault).
