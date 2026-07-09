# AGENTS.md — `cli/` (dotf Go module)

Nearest-file instructions for the Go CLI. The repo-root `AGENTS.md` remains the
SSOT for **behaviour** (Standing Orders, commit/PR rules, SDD); this file adds
only the Go build/test/lint contract for work under `cli/`. For command
semantics see `cli/README.md`.

Module: `github.com/mlorentedev/dotfiles/cli` (nested module, `go 1.26`, cobra).
Entry point: `cmd/dotf`. Logic lives in `internal/<area>/` (doctor, env, mem,
memlink, secrets, spec, tools, review, initrepo).

## Commands (run from `cli/` — this is CI's `working-directory`)

```sh
go build ./...                 # build all packages
go test ./...                  # run all unit tests
go vet ./...                   # vet
gofmt -l .                     # list unformatted files (must be empty)
golangci-lint run              # lint (CI: golangci-lint-action@v8, defaults)
go run ./cmd/dotf --help       # run locally
go run ./cmd/dotf version      # smoke: version
```

CI (`.github/workflows/cli.yml`) runs `go build ./...`, `go test ./...`, a
`--help`/`version` smoke, and `golangci-lint`. Release is goreleaser on a plain
`vX.Y.Z` tag (binaries, not `go install`).

## Conventions

- **Testable seams over globals.** External surfaces (git, exec, filesystem) are
  injected via a `Deps` struct so `Run` is unit-tested with no real side effects
  (see `internal/update`). Add behaviour behind such a seam, not a direct call.
- **Table-driven tests**, one case per branch; assert the stable status tag, not
  prose. Test files (`*_test.go`) do not count toward the SDD spec-gate.
- **Benign-skip contracts**: scheduler-invoked paths return a non-nil error only
  on a real failure; every "nothing to do" branch is a nil-error skip.
- Keep new subcommands in their own `internal/<area>/` package with a thin
  `cmd/` wiring layer; mirror the strangler-fig migration notes in `README.md`.
