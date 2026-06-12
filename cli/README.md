# dot — dotfiles tooling CLI

Single cross-platform entry point for the repo tooling ([ADR-020](../docs/adr/adr-020-tooling-cli-go-convergence.md)).
Shell script twins under `scripts/` converge here one subcommand at a time (strangler-fig).

Nested Go module: `github.com/mlorentedev/dotfiles/cli`. Release tags are plain
`vX.Y.Z` — the CLI is the repo's only released artifact, and goreleaser OSS has
no monorepo tag-prefix support (Pro-only). Distribution is by release binaries,
not `go install`.

## Commands

### `dot review` — cross-model code review

Reads a unified diff from stdin and asks a non-Claude model for a decorrelated
second-opinion review (markdown on stdout):

```sh
git diff main...HEAD | dot review                        # NaN, deepseek-v4-flash
git diff main...HEAD | dot review --provider openrouter  # OpenRouter, deepseek/deepseek-chat
```

| Provider | Required env | Default model |
|---|---|---|
| `nan` (default) | `NAN_BASE_URL`, `NAN_API_KEY` | `deepseek-v4-flash` |
| `openrouter` | `OPENROUTER_API_KEY` | `deepseek/deepseek-chat` |

Flags: `--model` (override), `--max-bytes` (fail instead of silently truncating,
default 200000), `--timeout` (default 120s). Exit 0 = review produced; exit 1 on
empty stdin, missing env, oversized diff, HTTP error or timeout.

**Privacy**: the diff is sent to the selected third-party API. Think before
piping diffs from repositories you do not own.

**Known limitation**: NaN's gateway can drop long non-streaming responses
(observed with a ~12KB diff); keep diffs focused or use `--provider openrouter`
for large ones. Streaming is a deliberate non-goal until this hurts enough
(spec CLI-003, out of scope).

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
