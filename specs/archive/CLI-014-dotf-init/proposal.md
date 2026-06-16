---
id: "CLI-014-dotf-init"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-14"
issue: "dotfiles#384"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-014-dotf-init

> **Naming**: file lives at `<repo>/specs/CLI-014-dotf-init/proposal.md`. `CLI-014-dotf-init` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #384: CLI-014: dotf init flagship — consolidate init-project + init-repo-{agents,github} (closes #248, #299) -->

Repo initialization is split across four shell twin pairs (`init-project` + `init-repo-{agents,standards,github-defaults}`, 816 LOC `.sh` + ~840 `.ps1`, dual-maintained and dual-tested). Worse, they hard-couple to two host assets a fresh machine may lack — the vault (`$VAULT_PATH/00_meta/templates/`) and `~/.claude` — so off-machine the generated `AGENTS.md` leaks unexpanded `$VAULT_PATH` literals (#244/#248) and degrades silently. ADR-021 names `dotf init` the **flagship** port: not a translation of `init-project.sh` but the distillate of cross-project practice — one command that scaffolds a fully-practiced repo from line 1, self-contained, on any machine.

## What

A `dotf init` Cobra command (orchestrator + re-runnable subcommands, per ADR-022) that scaffolds a new repo from **templates embedded in the binary** (`//go:embed`, drift-tested against the vault SSOT — the `dotf spec` precedent):

- `dotf init [path]` — full scaffold (default): repo structure + `AGENTS.md`+SDD (AGENTS.md-first, no `$VAULT_PATH` leak) + CLAUDE.md pointer + guardrail CI (spec-gate + incident→guard) + pre-commit (gitleaks) + `.gitignore` + stack init + `git init` + env-contract (`env-contract.json`) + vault entry (auto-skip) + GitHub defaults (auto-skip).
- `dotf init agents [--repo P] [--force]` — re-runnable AGENTS.md+SDD bootstrap (the standalone target for HARNESS-013 fleet backfill).
- `dotf init github [--repo o/n] [--dry-run]` — re-runnable `gh` repo defaults (`delete_branch_on_merge`).

Host-coupled steps (vault entry, GitHub defaults, `pre-commit install`) auto-skip with a clear `[WARN]` when their dependency is absent — never fatal. The `scripts/init-{project,repo-*}.{sh,ps1}` twins + their bats are deleted on contact (Windows `.ps1` gated on a Windows `dotf` install, per #380 item 1).

## Out of scope

- The JSON→YAML migration of `env-contract` — that is **#227** (`dotf` config). `dotf init` scaffolds the format `dotf doctor` consumes today (JSON).
- Auto-creating a bitácora board — the bitácora is **one shared GitHub Project** across all projects (ADR-018), not per-repo. `dotf init` wires issue-driven SDD, not a board.
- `docs/standards.md` / `init-repo-standards` — **dropped** (0/6 adoption in the ADR-022 audit); the twin is deleted, not ported.
- `dotf setup` (the bootstrap orchestrator) — ADR-021 step 7, last and separate.
- Windows `.ps1` deletion until a Windows `dotf` install path exists (#380 item 1 territory).

## Risks / open questions

- **Behavioural-parity surface is the largest of any noun so far** (orchestrator + 2 subcommands). Mitigation: enumerate each baked artefact as a `tasks.md` checklist; golden-test stable outputs.
- **Embedded-template drift.** The drift-test must stay green as the vault evolves. Mitigation: reuse the exact `spec/drift_test.go` mechanism (already proven).
- **`--work-sdk` (vault-only) placement.** A `dotf vault` concern, not repo-init. (Resolved: **removed** on both OSes — not ported, not extracted to a transitional script; restore in `dotf vault`, tracked by #388.)
- **Idempotency across re-runs** (regenerate-between-markers for AGENTS.md, skip-if-present elsewhere) must match the current scripts' guarantees.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] `dotf init` scaffolds a fully-practiced repo on a machine with **no vault and no `~/.claude`**; the generated `AGENTS.md` contains zero unexpanded `$VAULT_PATH` literals (closes #248).
- [x] `dotf init agents` and `dotf init github` are independently re-runnable and idempotent (re-run is a safe no-op / clean regeneration).
- [x] `AGENTS.md` is the agent-instruction SSOT; `CLAUDE.md` is written as a thin pointer (closes #299).
- [x] Embedded templates are drift-tested against the vault SSOT (`go test` fails on divergence).
- [x] Vault entry + GitHub defaults auto-skip with a `[WARN]` when their dependency is absent; exit code stays 0.
- [x] `scripts/init-{project,repo-agents,repo-standards,repo-github-defaults}.sh` + their bats are removed; guard-grep `init-(project|repo)` returns only provenance (CHANGELOG / ADRs / `specs/`).
- [x] `setup-linux.sh` + docs + `AGENTS.md` repoint to `dotf init`.
- [x] `go test ./...` covers the scaffold + drift; `dotf init` smoke-tested end-to-end on a throwaway checkout.

## References

- ADR: `docs/adr/adr-022-dotf-init-flagship.md` (the decision) · `docs/adr/adr-021-cli-orchestration-roadmap.md` (step 2)
- Work-gate: `dotfiles#384` (CLI-014) · closes #248 (SELF-001), #299 (AI-agnostic) · ticks #275 (HARNESS-013) · epic #131
- Precedent: `cli/internal/spec/spec.go` + `cli/internal/spec/drift_test.go` (embed + vault-drift guard)
- Twin sources: `scripts/init-project.sh` (470), `scripts/init-repo-{agents,standards,github-defaults}.sh`
