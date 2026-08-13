---
id: adr-021-cli-orchestration-roadmap
type: adr
status: accepted
date: 2026-06-14
parent: adr-020-tooling-cli-go-convergence
supersedes: "audit-005-scripts-classification (§5 roadmap only)"
related: [adr-020-tooling-cli-go-convergence, audit-005-scripts-classification, audit-002-cross-os-duplication, dotfiles-architecture-map]
tags: [cli, dotf, orchestration, roadmap, strangler-fig, cross-os]
---

# ADR-021 — CLI orchestration roadmap (executing ADR-020)

## Status

Accepted 2026-06-14.

## Context

ADR-020 set the two-language boundary — **Go owns user-facing logic; shell owns the thin bootstrap (detect OS/arch, fetch binary, PATH) + profile/env** — and a strangler-fig migration ("delete the twin pair on contact"). CLI-005 retired the first pair (`init-spec`/`archive-spec` → `dotf spec`). This ADR is the **execution roadmap** for the rest, with an explicit north star and a single prioritization metric.

AUDIT-005 (2026-05-21) classified `scripts/` (the 9-category inventory this roadmap reuses) but **predates the CLI-as-primary-strategy**. Its §5 roadmap — polish `.ps1` parity, a *shell* `vault-cli`, port gates to `.ps1` — points the opposite way (maintain the twins). That §5 is **superseded here**; AUDIT-005's §1–§4 classification stands.

## North star

**Simplify and orchestrate maximally from a single cross-compiled `dotf` binary.** Where today you run a chain of per-OS `.sh`/`.ps1` scripts, tomorrow you run one `dotf <noun>`. The user's framing: *minimise what is dependent on each different system.*

## Decisions

1. **Port = consolidate, not translate.** Each `dotf` noun absorbs its domain's *accumulated best practice*, not a 1:1 shell translation. **`dotf init` is the flagship**: it scaffolds a new repo fully-practiced from line 1 — bitácora board, `AGENTS.md` + SDD spec scaffolding, guardrail CI (spec-gate + incident→guard), pre-commit hooks, YAML contracts (`env-contract`), the knowledge-placement model. Not "init-project in Go" — the distillate of everything learned cross-project, in one command.
2. **Prioritise by per-system dependence eliminated × step-collapse.** A divergent `.sh`+`.ps1` pair absorbed into one binary is the highest-value move; a singleton logic script gains the other OS for free.
3. **Thin-shim for the irreducible shell.** The pieces that *cannot* live in the binary (Claude hooks; the secrets env-export, which must mutate the parent shell) stay as 2-line shell shims that call `dotf` — the logic still goes to Go. E.g. `eval "$(dotf secrets env)"`. **Amendment ([ADR-028](adr-028-secrets-two-tier-bitwarden-age.md), accepted):** this shim plan was reversed — no `dotf secrets env` shipped, and none is planned. The no-ambient-secrets decision means secrets never populate the parent shell at all; `dotf secrets run -- <cmd>` (child-process-only injection) is the shipped shape instead.
4. **Bootstrap goes to `dotf setup` — LAST.** End state: only a minimal `curl | bash` bootstrap (IDEAS-005) + `install-dotf` stay shell; `dotf setup` owns the rest. Ported after the logic nouns (highest risk last).
5. **First port: `dotf doctor`.** Runs on every setup, eliminates 3 divergent pairs, immediate validation surface, medium risk.

## The `dotf` noun surface (target)

| `dotf <noun>` | Absorbs (AUDIT-005 category) | Pairs killed | sh-only logic |
|---|---|---|---|
| `spec` *(in progress)* | SDD/CI: init-spec ✓, archive-spec ✓ + the gates | 2 ✓ | check-spec-gate, check-backlog-*, check-md-escapes |
| `doctor` | Diagnostics: healthcheck, doctor, diff-check, vault-health | 3 | vault-health |
| `init` *(flagship)* | Repo-init: init-project + init-repo-{agents,standards,github-defaults} | 4 | — |
| `vault` | Knowledge: knowledge-crystallize, obs-cli, vault-maintenance-weekly | 3 | vault-health, check-md-escapes |
| `secrets` *(hybrid)* | Secrets: load-secrets (decrypt/map), github-secrets-manager, age-* | 1 (worst) | github-secrets-manager, age-* |
| `sync` / `update` | Lifecycle: dotfiles-sync, dotfiles-selfupdate | 2 | — |
| `mem` | Claude: claude-mem-heal, claude-session-start (hook → shim) | 2 | — |
| `harness` | compile-harness (changelog-gen retired → release-please/CLI-011; skills-to-opencode retired → `harness/skills` `targets[]` render) | — | compile-harness |

