---
id: "HARNESS-046"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-25"
issue: "mlorentedev/dotfiles#562"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-046

## Why

`00_meta/agents/ROSTER.md` has declared six invocable roles — one per phase of the work cycle — since v1, and only `curator` was ever written. Five phases had a catalog row and nothing any harness could deploy, and the autonomous instance the roster names (`hermes-nan`) had no catalog entry at all. The consequence is measurable rather than aesthetic: the orchestration epic reads as built while the thing it orchestrates is one seventh present, which is why the epic has felt long with no observable benefit.

## What

`00_meta/agents/definitions/` gains `architect`, `planner`, `builder`, `reviewer`, `shipper` and `hermes-nan`, and `scripts/compile-harness.sh --refresh` renders all seven into `harness/agents/`. `dotf doctor`'s agent-tier check goes from validating one record to validating seven. Each definition declares a neutral tier and its forced skills; none declares a model.

## Out of scope

- **Wiring the executor to the personas.** `dotf agent run --role X` does not read these definitions — nothing under `cli/internal/agent` references `harness/agents`, and `role` is passed through as a string. That gap is real and is stated in the PR rather than closed here; a persona that renders is not a persona that is consumed.
- **Adding a second entry to `tiers.top`.** It currently has one pool and no fallback, so `architect` and `curator` cannot be served when that pool is unavailable. Separate concern.
- **Reconciling the persona tier with the reviewer pool.** `reviewer` declares `mid`, whose chain begins with an Anthropic model, while the adversarial-review pool excludes Anthropic models by standing rule. The pool enforces independence on its own; nothing reconciles the two declarations, and nothing here changes that.

## Risks / open questions

- **A definition can declare a tier that no longer resolves.** Mitigated by the existing `dotf doctor` agent-tier check, which is exactly the consumer this change grows from 1 record to 7.
- **The roster and the definitions can drift.** They already had: `architect` broke the roster's own `>=3 skills` rule and the `curator` row omitted `dispose-proposals`. Both corrected here in the same commit as the definitions; nothing yet *prevents* the next drift, which is the honest limit of this change.
- **Records are generated and carry a `generated_sha`.** Hand-editing them under `harness/agents/` produces silent drift from the vault SSOT. The engine's drift check covers this.

## Acceptance criteria

- [ ] All six invocable definitions plus the `hermes-nan` catalog entry render via `compile-harness.sh --refresh`
- [ ] A second `--refresh` pass is byte-identical (`changed=0`) — a generator that changes something every run is not one
- [ ] `dotf doctor`'s agent-tier check validates seven records and every declared tier resolves
- [ ] `ROSTER.md` and the definitions agree on forced skills, and no role bundles fewer than three
- [ ] `hermes-nan` points at `80_agents/hermes-nan/` and duplicates none of its state

## References

- Bitácora board: `mlorentedev/dotfiles#562` (see the `issue:` frontmatter field)
- Epic: `#558` (HARNESS-042, cross-harness agent pipeline)
- ADR: `docs/adr/adr-027` §4/§6, `docs/adr/adr-032-cross-harness-agent-orchestration.md`
- Design SSOT: `00_meta/agents/ROSTER.md` (vault) — the phase axis and the forced-skill sets predate this spec
