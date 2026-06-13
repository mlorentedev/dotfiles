---
id: "CLI-002-repo-structure"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-06-13"
issue: "dotfiles#336"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-002-repo-structure

> **Naming**: file lives at `<repo>/specs/CLI-002-repo-structure/proposal.md`. `CLI-002-repo-structure` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #336: CLI-002: Reorganize repo folder structure for the Go CLI -->

The repo must follow community-standard structure patterns so it can be shared publicly and serve as the owner's central tool across employment, personal projects, and freelancing — everything automatable, agent/OS-agnostic, and one-click. Today's tree does not read like the community-standard repos it competes with (a flat ~29-file `scripts/`, a bolted-on `cli/`), which taxes sharing, onboarding, and every new consumer context. [AGENT-SUGGESTION — accept or remove] Without a declared target tree, each twin migrated under ADR-020 lands ad-hoc and the reorganization happens in a vacuum — the exact failure mode issue #336 exists to prevent.

## What

1. `cli/` follows the community-standard Go CLI layout (go.dev module layout; same shape as `gh`/`chezmoi`): `cmd/dot/main.go` is entrypoint-only, Cobra wiring lives in `internal/cmd/` (root, version, review + their tests), domain logic lives in `internal/review/`. Every future twin port lands as `internal/cmd/<verb>.go` + `internal/<domain>/` — `cmd/dot/` never grows beyond `main.go` (the anti-flat-`scripts/` guard). The module stays at `cli/` with its own `go.mod` (ADR-020); CI path filters and goreleaser keep working.
2. `docs/architecture.md` exists as the normative "where does X live" doc: root directory table, the declared target tree, a pointer to AGENTS.md §Language Boundary (no duplication), and the `scripts/` shrink trajectory (per-script migration map lives in epic #131). `README.md` links it prominently.
3. The root tree itself is otherwise untouched in this PR — moves happen on contact as twins port (strangler-fig), against the declared target.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- **No root or `scripts/` moves** — no bootstrap/tooling split, no dir renames, no twin deletions; those land on contact with each port (`scripts/` only shrinks via epic #131).
- **No new subcommands, no twin ports** — this is a reorganization of existing code (`root`, `version`, `review`); zero new logic.
- **No duplication of the language boundary or the per-script migration map** in `docs/architecture.md` — pointers to AGENTS.md §Language Boundary and epic #131; one SSOT per datum (Standing Order #2).

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- **R1 — `docs/architecture.md` rots** (known bug class in this repo: docs-drift). RESOLVED by design: a bats guard in the same PR (pattern of `agents-md.bats`) asserts the doc exists, README links it, and the top-level dirs declared in its table match the real tree.
- **R2 — build/release paths**: `.goreleaser.yaml` and CI build `./cmd/dot` from `cli/`; the move changes import paths but not the entrypoint. RESOLVED by design: goreleaser injects the version via `-X main.version` — if `version` moved to `internal/cmd` the ldflags would break *silently* (green build, version stuck at "dev"). Therefore `version` stays in `cmd/dot/main.go` and is passed to the root constructor (`cmd.New(version)`); release config untouched.
- **R3 — over-engineering the `internal/cmd` vs `internal/<domain>` split with only 3 commands**: accepted deliberately — with ~18 known incoming ports, declaring the structure now is cheaper than re-reorganizing at the fifth port (scope decision, recorded in What).

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **Layout applied and verifiable:** `cli/cmd/dot/` contains only `main.go`; Cobra wiring lives in `cli/internal/cmd/`, review domain logic in `cli/internal/review/` — `go test ./...` green on the Linux+Windows matrix with no behavior change (review and root tests pass with their asserts untouched).
- [ ] **Release intact:** the goreleaser snapshot job stays green (entrypoint `./cmd/dot` unchanged).
- [ ] **Normative doc with drift guard:** `docs/architecture.md` exists (dir table, target tree, pointers to AGENTS.md §Language Boundary and epic #131), `README.md` links it, and a new `tests/architecture-md.bats` fails if any real top-level dir is missing from the table (R1 guard).
- [ ] **Blast radius confined:** the diff touches only `cli/`, `docs/`, `README.md`, `tests/` and `specs/` — nothing in `scripts/`, setup or configs.

## References

- Work-gate: issue #336 (epic #131 — Go CLI convergence)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (module stays at `cli/`; two-layer boundary)
- AGENTS.md §Language Boundary (declared in CLI-004, PR #354)
- Layout references: go.dev/doc/modules/layout; `gh` CLI (`cmd/` + internal packages); chezmoi (`internal/cmd/`)
- Predecessors: `specs/archive/CLI-001-dot-scaffold/`, `specs/archive/CLI-003-dot-review/`
