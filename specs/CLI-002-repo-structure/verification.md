---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - CLI-002-repo-structure

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Layout applied -> `cli/cmd/dot/` holds only `main.go` + `main_test.go`; wiring in `internal/cmd/` (root.go, review.go + tests), domain in `internal/review/`; `go test ./...` green in all three packages with asserts untouched (only package decls + `newRootCmd()` -> `New("dev")` call sites changed)
- [x] Release intact -> `.goreleaser.yaml` untouched (`main: ./cmd/dot`, `-X main.version`); `var version` verified still in `cmd/dot/main.go` (features f3); snapshot job re-verified by CI on the PR
- [x] Normative doc with drift guard -> `docs/architecture.md` + README link; `tests/architecture-md.bats` 5/5 green (and red-confirmed 5/5 before the doc existed — true TDD red->green)
- [x] Blast radius confined -> `git diff --name-only origin/main...HEAD` shows only `cli/`, `docs/`, `README.md`, `tests/`, `specs/` paths (features f5)

## Test status

- Go suite: `cd cli && gofmt -l . && go vet ./... && go test ./... && go build ./...` -> all green (`ok cmd/dot`, `ok internal/cmd`; `internal/review` covered end-to-end through the command tests)
- Drift guard: `bats tests/architecture-md.bats` -> 5/5
- Full bats suite: see PR CI + local run recorded below
- Manual smoke: `go run ./cmd/dot version` -> `dot version dev`; `--help` lists both subcommands
- No regressions: review/root test asserts byte-identical to pre-move

## Decisions made during implementation

- **`var version` stays in `cmd/dot/main.go`** and is injected via `cmd.New(version)`: goreleaser's `-X main.version` ldflags would break *silently* if the var moved to `internal/cmd` (green build, version stuck at "dev"). Release config untouched as a result.
- **`internal/review` has no own test files**: the 12 review subtests exercise the domain end-to-end through the Cobra command (httptest mock). Splitting unit tests per package would duplicate coverage with zero new signal; revisit when the domain grows beyond one consumer.
- **`TestVersionDefault` moved to `cmd/dot/main_test.go`**, next to the var it asserts.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? no — the ldflags gotcha is recorded in R2, `docs/architecture.md` and `main.go`'s comment; it is now structurally guarded, not just remembered
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — ADR-020 already governs; this materializes its §6 home decision
- [ ] New pattern candidate for `00_meta/patterns/`? no — single-repo concern; the layout itself IS the community pattern

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-002-repo-structure/` -> `specs/archive/CLI-002-repo-structure/`
- [ ] Backlog entry: close issue #336 (bitácora auto-moves to Done)
- [ ] Promotions above executed (if any)
