# dotf — dotfiles tooling CLI

Single cross-platform entry point for the repo tooling ([ADR-020](../docs/adr/adr-020-tooling-cli-go-convergence.md)).
Shell script twins under `scripts/` converge here one subcommand at a time (strangler-fig).

Nested Go module: `github.com/mlorentedev/dotfiles/cli`. Release tags are plain
`vX.Y.Z` — the CLI is the repo's only released artifact, and goreleaser OSS has
no monorepo tag-prefix support (Pro-only). Distribution is by release binaries,
not `go install`.

## Commands

| Subcommand | What it does |
|---|---|
| `doctor` | Post-setup diagnostics — the consolidated healthcheck + doctor twins (ADR-021) |
| `env` | Per-machine path resolution (`paths.sh` / `paths.ps1`), ADR-025 |
| `init` | Scaffold a fully-practiced repo from embedded templates (ADR-022) |
| `mem` | Cross-agent memory session hooks (session-end / session-start), ADR-014 |
| `review` | Cross-model code review of a diff read from stdin |
| `secrets` | On-demand secrets — inject into a child process, never the shell (ADR-028) |
| `spec` | Spec-driven development scaffolding (ADR-020) |
| `tools` | Declarative cross-OS package catalog (`packages.json`) |
| `update` | Self-deploy: fast-forward the dotfiles repo and re-run setup (opt-in, scheduler-invoked) |
| `vault` | Scaffold knowledge-vault entries |
| `version` | Print the `dotf` version |

Run `dotf <cmd> --help` for any subcommand's full usage. Two get a deep section below
because their flags and failure modes are non-obvious; the rest are self-explanatory
from `--help`.

### `dotf review` — cross-model code review

Reads a unified diff from stdin and asks a non-Claude model for a decorrelated
second-opinion review (markdown on stdout):

```sh
git diff main...HEAD | dotf review                        # NaN, deepseek-v4-flash
git diff main...HEAD | dotf review --provider openrouter  # OpenRouter, deepseek/deepseek-chat
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

### `dotf doctor` — post-setup diagnostics

The Go consolidation of scripts/healthcheck.sh (the 12-section sweep) and
scripts/doctor.sh (the env-contract.json verifier) — both retired — read
natively, no `jq`
(CLI-012, the first port of [ADR-021](../docs/adr/adr-021-cli-orchestration-roadmap.md)).
Run after `setup-linux.sh` and on demand:

```sh
dotf doctor              # full sweep; exit 0 if all checks pass, 1 on any FAIL
dotf doctor --fix        # also print profile lines for missing env defaults + wire safe repaired state
dotf doctor --quick      # env-contract sweep only — fast, no compile-harness gate (SessionStart hook)
dotf doctor --verbose    # list passing checks too (default summarises them per section)
```

Checks: core tools on PATH, versioned tool dirs, version-pin match (`versions.conf`),
key symlinks, environment variables + PATH (`env-contract.json`), optional tools,
vault presence, secrets integrity, tmux, opencode/pi, harness drift, Antigravity.
Advisory `WARN`/`SKIP`/`INFO` never fail the run. It resolves `DOTFILES_DIR`
(default `$HOME/.dotfiles`), falling back to the git repo root.

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
