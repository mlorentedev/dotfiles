---
id: "ADR-026-agent-config-ssot-topology"
type: adr
status: accepted
owner: manu
date: "2026-06-21"
supersedes: [adr-016-deploy-canonical-agents-to-vault]
extends: [adr-009-multi-agent-runtime, adr-012-deploy-strategy-copy-with-drift-assertion, adr-013-agent-artifact-deploy-engine]
tags: [architecture, decision, agents, harness, ssot, data-plane, control-plane, vault, agents-md, manifest]
created: "2026-06-21"
---

# ADR-026: Agent-config SSOT topology — owner-by-layer, committed projections, manifest-governed

> Settles the recurring session-after-session confusion: *"when I change a rule, where do I edit it, and where does each agent artifact live?"* Decision = a four-zone, owner-by-layer SSOT with committed projections, governed by the ADR-013 manifest. Extends ADR-009/012/013; supersedes ADR-016's directionality for the cross-project slice.

## Status

Accepted (model). Implementation is **gated** on `knowledge#120` (reliable cross-machine vault sync) and lands via the existing `HARNESS-001` deploy-engine epic — not as a new infrastructure track.

## Date

2026-06-21

## Context

The recurring pain is an editing-workflow ambiguity: a behavioural rule, a CLI-tool quirk, and an agent's private state all *feel* like "agent config", but they have different owners and different correct homes — and that was never written down. Every session re-derived it.

A first framing — *"move all SSOT into the vault `00_meta` and make dotfiles a pure render/deploy tool"* — was tested in this session and found wrong in two independent ways:

1. **The strong form was already rejected.** ADR-009 (Alt-2) rejected making `CLAUDE.md` the SSOT read by others; ADR-016 (Opt-4) rejected inverting the AGENTS.md SSOT to the vault. Both rest on the **directionality invariant** of knowledge placement (`pattern-knowledge-placement` in the vault): *a repo never depends on the store.* If a repo's behaviour contract lives only in the private vault, every repo — and every fresh machine — depends on the vault to bootstrap and cannot be opened share-ready (KPM-001 onboarded 11 repos specifically to *not* depend on the store).

2. **"dotfiles = pure deployer" is a root framing error.** The dotfiles CI has **zero visibility into the private vault**, so a contract that lives only in the vault cannot be verified in CI — this is exactly the regression class of #156. The rendered artifacts must **live committed** in dotfiles as a verifiable projection (DTO), per ADR-013 §3 (generate-and-commit). The committed render is also the bootstrap fallback and what keeps repos share-ready. dotfiles holds real, load-bearing content; it is not a pure deployer.

A concrete incoherence grounded the decision: the deploy manifest (`harness/manifest.json`) governs only `agents` (native) and `claude`. `AGY.md` and `copilot-instructions.md` are instruction surfaces **hand-maintained outside the engine** → live #156-class drift for those agents. The "incoherence" felt in `ai/<agent>/` is not a design flaw; it is ADR-013 rolled out at ~40%.

## Decision Drivers

- Stated objective: **automation + reuse + organization + maintainability.**
- The directionality invariant (a repo never depends on the store) — load-bearing for self-containment, bootstrap, and share-ready repos.
- CI verifiability: consumers read **committed projections**, never the live private vault.
- Kill the #156 drift class for **every** instruction surface, not just `claude`.

## Decision

### 1. Owner-by-layer SSOT — the owner of a datum decides where its SSOT lives

| Datum | Owner | SSOT location | Consumer reads |
|---|---|---|---|
| Cross-project behavioural slice (Identity, Standing Orders, Model tiers, Competence Retention) | the cross-project brain | **vault `00_meta/patterns`** (`enforce: true`) | the committed render |
| A repo's own behaviour contract | the repo | **repo-root `AGENTS.md`** | the file itself (self-contained) |
| Common agent library (definitions, doctrine, templates shared across agents) | the cross-project brain | **vault `00_meta/agents`** | read-shared |
| A deployed autonomous agent's private state (config sync, backups, file exchange) | that agent | **vault `80_agents/<agent>`** | agent-write-only |
| A CLI tool's runtime config + tool-specific quirks | the tool | **dotfiles `ai/<agent>`** | hand-authored / engine-rendered |

### 2. Committed projections, not pure deployment

The flow is **vault SSOT → dotfiles renders AND COMMITS the DTO → CI verifies the commit → deploy** (ADR-013 §3). The committed render is load-bearing: it makes CI verification possible without the private vault, keeps repos share-ready, and is the bootstrap fallback. No "embedded stale fallback" — the committed render *is* the fallback.

### 3. The manifest governs EVERY instruction surface (complete ADR-013)

