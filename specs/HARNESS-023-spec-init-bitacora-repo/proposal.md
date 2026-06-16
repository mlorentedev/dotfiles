---
id: "HARNESS-023-spec-init-bitacora-repo"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-16"
issue: "mlorentedev/dotfiles#392"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-023-spec-init-bitacora-repo

> **Naming**: file lives at `<repo>/specs/HARNESS-023-spec-init-bitacora-repo/proposal.md`. `HARNESS-023-spec-init-bitacora-repo` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #392: HARNESS-023: dotf spec init assumes bitácora == dotfiles repo (wrong gate repo + hardcoded frontmatter prefix) -->

`dotf spec init --issue N` assumed the work-gate issue always lives in the dotfiles repo. The bitácora spans many repos (kubelab, knowledge, …) on one GitHub Project, so a cross-repo gate broke twice: the gate ran `gh issue view N` against the *current* repo's default (checking the wrong, often-closed issue), and the scaffolded frontmatter hardcoded `issue: "dotfiles#N"`. Found scaffolding a kubelab spec gated by `knowledge#104`.

## What

- The gate check AND the proposal frontmatter both use the repo that actually hosts the issue.
- Resolution precedence: `--bitacora-repo owner/repo` flag → `$DOTF_BITACORA_REPO` → the current repo's `git remote origin` slug.
- Frontmatter records the full `owner/repo#N` (e.g. `mlorentedev/knowledge#392`), not the bare/hardcoded `dotfiles#N`.
- A gated init whose repo cannot be resolved errors (pointing at `--bitacora-repo`) instead of fabricating a bogus `#N`.
- The `[INFO] Work-gate OK` line names the repo (`owner/repo#N`), so a wrong-repo gate is visible, not silent.

## Out of scope

- Where the bitácora Project lives or its field schema.
- Any `dotf` command other than `spec init`.
- Auto-discovering which repo hosts an issue from the number alone (no cross-repo search).

## Risks / open questions

- **Default = current-repo origin** (resolved): correct for the common same-repo gate; cross-repo needs the explicit flag/env. Chosen over a fixed `mlorentedev/knowledge` default, which would have broken same-repo gating — swapping one hardcode for another.
- `OriginRepo` shells out to `git`; environments without git or an origin remote must pass the flag/env — the error message names this path.

## Acceptance criteria

- [x] Gate checks the issue in its real repo via `gh issue view --repo <slug>`, not the current repo's default.
- [x] Frontmatter records the full `owner/repo#N` resolved from flag/env/origin, never a hardcoded `dotfiles#`.
- [x] `--bitacora-repo` flag + `DOTF_BITACORA_REPO` env override the default; precedence flag > env > origin.
- [x] A gated init with an unresolvable repo errors pointing at `--bitacora-repo` (no fabricated issue prefix).
- [x] `go test ./cli/...` green; regression guards added in `spec_test.go` + `cmd/spec_test.go`.

## References

- Issue: `mlorentedev/dotfiles#392`
- Convention precedent: kubelab `specs/NOTIFY-001` (full `owner/repo#N` in frontmatter)
- Touched: `cli/internal/spec/spec.go` (Gate/Render/Scaffold), `cli/internal/cmd/spec.go` (resolution + `--bitacora-repo`)
