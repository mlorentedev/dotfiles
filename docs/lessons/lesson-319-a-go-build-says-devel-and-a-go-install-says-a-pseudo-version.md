---
id: "lesson-319-a-go-build-says-devel-and-a-go-install-says-a-pseudo-version"
type: lesson
status: active
title: "A go build says (devel); a go install says a pseudo-version"
created: "2026-09-30"
owner: manu
tags: [lesson, go, release, versioning, cli]
---

# A go build says (devel); a go install says a pseudo-version

## What happened

`dotf version` printed `dev` for every binary that goreleaser had not stamped. That included `go install github.com/mlorentedev/dotfiles/cli/cmd/dotf@latest`, the install path an outside user reaches for first. That binary carries nothing else that says which code it is, so a bug report from it could not name a version.

The obvious fix is to read `debug.ReadBuildInfo().Main.Version` whenever the ldflag is absent. That fix rests on an assumption about what Go writes there, so the assumption was measured on Go 1.26.0 first:

| Build | `Main.Version` | `vcs.*` settings |
|---|---|---|
| `go build ./cmd/dotf` in the main checkout | `(devel)` | present (`vcs.revision`, `vcs.modified`) |
| `go build ./cmd/dotf` in a linked worktree | `(devel)` | **absent**, even with `-buildvcs=true` |
| `go install …/cli/cmd/dotf@latest` | `v0.0.0-20260930012845-db2b904794b4` | absent |

Two findings changed the design:

- **A checkout build does not stamp a pseudo-version here.** Our release tags are `vX.Y.Z` at the repo root, but the module lives in `cli/`, where Go looks for `cli/vX.Y.Z` tags. It finds none. That is also why `go install …@v0.61.0` does not resolve and `@latest` falls back to a pseudo-version.
- **A linked worktree gets no VCS stamp at all.** A rule of "`vcs.*` present means source build" would miss every worktree build. So `(devel)` has to count as a source build on its own.

## Rule

- Report the build-info version only when the ldflag is absent, `Main.Version` is neither empty nor `(devel)`, and no `vcs.*` setting is present. Every other case keeps `dev`.
- `dev` is a contract, not a placeholder. `install-dotf.{sh,ps1}` decline to replace a source build on it, and doctor skips the pin check. Widening what prints a version therefore changes installer behaviour, which is why `tests/install-dotf.bats` pins how a pseudo-version parses (`0.0.0`: neither `dev` nor the pin, so the installer converges).
- Measure build info on the toolchain in use before designing on it. What a build stamps depends on the Go version, the tag layout and the worktree type.

## References

- `cli/cmd/dotf/main.go` `resolveVersion`, and its table test in `main_test.go`
- EPIC #1843, row E4