Every instruction surface (`claude`, `agy`, `copilot`, `hermes`, and the native `AGENTS.md`) becomes a manifest target with a declared `kind` (`native` | `pointer` | `provisioned`). No instruction file is hand-maintained. The manifest is the **single legend** for `ai/<agent>/`: it explains why `opencode`/`pi` carry no instruction file (native readers) and `claude`/`agy` do (pointers). Within `ai/<agent>/`, the engine-rendered overlay (carrying the `GENERATED` marker) is kept distinct from hand-authored runtime config.

### 4. Zone symmetry — read-down library vs write-up state

- `00_meta/agents` is a **descending**, curated library that agents *read* (definitions, doctrine, the `_template/`).
- `80_agents/<agent>` is an **ascending**, private channel the agent *writes* (and only it), per the AGENTS.md write-boundary rule.

These are not two versions of one thing: one is the shared recipe, the other is each instance's private kitchen.

**Boundary `00_meta/agents` vs `00_meta/patterns`:** if it becomes text inside an `AGENTS.md`/`CLAUDE.md` → it is a `pattern` (`enforce: true`, rendered in); if it is scaffolding to *instantiate* an agent → it is in `agents`.

## Edit workflow (the practical contract this ADR exists to fix)

| I want to change… | I edit… | How it reaches the runtime |
|---|---|---|
| A rule for **every agent in every repo** | `00_meta/patterns/<p>.md` (`enforce:true`) in the vault | run the engine → renders + commits the block into every agent file; never hand-edit the generated block |
| A rule for **one repo** | that repo's root `AGENTS.md` (authored section) | the file itself; self-contained, no engine |
| A **CLI-tool quirk** (Claude tool vocab, line cap, Copilot phrasing) | dotfiles `ai/<agent>/` | engine → deploys to the runtime (`~/.claude/`, …) |
| An **autonomous agent's** operating law | vault `80_agents/<agent>/` | read via Hive (it cannot reach dotfiles) |

## Reconciliation with prior decisions

- **Supersedes ADR-016.** The cross-project slice's ownership flips from repo-authored to vault-pattern-authored-and-rendered-in. ADR-016's goal (repo-less agents reach the canonical contract) is preserved — they read the committed render — and self-containment is preserved (committed render). What changes is *who owns the slice* (vault), not *whether a repo can read its contract without the vault* (it still can).
- **Not a zigzag vs ADR-018.** ADR-018 de-vaulted *task state* (collaboration plane → forge). This ADR vaults the *cross-project behavioural slice* (decision/methodology plane → vault). Different planes: reducing vault coupling for collaboration state and increasing it for decision state are consistent, not contradictory.
- **Extends ADR-013** (manifest + generate-and-commit) and **ADR-012** (copy + drift assertion).
- **Consistent with HARNESS-032 / #499** (`00_meta` agnosticism): agent-specific homes are `dotfiles ai/<agent>` (CLI overlays) and `80_agents/<agent>` (autonomous agents). `00_meta` stays agnostic — no `00_meta/ai/<agent>` carve-out, so no guard exception is created.

## Consequences

**Positive**

- One SSOT per datum (reuse); the manifest is the legend (organization + maintainability); the #156 drift class dies for **all** instruction surfaces (automation); the committed render keeps repos self-contained, CI-verifiable, and share-ready.

**Negative / debt**

- **Two drift axes.** Existing repo↔deploy drift (`checkDeployDrift`) plus a NEW vault-SSOT ↔ render-commit axis. The new axis needs its own ADR-012-style assertion that runs on the *committed render* (the vault is not present in the repo CI).
- **Depends on vault sync, which has open debt** (`knowledge#120` — divergent `MEMORY.md`, missing junction). Putting the agent-config plane on an unreliable sync substrate is a sequencing risk.

## Implementation gate

Do **not** implement until `knowledge#120` (reliable cross-machine vault sync) is closed. Then implement via the `HARNESS-001` deploy-engine epic PR sequence:

1. Extend the manifest to **every** instruction surface (`agy`, `copilot`, `hermes`) with a declared `kind`.
2. Promote the cross-project behavioural slice to `enforce:true` vault patterns (extend the existing block-enforced mechanism — 4 enforced rules today).
3. Add the vault↔render-commit drift assertion (runs on the committed render).

## References

- ADR-009 (AGENTS.md cross-agent SSOT), ADR-012 (copy + drift assertion), ADR-013 (manifest + generate-and-commit deploy engine), ADR-016 (deploy canonical AGENTS.md to vault — superseded for slice directionality), ADR-018 (de-vault task placement).
- `pattern-knowledge-placement` (directionality invariant), HARNESS-032 / `knowledge#499` (`00_meta` agnosticism), `knowledge#120` (vault sync — implementation gate), `HARNESS-001` epic (deploy engine).
- Architecture session 2026-06-21 (this decision) + a parallel dotfiles session that surfaced the "committed render is load-bearing" framing correction and the manifest-coverage gap.
