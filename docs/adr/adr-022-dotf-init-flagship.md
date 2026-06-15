---
id: adr-022-dotf-init-flagship
type: adr
status: accepted
date: 2026-06-14
parent: adr-021-cli-orchestration-roadmap
issue: "dotfiles#384"   # CLI-014 — epic #131; closes #248, #299
related: [adr-021-cli-orchestration-roadmap, adr-020-tooling-cli-go-convergence, adr-012-deploy-strategy-copy-with-drift-assertion, adr-018-de-vault-task-placement]
supersedes: "init-repo-standards.{sh,ps1} (dropped, not ported)"
tags: [cli, dotf, init, scaffolding, strangler-fig, cross-os, knowledge-placement]
---

# ADR-022 — `dotf init` flagship: the repo scaffolder

## Status

Accepted 2026-06-14.

## Context

ADR-021 (the CLI orchestration roadmap) names **`dotf init` as the flagship** port — step 2, after `dotf doctor`. It is the most ambitious noun because it does not *translate* a script; it *distils* everything learned cross-project into one command: "scaffold a new repo fully-practiced from line 1."

`dotf init` absorbs four shell twin pairs (816 LOC `.sh` + ~840 LOC `.ps1`):

| Twin | LOC (sh) | Does |
|---|---|---|
| `init-project.sh` | 470 | Orchestrator: base dirs, AI-memory inject from `~/.claude`, vault entry (`10_projects/<n>/`), memory symlink, stack init, `git init`, pre-commit (gitleaks), `.gitignore`, then calls the 3 helpers |
| `init-repo-agents.sh` | 112 | Bootstrap `AGENTS.md` + the SDD section (snippet from the vault) |
| `init-repo-standards.sh` | 137 | Mirror the vault pattern-index into `docs/standards.md` (regenerate-between-markers) |
| `init-repo-github-defaults.sh` | 97 | `delete_branch_on_merge=true` via `gh api` (idempotent) |

