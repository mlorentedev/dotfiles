# Dotfiles Project

> Personal development environment: shell configs, AI tool integration, secrets management.

## Shell Compatibility (MANDATORY)

All shell scripts MUST work in **both bash and zsh**. Before modifying any `.sh` file:

| Prohibited Pattern | Use Instead | Why |
|---|---|---|
| `echo -e "..."` | `printf '%b' "..."` | zsh doesn't support `echo -e` |
| `&>/dev/null` | `>/dev/null 2>&1` | Not POSIX; fails in some zsh configs |
| `source file` | `. file` | POSIX portable sourcing |
| `declare -g VAR` | `eval "VAR=value"` | `declare -g` is bash-only |
| `declare -a ARR` | `ARR=()` | Unnecessary; both shells handle it |
| `${BASH_SOURCE[0]}` alone | `${BASH_SOURCE[0]:-$0}` | Fallback needed for zsh |
| `((count++))` with `set -e` | `count=$((count + 1))` | Exits with code 1 when count=0 |
| `${!var}` (indirect) | See `utils.sh` zsh branch | bash/zsh have different indirect syntax |
| `for d in path/*/` where `path` may not exist | `bash -c 'shopt -s nullglob; …'`, or test the dir first | **Fails silently.** zsh's default `NOMATCH` makes an unmatched glob abort the *whole* compound command — the loop never runs and prints nothing |
| `set -- $var` / unquoted `$var` to split into fields | read line by line, or `${=var}` in zsh | **Fails silently.** zsh does not word-split unquoted parameters: you get one field containing everything, not N |
| `. file` (no slash) to source from the cwd | `. ./file` | **Fails silently.** A slashless argument to `.` is searched on `$PATH` only; bash also falls back to the cwd, zsh does not. In a `$(...)` the result is an empty string, not an error |

> The last three rows fail **silently**: they return an empty or single-element result instead of an
> error, and empty reads as a finding. Every row above them breaks loudly. Before believing an
> empty result from a shell sweep, re-run it in the other shell — see `docs/lessons.md`,
> *"a shell incompatibility that answers wrongly beats one that fails"*.

## Verification Commands

Two layers, two loops (ADR-020). Run the one you touched — and note the Go
layer is the primary one, so "I ran shellcheck" is not verification of a `cli/`
change.

```bash
# --- Shell layer ---
~/.local/bin/shellcheck scripts/*.sh setup-linux.sh
~/.local/bin/bats tests/*.bats                        # ~1200 tests, bash+zsh
for f in scripts/*.sh setup-linux.sh; do bash -n "$f"; done

# --- Go layer (cli/) ---
cd cli
go build ./... && go vet ./... && go test ./...
golangci-lint run
```

**Use the pinned linter, not whatever is installed.** CI resolves
`GOLANGCI_LINT_VERSION` from `versions.conf`; a local binary on a different
major reports "0 issues" on code CI rejects (BUG-071). `dotf doctor` reports
the drift under *Go lint toolchain*; install the pin with:

```bash
# `. ./versions.conf`, not `. versions.conf` — a slashless argument to the
# source builtin is searched on $PATH only, so bash finds it in the cwd and zsh
# does not. Under zsh the bare form expands to an empty version and installs
# `@v`. Same class as the prohibited-pattern table above.
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(. ./versions.conf; echo "$GOLANGCI_LINT_VERSION")
```

```bash
# --- Post-setup verification (both layers) ---
dotf doctor
```

## Key Infrastructure

| Area | Primary Entrypoint | Description |
|---|---|---|
| **Tooling & CLI** | `cli/` (`dotf`) | Go CLI for doctor, spec, secrets, init, mem, tools |
| **Shell Foundation** | `scripts/utils.sh` | Portable bash/zsh functions and bootstrap |
| **Versions SSOT** | `versions.conf` | Version pins for languages and CLI tools |
| **Secrets SSOT** | `secrets/registry.yaml` | Secret mappings (managed via `dotf secrets`) |
| **Harness & Skills** | `harness/skills/` | Agent skills deployed by `scripts/compile-harness.sh` |

## Core Workflows & Commands

```bash
# Secrets Management (ADR-028)
dotf secrets ls                     # inventory: ids, plane, exposed vars (no values)
dotf secrets verify                 # resolve everything, report OK/MISSING/FAILED
dotf secrets run -- <cmd>           # the only sanctioned way to hand a secret to a process

# Diagnostics & Health
dotf doctor                         # Full post-setup verification (Go + Shell + Doctrines)
dotf doctor --fix                   # Safely repair auto-fixable drift

# Spec-Driven Development & Review
dotf spec init <spec-id> --issue <N> # scaffold a feature spec gated on an open issue
dotf spec review <spec-id>          # launch detached adversarial review in tmux
dotf spec archive <spec-id>         # archive spec upon successful independent review
```

## Self-Verification Loop

After modifying ANY shell script:
1. `~/.local/bin/shellcheck <changed-file>`
2. `~/.local/bin/bats tests/*.bats`
3. If new lessons were learned, write them to `docs/lessons.md`.