# dotf

A cross-platform CLI for a dotfiles repository. It checks that a machine is set
up the way the repository says it should be, injects secrets into a single
process without putting them in your shell, scaffolds and gates spec-driven
work, and closes the loop on pull-request reviews.

Release builds exist for Linux, macOS and Windows; CI tests Linux and Windows. It is developed in and for
[mlorentedev/dotfiles](https://github.com/mlorentedev/dotfiles), and most
commands expect that repository's layout.

## Install

From a release (checksum-verified, installs to `~/.local/bin`):

```sh
curl -fsSL https://raw.githubusercontent.com/mlorentedev/dotfiles/main/scripts/install-dotf.sh | bash
```

On Windows, run `scripts/install-dotf.ps1` from a checkout.

With Go:

```sh
go install github.com/mlorentedev/dotfiles/cli/cmd/dotf@latest
```

A `go install` build reports a Go pseudo-version (`dotf version 0.0.0-<date>-<commit>`)
rather than a release number, because release tags are not module-prefixed. Setup
replaces it with the pinned release.

## Commands

| Command | What it does |
|---|---|
| `converge` | Brings this machine to the state the checkout declares; `--plan` shows the changes first |
| `doctor` | Checks this machine's tools, configs and secrets against the repository |
| `secrets` | Resolves secrets and injects them into one child process (`dotf secrets run -- <cmd>`) |
| `spec` | Scaffolds, reviews and archives spec folders for spec-driven development |
| `pr` | Pull-request review loop helpers: `triage-queue` lists unanswered reviewer output; `land` merges PRs, one at a time and in the order given, only when CI, triage and freshness hold on one head, under a per-repository lock |
| `review` | Cross-model code review of a diff read from stdin |
| `worktree` | Git worktree lifecycle and safe garbage collection |
| `hooks` | Installs and dispatches global git hooks |
| `forge` | Branch protection declared in git, checked and applied |
| `init` | Scaffolds a repository with AGENTS.md, specs and guardrail CI |
| `tools` | Installs the tools in `packages.json`; `install --dry-run` shows the plan first |
| `deploy` | Installs agent configs from the checkout to their deployed locations |
| `env` | Resolves per-machine paths (`paths.sh` / `paths.ps1`) |
| `update` | Fast-forwards the repository and re-runs setup (opt-in, run by a scheduler) |
| `agent`, `harness`, `pi`, `orca`, `mem`, `vault`, `search` | Agent harness and knowledge-vault tooling for this repository |
| `version` | Prints the version |

Run `dotf <command> --help` for full usage. Three commands get a section below,
because their flags and failure modes are not obvious from `--help`.

### `dotf converge` — bring the machine to its declared state

Runs an ordered list of reconcilers, each converging one part of the machine
from data in the checkout. Records come first: the harness mirror,
then the agents' instruction files, so no agent runs before its instructions
exist. Then the tools, then every `ai/deploy.json` config that applies to this
machine (`configs-deploy`, the loop behind `dotf deploy`), then the git config.

```sh
dotf converge --plan     # what each reconciler would change; writes nothing
dotf converge            # apply, then prove each reconciler's post-condition
```

- **One code path.** Each reconciler has a plan and an apply on the same path,
  so they cannot disagree.
- **Mandatory probe.** Every reconciler has a post-condition probe. An apply
  that fails its probe fails the run and names the reconciler, and the
  reconcilers after it are reported `not reached`.
- **Platforms.** A reconciler that does not apply to this OS is reported as
  `skipped`, naming the OS, never as passed.
- **Idempotence.** A second run on a converged machine reports `0 changed`.
- **Secrets.** A config whose secrets the store cannot resolve during the run
  (locked or unreachable) keeps its installed file, and the report names it as
  `kept (secrets locked)`; it never installs the placeholder over the value.
- **Report.** Every apply writes a report to
  `$XDG_STATE_HOME/dotfiles/converge/last.json` (default
  `~/.local/state/dotfiles/converge/last.json`). It records the result, the
  error if any, the total changes, and each reconciler's status and detail. A
  plan writes no report.

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
for large ones. Streaming is not supported.

### `dotf doctor` — post-setup diagnostics

Run it after setup and on demand:

```sh
dotf doctor              # full sweep; exit 0 if all checks pass, 1 on any FAIL
dotf doctor --fix        # also print profile lines for missing env defaults + wire safe repaired state
dotf doctor --quick      # env-contract sweep only, fast (used by the session-start hook)
dotf doctor --verbose    # list passing checks too (default summarises them per section)
```

Checks: core tools on PATH, versioned tool dirs, version pins (`versions.conf`, `packages.json`),
key symlinks, environment variables and PATH (`env-contract.json`), optional tools,
vault presence, secrets integrity, tmux, agent harness drift. Advisory `WARN`, `SKIP`
and `INFO` never fail the run. It resolves `DOTFILES_DIR` (default `$HOME/.dotfiles`),
falling back to the git repository root.

## Develop

The module is `github.com/mlorentedev/dotfiles/cli`, nested in the repository.

```sh
go build ./... && go vet ./... && go test ./...
GOOS=windows go vet ./...      # the Windows CI leg compiles the same tree
golangci-lint run              # use the version pinned in versions.conf
```

A plain `go build` prints `dotf version dev`, which the installers treat as a
source build and never replace.

## Release

Releases are cut by release-please and built by goreleaser in CI. The steps and
the deploy order are in [docs/runbooks/release-dotf.md](../docs/runbooks/release-dotf.md).
A local snapshot, with no tag and nothing published:

```sh
goreleaser build --snapshot --clean
```