The current scripts hard-couple to two host assets that a portable Go binary cannot assume exist: the **vault** (`$VAULT_PATH/00_meta/templates/`, `…/patterns/_index.md`) and **`~/.claude`** (CLAUDE.md / AGY.md / skills). Off-machine, the generated `AGENTS.md` even leaks unexpanded `$VAULT_PATH` literals (#248). Shedding those couplings — without losing the vault as the canonical source — is the core problem.

This decision is grounded in a 6-repo empirical audit (Regla del 3, per ADR-015) and resolves a cluster of open issues routed by epic #131 to "dot init project": **#248 (SELF-001, self-contained)** and **#299 (AI-agnostic, AGENTS.md-first)**, plus the standalone half of **#275 (HARNESS-013, bootstrap AGENTS.md across the fleet)**.

## Evidence — 6-repo practice audit (Regla del 3)

| Repo | AGENTS | CLAUDE | docs/ | specs/ | pre-commit | CI | docs/adr | docs/standards |
|---|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
| dotfiles | Y | Y | Y | Y | Y | Y | Y | . |
| knowledge | Y | Y | (vault) | Y | Y | Y | n/a | . |
| kubelab | . | Y | Y | Y | Y | Y | Y | . |
| hive | . | Y | Y | Y | . | Y | Y | . |
| iris | Y | . | Y | Y | Y | Y | Y | . |
| pollex | . | Y | Y | Y | . | Y | Y | . |

**Divergence log:**
- **Universal (bake):** `specs/` (6/6), CI (6/6), `docs/adr/` (5/5 code repos), `docs/` placement, AGENTS-or-CLAUDE (6/6), pre-commit gitleaks (4/6 — the 2 gaps are drift, not intent; the "agents paste secrets" threat model is universal).
- **Adopted 0/6: `docs/standards.md`.** No repo keeps the artefact `init-repo-standards.sh` generates. The pattern-index mirror is the least-adopted output — **dropped**, not ported.
- **AGENTS vs CLAUDE is mixed → converge on AGENTS.md-first** (#299); CLAUDE.md becomes a thin pointer.

## Constraints (evaluated against)

| # | Constraint | Origin |
|---|---|---|
| C1 | Self-contained: works on a clean machine with no vault and no `~/.claude` (embedded templates) | ADR-021 north star + #248 |
| C2 | Vault-drift guard: embedded templates are drift-tested against the vault SSOT | `cli/internal/spec/drift_test.go` precedent |
| C3 | Placement model: repo gets `docs/` + `specs/` + `AGENTS.md`; vault keeps only strategy + memory | ADR-018, WORKMODE-001 |
| C4 | Idempotent: re-running any piece is safe (regenerate-between-markers / skip-if-present) | all 4 scripts today |
| C5 | Strangler-fig: the `.sh` + `.ps1` twins are deleted on contact; guard-grep clean; bats → `go test` | ADR-020/021 |
| C6 | AGENTS.md-first: `AGENTS.md` is the agent-instruction SSOT the binary writes | #299 |
| C7 | Host-coupling degrades gracefully: steps needing network / `gh` / vault auto-skip with a clear message, never fatal | init-project today |

## Decision

**`dotf init` is a Cobra orchestrator with re-runnable subcommands** (Option B — the idiomatic `git`/`kubectl` shape), templates **embedded via `//go:embed` and drift-tested against the vault** (the `dotf spec` precedent).

### Subcommand surface

| Command | Role |
|---|---|
| `dotf init [path]` | **Default orchestrator** — full scaffold. Runs every step below; host-coupled steps auto-skip (C7). |
| `dotf init agents [--repo P] [--force]` | Re-runnable: bootstrap / refresh `AGENTS.md` + SDD section. Standalone target for **HARNESS-013** (fleet-wide AGENTS.md backfill). |
| `dotf init github [--repo o/n] [--dry-run]` | Re-runnable: apply GitHub repo defaults (`delete_branch_on_merge`). |

Parent flags: `--stack {go,python,node,none}`, `--skip-vault`, `--skip-github`, `--skip-agents`, `--work-sdk <family> <component>` (vault-only mode, retained for parity — flagged to migrate to `dotf vault`).

### What `dotf init` bakes (the practice stack)

1. **Repo structure** — `src/ tests/ scripts/ specs/ docs/{adr,runbooks,troubleshooting} .claude/` + `docs/lessons.md` stub (placement model, C3).
2. **`AGENTS.md` + SDD section** — AGENTS.md-first (C6), from embedded templates with **no `$VAULT_PATH` literals** (fixes #248); CLAUDE.md written as a thin pointer to AGENTS.md.
3. **Guardrail CI** — `.github/workflows/` with the spec-gate job (≥50 LOC production diff requires an active `specs/<id>/`) and the incident→guard convention documented in AGENTS.md.
4. **pre-commit** — `.pre-commit-config.yaml` (gitleaks), `pre-commit install` when available.
5. **`.gitignore`** + **stack init** (poetry/uv | `go mod` + Makefile | npm/tsc) + **`git init`**.
6. **env-contract** — scaffold the structural contract in the **format `dotf doctor` consumes today (`env-contract.json`)**. The JSON→YAML migration is **#227's** concern, not init's (resolves the ADR-021 "env-contract YAML" wording).
7. **Vault entry (auto-skip)** — `10_projects/<name>/{00-context,10-roadmap,11-tasks,memory/MEMORY}` + memory symlink, **when a vault is present**; `--skip-vault` or vault-absent → skip with `[WARN]` (C1/C7, closes the #248 vault-guard).
8. **GitHub defaults (auto-skip)** — applied when an `origin` remote + authenticated `gh` exist.

### Dropped / out of scope

- **`docs/standards.md` / `init-repo-standards`** — dropped (0/6 adoption). The `.sh` + `.ps1` are deleted with the rest of the twins; the vault pattern catalog stays discoverable via AGENTS.md's standards pointer, not a per-repo mirror.
- **Bitácora board creation** — *not* auto-created. The bitácora is **one shared GitHub Project across all projects** (ADR-018), not a per-repo board. `dotf init` wires the *issue-driven SDD* (AGENTS.md + spec-gate referencing issues), not a new board.
- **AI-memory injection from `~/.claude`** (CLAUDE.md/AGY.md/skills copy) — superseded by embedded AGENTS.md-first templates; the binary is the source, not `~/.claude`.

## Build sequence (TDD-ordered, one spec / PR)

1. `cli/internal/cmd/init.go` + `cli/internal/initrepo/` package; wire `newInitCmd()` into `root.go`. Embed templates under `cli/internal/initrepo/templates/` + **drift-test vs vault** (mirror `spec/drift_test.go`).
2. `dotf init agents` — port `init-repo-agents.sh` (AGENTS.md + SDD, idempotent, no `$VAULT_PATH` leak). Migrate its bats to `go test`. Delete the `.sh`/`.ps1` twin.
3. `dotf init github` — port `init-repo-github-defaults.sh` (`gh api`, dry-run, auto-skip without remote). Delete the twin.
4. `dotf init` orchestrator — structure + gitignore + pre-commit + stack + git + CI scaffold + env-contract + the AGENTS/github steps + vault entry (auto-skip). Port `init-project.sh`; delete the twin + `init-repo-standards.{sh,ps1}`.
5. Guard-grep `init-(project|repo)` returns only provenance (CHANGELOG / ADRs / `specs/`). Repoint `setup-linux.sh` / docs / AGENTS.md to `dotf init`.
6. Close #248, #299; tick the HARNESS-013 standalone-AGENTS half.

## Consequences

### Positive
- One command scaffolds a fully-practiced repo on **any** machine (no vault, no `~/.claude` required) — closes the #248 self-containment defect at the root.
- Re-runnable `dotf init agents` gives the fleet-wide AGENTS.md backfill (HARNESS-013) for free.
- Four twin pairs (~1,650 LOC dual-maintained) collapse to one Go command + one `go test` suite; `docs/standards` drift surface eliminated by deletion.
- AGENTS.md-first removes the CLAUDE.md/AGENTS.md divergence and the `$VAULT_PATH` leak.

### Negative
- Largest port so far (orchestrator + 2 subcommands + embedded templates + drift-test). Highest behavioural-parity surface of any noun to date.
- Embedded templates add a maintenance edge: the drift-test must stay green as the vault evolves (mitigated — same mechanism already proven for `dotf spec`).

### Neutral
- `--work-sdk` (vault-only) rides along for parity; it is really a `dotf vault` concern and is flagged to move there later.
- Windows `.ps1` twins are deleted only once a Windows `dotf` install path exists (same gating as CLI-012 / #380 item 1); until then they stay as divergent orphans tracked in the WIN queue.

## References

- Roadmap: `docs/adr/adr-021-cli-orchestration-roadmap.md` (step 2, flagship)
- Boundary: `docs/adr/adr-020-tooling-cli-go-convergence.md` (Go owns logic; shell owns bootstrap)
- Precedent: `cli/internal/spec/spec.go` + `drift_test.go` (embed + vault-drift guard)
- Closes: #248 (SELF-001 self-contained), #299 (AI-agnostic / AGENTS.md-first); ticks #275 (HARNESS-013 standalone AGENTS)
- Adjacent (not this ADR): #227 (env-contract → config, the JSON→YAML migration), #249 (init-spec → `dotf spec`, done)
- Epic: #131
