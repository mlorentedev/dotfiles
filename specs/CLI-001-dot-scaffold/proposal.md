---
id: "CLI-001-dot-scaffold"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-12"
issue: "dotfiles#335"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-001-dot-scaffold

> **Naming**: file lives at `<repo>/specs/CLI-001-dot-scaffold/proposal.md`. `CLI-001-dot-scaffold` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #335: CLI-001: Scaffold the `dot` Go CLI (module, Cobra root, CI, goreleaser) -->

The goal is to simplify all repo tooling under one CLI verb (`dot`), per ADR-020. Today every behavioural fix to the ~18 `.sh`/`.ps1` twin scripts is written twice and tested twice (bats + Pester) — a recurring tax already paid in PRs #242 and #259. This scaffold is the entry point of the convergence epic (#131): until it lands, `dot review` (#337) and every twin migration are blocked and the dual-maintenance tax keeps accruing.

## What

1. A `cli/` Go module exists in this repo (`github.com/mlorentedev/dotfiles/cli`, own `go.mod`) with a Cobra root command: `dot --help` and `dot version` build and run on Linux and Windows from one codebase. Viper wiring is deferred until the first config-consuming subcommand (Decision Hierarchy: no dependency before a real need).
2. CI runs `go test` (table-driven) + lint on the module for every PR touching `cli/` (path-filtered job).
3. Pushing a plain `vX.Y.Z` tag triggers goreleaser and publishes statically-linked per-OS/arch binaries (linux/macOS/windows) to GitHub Releases. Plain tags, not the `cli/vX.Y.Z` nested-module convention: goreleaser OSS has no monorepo tag-prefix support (Pro-only — verified empirically during implementation), the repo has no other tagged artifact, and distribution is by binaries, not `go install`. Revisit if the repo ever needs releases of its own.
4. `specs/DX-002-dot-umbrella-command/` is archived as superseded by ADR-020 (its subcommand map and GraphViz-collision risk are harvested into this spec).

## Out of scope

- **No twin migration**: no existing `.sh`/`.ps1` script is ported or deleted here — the strangler starts in later PRs.
- **No `dot review`** — that is CLI-003 (#337).
- **No full target-tree definition or `scripts/` shrink plan** — that is CLI-002 (#336). This PR places only the minimal `cli/cmd/dot/main.go` (standard Go layout), pre-conforming to the tree CLI-002 will formalize. The module *location* (own repo vs subtree) IS resolved here, per #335's "open decision to resolve here".
- **No bootstrap/PATH integration** (setup scripts fetching the binary onto PATH) — lands with the first migrated twin, when there is something to run in production.
- **No `AGENTS.md` language-boundary declaration** — that is CLI-004 (#338), explicitly sequenced after CLI-003 proves the pattern.

## Risks / open questions

- **R1 — `dot` name collides with GraphViz `dot`** (harvested from DX-002-R1). Not blocking here (nothing is installed on PATH in this PR), but MUST be resolved before the bootstrap/PATH-integration PR: detect GraphViz at install time, warn and offer an alias.
- **R2 — goreleaser monorepo tag-prefix (`cli/vX.Y.Z`) unproven in this repo.** RESOLVED during implementation, empirically: `monorepo.tag_prefix` is GoReleaser **Pro-only**; OSS derives the version verbatim from a prefixed tag (`cli/v0.0.1`), and the slash corrupts artifact names. Decision: plain `v*` tags (the CLI is the repo's only released artifact; revisit if that changes). CI still runs `goreleaser build --snapshot` on every PR touching `cli/`, so the pipeline stays exercised without waiting for a real tag.
- **R3 — Go toolchain version pinning.** Resolved by design: the `toolchain` directive in `cli/go.mod` is the single source; `actions/setup-go` reads it via `go-version-file`. No second SSOT in `versions.conf` — the Go version belongs to the module, not the environment.
- **R4 — "runs on Windows" acceptance must be empirical.** CI adds a `windows-latest` job running `go test` + `dot --help`, so the criterion is proven on every PR without depending on a physical Windows box.
- **R5 — TDD/CI cannot catch everything.** A manual QA/exploratory pass on Linux and Windows is a standing task for every CLI PR (this one included): run the binary, poke at edges, file GitHub issues for anything off. Tracked in `tasks.md`.

## Acceptance criteria

- [ ] `go build ./...` and `go test ./...` green in CI on Linux **and** Windows (matrix) for the `cli/` module.
- [ ] `dot --help` exits 0 and prints usage; `dot version` prints the version — verified on both OSes in CI.
- [ ] `golangci-lint` passes in CI for `cli/` from day one.
- [ ] `goreleaser build --snapshot` in CI produces statically-linked linux/macOS/windows binaries (amd64 + arm64).
- [ ] CLI jobs run only on PRs touching `cli/**` (path filter verified on an unrelated change).
- [ ] `DX-002-dot-umbrella-command` archived to `specs/archive/_abandoned/` with `status: abandoned`.
- [ ] `cli/README.md` documents build / test / release as one-liners.
- [ ] Manual QA pass on Linux and Windows recorded in `verification.md`; GitHub issues filed for any finding.

## References

- Work-gate: issue #335 (epic #131 — Go CLI convergence)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md`
- Supersedes: `specs/archive/_abandoned/DX-002-dot-umbrella-command/` (shell umbrella draft, pre-ADR-020)
- Prior art: `gopls` nested module in `golang.org/x/tools` (tag prefix convention); chezmoi (single static binary)
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md`
