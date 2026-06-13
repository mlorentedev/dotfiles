---
id: "CLI-010-rename-dot-to-dotf"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-13"
issue: "dotfiles#367"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-010-rename-dot-to-dotf

> **Naming**: file lives at `<repo>/specs/CLI-010-rename-dot-to-dotf/proposal.md`. `CLI-010-rename-dot-to-dotf` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #367: CLI-010: Rename the `dot` CLI to `dotf` (avoid Graphviz collision) -->

The ADR-020 convergence CLI is named `dot`, which collides with **Graphviz** (`graphviz` 14.1.2 ships `/usr/bin/dot`). `install-dot.sh` deploys our binary to `~/.local/bin/dot`, and that dir precedes `/usr/bin` on `PATH`, so once `setup` runs our CLI **shadows** Graphviz — any tool that shells out to a bare `dot` (plantuml, doxygen `HAVE_DOT`, render scripts) would silently hit our CLI and fail. ADR-020 modelled the name on `gh`/`kubectl`/`chezmoi` but never weighed this collision; the decision was made without that information. Rename the CLI `dot` → `dotf` now, while the cost is minimal (v0.1.0 just published, binary not yet deployed on any box, no external consumers).

## What

The user-facing tooling binary is `dotf` everywhere — built from `cli/cmd/dotf`, released as `dotf` artifacts, installed to `~/.local/bin/dotf`, invoked as `dotf spec init`, `dotf review`, etc. The Graphviz `dot` is no longer shadowed. After this PR there is exactly one CLI name (`dotf`) across the binary, the release pipeline, the bootstrap (install + healthcheck + setup), and all live docs; ADR-020 carries an amendment recording the collision and the rename.

## Out of scope

- **No CLI behaviour change** — only the binary/command name changes. `spec`/`review` logic is untouched.
- **CLI-005 #339 (retire shell twins)** — lands after this, repointing to `dotf spec`. Reordered, not folded in.
- **Issue/spec titles of not-yet-implemented downstream work** that name `dot` (#340 CLI-006 per-agent adapters, #355 AI-026 pi package) — updated when those are picked up, not here.
- **Cutting the v0.2.0 release tag** — a post-merge operator step (see Risks), not part of this PR's diff.
- **`setup-windows.ps1` install of the CLI** — already a Windows-empirical follow-up; the `.ps1` healthcheck string updates here, but Windows install wiring is unchanged.
- **Historical artifacts** — `CHANGELOG.md`, `docs/adr/audit-*`, archived specs, and the v0.1.0 release stay as-is (provenance, not live references).

## Risks / open questions

- **Release sequencing (resolved, must be honored).** The published v0.1.0 release only carries `dot` artifacts. After merge, `install-dot` looks for `dotf`, which exists in no release until a new tag is cut. Required order: **merge → tag v0.2.0 → CI `release` job builds `dotf` artifacts → install-dot works → setup installs `dotf`**. Until v0.2.0 is cut, a fresh `setup` cannot install the CLI. Acceptable for a solo dotfiles repo; documented in `verification.md` and the issue.
- **CLI-005 gate shifts.** CLI-005's "dot installs on PATH" gate becomes "v0.2.0 with `dotf` installed". CLI-005 rebases onto this and repoints to `dotf spec`.
- **Local smoke independent of release.** `go build ./cmd/dotf` produces the binary directly; the rename is verifiable without waiting for v0.2.0.
- **Latent, not active.** `~/.local/bin/dot` is not deployed on the dev box yet, so nothing breaks at merge time; the rename is the clean window before it activates.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `cli/cmd/dotf/` exists, `cli/cmd/dot/` does not; `go build -o dotf ./cmd/dotf` succeeds and `go test ./...` is green.
- [ ] `cli/.goreleaser.yaml` `project_name` / `builds.id` / `binary` are `dotf`; `.github/workflows/cli.yml` references `dotf`.
- [ ] `install-dot.sh` fetches the `dotf` artifact and installs `~/.local/bin/dotf`; `healthcheck.sh`/`.ps1` check `dotf`; `setup-linux.sh` and `versions.conf` (`DOTF_VERSION`) reference `dotf`.
- [ ] No live `dot`-as-CLI reference remains in `AGENTS.md`, `README.md`, or `docs/architecture.md` (whose `cmd/dot` mention is drift-guarded by `tests/architecture-md.bats`); all point to `dotf`. (References to the `init-spec`/`archive-spec` *shell twins* in `SKILL.md` / `check-spec-gate.sh` / the architecture-map are CLI-005's repoint, not this PR's.)
- [ ] ADR-020 carries an amendment documenting the Graphviz collision and the `dot`→`dotf` rename.
- [ ] `dotf spec init` / `dotf spec archive` smoke-tested end-to-end; `shellcheck` clean on changed scripts; `install-dot`/`healthcheck`/`agents-md` bats green.

## References

- GitHub issue: `dotfiles#367` (work-gate)
- Epic: `dotfiles#131` ([epic] CLI convergence) — `docs/adr/adr-020-tooling-cli-go-convergence.md`
- Reorders: CLI-005 #339 (retire spec shell twins → repoints to `dotf spec`)
- Predecessors: CLI-007 (`dot spec init`), CLI-008 (`dot spec archive`), CLI-009 #365 (setup installs the CLI)
