---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - CLI-001-dot-scaffold

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] `go build` + `go test` green (Linux local) -> `ok github.com/mlorentedev/dotfiles/cli/cmd/dot 0.003s` (`TestRootCmd` table-driven, 4 cases + `TestVersionDefault`); Windows leg -> CI `test (windows-latest)` job (pending first CI run)
- [x] `dot --help` exit 0 + usage; `dot version` prints version -> local smoke: help prints "Usage:/Available Commands:", `dot version dev`; unknown subcommand `bogus` exits 1 with usage hint
- [ ] `golangci-lint` passes in CI -> local binary (v1.64.8, built with go1.24) cannot analyze a go 1.26 module; `gofmt -l` clean + `go vet` OK locally; full lint delegated to CI `lint` job (golangci-lint-action@v8)
- [x] goreleaser snapshot + release pipeline -> exercised locally with goreleaser OSS binary: throwaway tag `v0.0.1` produced 6 clean artifacts (`dot_0.0.1_{linux,darwin,windows}_{amd64,arm64}`), checksums.txt, and ldflags version injection verified (`./dist/dot_linux_amd64_v1/dot version` -> `dot version 0.0.1`); tag deleted, dist/ removed
- [ ] Path filter verified on an unrelated change -> verify after PR opens (cli jobs must NOT run on a docs-only PR)
- [x] DX-002 archived -> `git mv` to `specs/archive/_abandoned/DX-002-dot-umbrella-command/`, frontmatter `status: abandoned`, supersession note pointing to ADR-020 + this spec
- [x] `cli/README.md` -> build/test/lint/release one-liners present
- [ ] Manual QA pass -> Linux: binary exercised locally (help/version/bogus/release binary). Windows: review CI `test (windows-latest)` smoke output on the PR; file issues for findings

## Test status

- Test suite: `cd cli && go test ./...` -> `ok github.com/mlorentedev/dotfiles/cli/cmd/dot` (5 subtests, 0 failures)
- Manual smoke test: `go run ./cmd/dot --help` (exit 0, usage shown), `go run ./cmd/dot version` (`dot version dev`), `go run ./cmd/dot bogus` (exit 1, "unknown command ... Run 'dot --help' for usage."), released binary prints injected version
- No regressions in existing test suite: yes — `cli/` is additive; no shell script or bats/Pester test touched

## Decisions made during implementation

- **Plain `v*` release tags, not `cli/vX.Y.Z`**: `monorepo.tag_prefix` is GoReleaser Pro-only (verified empirically — OSS derives version `cli/v0.0.1` verbatim and the slash corrupts artifact paths). The repo has zero tags and the CLI is its only released artifact. Revisit if the repo ever needs releases of its own. (User-approved.)
- **Viper deferred**: scaffold ships Cobra only; config dependency added when the first config-consuming subcommand lands (Decision Hierarchy: stdlib > libs > new deps).
- **No golangci-lint config file**: defaults work for both local v1 and CI v2; config added when a real need appears.
- **Local goreleaser testing via downloaded release binary** (`/tmp/goreleaser`), not `go install`: faster, no toolchain pollution.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? yes — "OSS/Pro feature splits invalidate trained memory: goreleaser monorepo.tag_prefix is Pro-only; verify paywalled features empirically before designing around them"
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — tag-scheme decision recorded here + in proposal R2; ADR-020 already covers the architecture
- [ ] New pattern candidate for `00_meta/patterns/`? no — single-repo concern so far

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-001-dot-scaffold/` -> `specs/archive/CLI-001-dot-scaffold/`
- [ ] Backlog entry: close issue #335 (bitácora auto-moves to Done)
- [ ] Promotions above executed (if any)