## Prioritised sequence (highest leverage first)

1. **`dotf doctor`** — healthcheck (448/520), doctor (218/233), diff-check (116/157 DRIFT) + vault-health. ~1,500 LOC dual-maintenance → one; fixes diff-check drift by deletion.
2. **`dotf init`** *(flagship)* — init-project (460/576) + init-repo ×3. 4 pairs; already shell-orchestrated (REFACTOR-004) → clean Go command tree; bakes in the full practice stack.
3. **`dotf vault`** — knowledge-crystallize, obs-cli (DRIFT), vault-maintenance-weekly + vault-health. Replaces the *shell* vault-cli AUDIT-005 §5 proposed.
4. **`dotf secrets`** — load-secrets (1058/405, the worst pair, 0.24). Kills the BUG-006 critical divergence. Shipped shape supersedes the shim plan above — see the ADR-028 amendment on decision 3.
5. **`dotf spec` gates** — fold check-spec-gate, check-backlog-*, check-md-escapes (sh-only) into Go; cross-platform for free.
6. **`dotf sync` / `mem` / `harness`** — remaining pairs + singletons; hooks become thin shims (changelog retired to release-please, CLI-011).
7. **`dotf setup`** *(last)* — the bootstrap orchestrator moves to Go, leaving only the curl bootstrap + install-dotf in shell.

Each step is its own spec/PR (SDD), guard-grep-verified, with its twins deleted on contact, tracked in the bitácora.

## The irreducible shell floor (stays shell — ADR-020 boundary)

- **`curl | bash` bootstrap** (IDEAS-005) — the entry point; runs before any binary exists.
- **`install-dotf.sh`** — fetch + verify + place the binary (chicken-and-egg).
- **secrets env-**export** shim** — a subprocess cannot export into the parent shell.
- **`shell-profile.sh`** — profiles the shell itself.
- **`windows-defaults.ps1`, `profile-heal.ps1`** — Windows OS-config.
- **RC glue** (`.zshrc`/`.bashrc`) and **`utils.sh`** (shrinks as consumers leave, dies with the last shell logic script).

## End state

`.ps1` files **21 → ~3** · cross-OS pairs **~18 → ~2** · `dotf` with **~8 nouns** owning all logic · `scripts/` holds only the thin bootstrap floor. Per-system divergence surface drops ~90%.

## Closing phase — agent-harness re-curation

After the migration, re-curate the **agent-facing surface** so agents are taught the `dotf`-orchestrated workflow, not the retired scripts: `ai/` skills, the cross-agent `AGENTS.md`, and the per-agent configs under `ai/<agent>/`. CLI-005 did a slice (repointed the `spec` + `adversarial-review` SKILLs and AGENTS §389/§406 to `dotf spec`); the full pass is **HARNESS-021 (#374)**, gated on the per-noun ports landing.

## Supersedes (AUDIT-005 §5 only)

AUDIT-005 §5's recommendations were *never filed as GitHub issues* — they were audit-internal labels. The `dotf` approach replaces them by **deleting** the twins rather than maintaining them:

- `load-secrets.ps1` parity (the "BUG-008/009/010" labels) → moot: `dotf secrets` deletes the `.ps1`.
- `check-spec-gate` → `.ps1` ("FEAT-001") → moot: `dotf spec gate` is cross-platform.
- shell `vault-cli` ("REFACTOR-005") → superseded by `dotf vault`.
- `obs-cli.ps1`/`diff-check.ps1` parity polish ("REFACTOR-006") → moot: those `.ps1` are deleted.
- *(Windows shell-profiling, "FEAT-002", is orthogonal — profiling the shell stays shell.)*

## Consequences

- The binary grows; the script surface and cross-OS divergence shrink ~90%.
- Each port carries the incident→guard discipline: twins deleted, references repointed (guard-grep as completeness oracle), tests migrated bats → `go test`.
- `utils.sh` and `test.sh` shrink continuously; the last port retires them.
