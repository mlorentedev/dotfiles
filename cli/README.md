# dot — dotfiles tooling CLI

Single cross-platform entry point for the repo tooling ([ADR-020](../docs/adr/adr-020-tooling-cli-go-convergence.md)).
Shell script twins under `scripts/` converge here one subcommand at a time (strangler-fig).

Nested Go module: `github.com/mlorentedev/dotfiles/cli`. Release tags are plain
`vX.Y.Z` — the CLI is the repo's only released artifact, and goreleaser OSS has
no monorepo tag-prefix support (Pro-only). Distribution is by release binaries,
not `go install`.

## Build

```sh
go build ./...
```

## Test

```sh
go test ./...
```

## Lint

```sh
golangci-lint run ./...
```

## Release

```sh
git tag vX.Y.Z && git push origin vX.Y.Z   # CI runs goreleaser release
```

Local snapshot (no tag, no publish):

```sh
goreleaser build --snapshot --clean
```
