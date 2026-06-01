---
id: "ADR-016-deploy-canonical-agents-to-vault"
type: adr
status: accepted
owner: manu
date: "2026-05-31"
extends: [adr-009-multi-agent-runtime, adr-013-agent-artifact-deploy-engine, adr-015-remote-self-provisioning-agent]
tags: [architecture, decision, agents, hermes, hermes-002, vault-ssot, agents-md, compile-harness, packaging, idea-file]
created: "2026-05-31"
---

# ADR-016: Deploy canonical AGENTS.md to the vault for repo-less agents (HERMES-002)

> Closes the three-layer drift debt anticipated in ADR-015. Builds on ADR-009 (AGENTS.md cross-agent SSOT) and ADR-013 (compile-harness deploy engine). Fuses with the "idea-file packaging" direction recorded in the 2026-05-31 prestudy.

## Status

Accepted

## Date

2026-05-31

## Context

ADR-015 shipped Hermes as a remote, repo-less agent with a three-layer instruction setup: the root `AGENTS.md` (canonical SSOT), the `ai/hermes/AGENTS.md` overlay (provisioning notes), and the vault constitution `80_agents/hermes-nan/AGENTS.md` (operating law).

Reading the three together, the real problem is **not duplication** — the constitution is mostly Hermes-specific (write-zone, sync, bootstrap, secrets) and duplicates little. The real problem is **access**: the constitution orders the agent to *"defer to the dotfiles root `AGENTS.md` as canonical authority"* for anything it does not cover (Standing Orders, Decision Hierarchy, Neural Hive Loop, MCP usage, Operational Rules) — but Hermes cannot reach the root, because it does not clone dotfiles. The "defer to root" is a pointer to an unreachable file. The canonical behavior contract is unreachable for the one agent that most needs it.

Horizon (confirmed this session): **at least 2-3 additional repo-less agents are foreseen**, and the "packaging / idea-file" direction in the 2026-05-31 prestudy points the same way. A per-agent runtime fetch does not scale to that.

`compile-harness.sh` (ADR-013) already implements the exact pattern needed, in the opposite direction: it deploys vault patterns/skills → dotfiles as committed, marker-delimited, CI-verifiable generated artifacts. HERMES-002 is one more output of the same engine: root `AGENTS.md` → vault.

## Decision Drivers

- A repo-less agent must reach the canonical behavior SSOT through the only channel it has (the vault, via Hive).
- Scale to N repo-less agents with zero per-agent work.
- Keep the "Hermes reads everything from the vault" design — no second knowledge channel.
- Reuse the existing deploy engine; no new infrastructure.
- Respect the knowledge-placement directionality invariant (a repo never depends on the store).

## Considered Options

1. **Deploy canonical `AGENTS.md` → vault via compile-harness, as a generated marker-delimited artifact** (CHOSEN). Build-time resolution + canonical-channel distribution.
2. **Curl the root `AGENTS.md` from GitHub during Hermes bootstrap.** Rejected — runtime fetch adds a second knowledge channel and a network dependency on the critical bootstrap path; it is re-implemented per agent and does not scale to the foreseen 2-3 more agents.
3. **CI drift check only** (assert the constitution reflects the root cross-agent sections). Rejected — detects drift but does not close the access gap; reconciliation stays manual.
4. **Invert the SSOT to the vault.** Rejected — breaks native in-repo reads (the local agents read the root directly) and violates knowledge-placement (the repo would depend on the store).

## Decision

Extend `compile-harness.sh --refresh` with an output that copies the root `AGENTS.md` to a canonical vault path consumed by repo-less agents, as a generated artifact:

1. **Target path:** `80_agents/_shared/canonical-agents.md` — shared across repo-less agents, not Hermes-specific.
2. **Generated + marked:** wrapped in the same `BEGIN/END HARNESS GENERATED` marker discipline as the existing override block, with a provenance + sha line, so it is never hand-edited; `--check` verifies it (CI / healthcheck).
3. **Constitution slims:** `80_agents/hermes-nan/AGENTS.md` keeps only Hermes-specific operating law and replaces the unreachable "defer to root" with a pointer to the deployed `canonical-agents.md`.
4. **Whole file now, sections later (YAGNI):** deploy the entire root `AGENTS.md`; technical sections that do not apply to an ops agent (language standards, frontend, MATLAB) are inert noise. Filter by section only if the noise proves costly.
5. **Idea-file fusion:** this canonical-in-vault artifact is the core of the "idea-file" packaging deliverable — a fresh repo-less agent reads it (plus its own constitution) to self-instantiate. Packaging and HERMES-002 are one decision, not two.

**Direction note (intentional):** compile-harness now carries two datums with opposite SSOTs — *patterns* (vault → repo) and *AGENTS.md* (repo → vault). Each flows from its own owner: patterns are the cross-project brain (vault), `AGENTS.md` is the behavior contract (repo). This does **not** violate the directionality invariant: dotfiles never depends on the vault for `AGENTS.md` (in-repo agents read the root natively); the vault receives a generated copy solely to serve repo-less agents.

## Consequences

- **Positive:** repo-less agents reach the canonical SSOT through the vault; scales to N agents with zero per-agent work; reuses the ADR-013 engine (no new infrastructure); closes the ADR-015 three-layer drift debt; the artifact doubles as the idea-file packaging core.
- **Negative / debt:** the vault copy is **stale between `--refresh` runs** — accepted, because a behavior contract does not change by the minute and cross-agent consistency matters more than intra-day freshness. Propagation requires a `--refresh` + vault commit; if forgotten, repo-less agents lag. Mitigate by running `--refresh` on the same cadence as harness regeneration and asserting freshness in `--check` where feasible.
- **Follow-up:** implementation is a separate SDD feature (`HERMES-002`): extend compile-harness, slim the constitution, add the CI assertion, and produce the idea-file doc. This ADR records the *why*; the spec implements the *how*.

## References

- ADR-009 (AGENTS.md cross-agent SSOT), ADR-013 (compile-harness deploy engine), ADR-015 (remote self-provisioning agent — explicitly anticipates this as "HERMES-002 territory").
- Prestudy: vault `dotfiles/25-prestudy/2026-05-31-knowledge-agent-stack-vs-state-of-art.md` (idea-file packaging adoption; N=3 reference audit).
- External: Karpathy "LLM Knowledge Bases" idea-file (ship the idea, the agent instantiates); graphify per-agent installer (one core + generated shims).
