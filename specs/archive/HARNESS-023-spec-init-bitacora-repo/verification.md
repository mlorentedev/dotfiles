---
tags: [spec, verification, templates]
created: "2026-06-16"
---

# Verification - HARNESS-023-spec-init-bitacora-repo

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof.

- [x] **AC1** gate uses the issue's repo -> `spec.Gate(num, repo)` appends `--repo <slug>`; covered by `cmd/spec_test.go:TestSpecInitBitacoraRepoFlagOverrides` + `TestSpecInitWithOpenIssueSetsFrontmatter`.
- [x] **AC2** full `owner/repo#N` frontmatter -> `spec/spec_test.go:TestRenderSubstitutesAndFixesIssueFrontmatter` asserts `issue: "mlorentedev/knowledge#358"` (full slug, not hardcoded `dotfiles#`); `cmd` test asserts `mlorentedev/dotfiles#358`.
- [x] **AC3** precedence flag > env > origin -> flag: `TestSpecInitBitacoraRepoFlagOverrides`; env: `TestSpecInitWithOpenIssueSetsFrontmatter` (`DOTF_BITACORA_REPO`); origin default: exercised live (this spec scaffolded itself with `issue: "mlorentedev/dotfiles#392"` resolved from origin).
- [x] **AC4** unresolvable repo errors (no fabricated `#N`) -> `TestSpecInitUnresolvableRepoFails` asserts the error names `--bitacora-repo`.
- [x] **AC5** suite green -> `go test ./...` all ok; `gofmt -l` clean; `go vet` clean.

## Test status

- `go test ./...` (from `cli/`): `ok` for `internal/cmd` (0.319s), `internal/spec` (0.008s), `internal/initrepo` (0.284s), `cmd/dotf`, `internal/doctor`. No failures.
- `gofmt -l internal/spec internal/cmd`: empty (clean). `go vet ./...`: clean.
- Manual smoke (built binary): `dotf spec init --help` lists `--bitacora-repo`; `--force-no-gate` scaffolds with `issue: ""` (no fabrication).
- **Dogfood**: this spec was scaffolded by the *fixed* binary gated on #392 — output `[INFO] Work-gate OK: mlorentedev/dotfiles#392 is open` and frontmatter `issue: "mlorentedev/dotfiles#392"` (full slug, resolved from origin), proving both symptoms fixed end-to-end.
- No regressions: existing `spec`/`cmd` tests migrated to the new `Gate`/`Render`/`Scaffold` signatures, all green.

## Decisions made during implementation

- **Default = current-repo origin, not a fixed `mlorentedev/knowledge`.** The issue proposed defaulting to `knowledge`, but that swaps one hardcode for another and breaks the common same-repo gate (a dotfiles spec gated by a dotfiles issue). Origin-resolution is right for same-repo; the flag/env covers cross-repo.
- **Slug resolved in the `cmd` composition layer**, then passed into the pure `spec` package (reusing `initrepo.OriginRepo`/`ValidRepoSlug`) — avoids a `spec`->`initrepo` dependency.
- **Errors instead of fabricating** an `issue:` prefix when the repo is unresolvable while gating — no bogus `#N` reaches the proposal.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "a migration that *broadens* scope (bitácora: dotfiles repo -> multi-repo Project) leaves single-repo assumptions hardcoded in the tools built against the old shape" (sibling of the incomplete-migration class). Decide at archive.
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — a bugfix + flag, no architectural shift.
- [x] New pattern candidate for `00_meta/patterns/`? no — repo-local.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-023-spec-init-bitacora-repo/` -> `specs/archive/HARNESS-023-spec-init-bitacora-repo/`
- [x] Bitácora #392 closed -> `Done` (built-in workflow); fix shipped in PR #393. (Task state lives in the bitácora, not vault `11-tasks.md`, per ADR-018.)
- [x] Promotions above executed (if any) -> lesson promoted to `docs/lessons.md` (this PR).
