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
Alias to `scripts/shell-profile.sh`. Diagnostic tool — use when interactive shell startup feels >300ms (Linux baseline ~100-150ms with current `.zshrc` + plugins).

### Running knowledge crystallization

```bash
# Weekly: read-only audit — what needs attention?
/insights

# When insights shows unvaulted observations or stale MEMORY.md:
/crystallize

# Automated MEMORY.md date maintenance (post-sprint or in CI):
./scripts/knowledge-crystallize.sh

# All projects at once (auto-discovers from ~/.claude/projects/):
./scripts/knowledge-crystallize.sh --all

# For a specific project:
./scripts/knowledge-crystallize.sh ~/Projects/kubelab
```

See runbook: `docs/runbooks/guide-knowledge-distillation.md`

### Using opencode (AI coding agent — primary daily after PR2)
```bash
oc                    # TUI launcher (opencode Go subscription, $10/mo fixed)
qq "tu pregunta"      # one-shot quick-question via opencode-go/qwen3.6-plus (bash/zsh/pwsh)
```
- Default TUI model: DeepSeek V4 Pro (Go catalog). A/B candidate: Kimi K2.6 — selectable via `/models` in TUI.
- `qq` wrapper pinned to `opencode-go/qwen3.6-plus` (multilingual, fast, never-rate-limited). One-shot: each call is a fresh session. Defined in `.zsh/aliases.zsh`, `.bashrc`, and `powershell/profile.ps1`. Cross-platform name is `qq` (not `??`) because PowerShell 7+ reserves `??` as null-coalescing operator.
- Frontier on-demand: provider `openrouter` (consumes existing `OPENROUTER_API_KEY` $5 credit).
- First-time setup: launch `oc`, run `/connect` → select **OpenCode Go**, paste API key from opencode.ai/zen.
- 3-layer PAYG guardrail: (1) `opencode.jsonc` lists only Go models, (2) Zen workspace cap $0, (3) no payment method for PAYG. Runbook: `docs/runbooks/guide-opencode-go-setup.md`.
- Config: `ai/opencode/opencode.jsonc` → `~/.config/opencode/opencode.jsonc` (deployed by `setup-linux.sh`).
- ⚠ Coexistence constraint: don't run `oc` and `claude` in parallel on the same repo until hive MCP adds a lock-file to its auto-commit.

### Modifying setup scripts
1. Edit both `setup-linux.sh` AND `setup-windows.ps1` if the change is cross-platform
2. Test Linux changes by running relevant sections (MCP registration, config deployment)
3. Verify Windows parity: same MCP servers, same configs deployed

## Self-Verification Loop

After modifying ANY shell script:
1. `~/.local/bin/shellcheck <changed-file>`
2. `~/.local/bin/bats tests/*.bats`
3. If new lessons were learned, write them to `docs/lessons.md`.