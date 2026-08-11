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

> The last two rows fail **silently**: they return an empty or single-element result instead of an
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

## Key Files

| File | Role | Notes |
|---|---|---|
| `versions.conf` | Central version manifest | Single source of truth for tool versions |
| `scripts/utils.sh` | Foundation library | Sourced by ALL other scripts |
| `cli/` | `dotf` Go CLI — the primary user-facing tool | `doctor`, `env`, `secrets`, `spec`, `init`, `mem`, `tools`, `vault`. Absorbs `.sh`/`.ps1` twins on contact (ADR-020) |
| `secrets/registry.yaml` | Secret mapping SSOT (ADR-028) | Replaced the retired sensitive/env-mapping.conf |
| `setup-linux.sh` | Linux bootstrap | Installs tools, deploys configs, registers MCP servers |
| `setup-windows.ps1` | Windows bootstrap | PowerShell equivalent (copies instead of symlinks) |
| `ai/claude/CLAUDE.md` | Master Claude instructions | Deployed to `~/.claude/CLAUDE.md` by setup |
| `harness/skills/*/SKILL.md` | Committed skill records (37) | Rendered from the vault by `compile-harness.sh --refresh`; deployed to each agent by `--deploy`. **Generated — edit the vault source, not these** |
| `harness/manifest.json` | Harness engine manifest | Which enforced regions inject where, and which agents receive skills |
| `scripts/compile-harness.sh` | Harness engine | `--refresh` (vault → records), `--deploy` (records → agent dirs), `--check` (offline drift) |
| `scripts/vault.sh` | Vault tooling dispatcher (REFACTOR-005) | `vault {health, maintenance, check-escapes}` — single discoverable entry point; each subcommand still runnable standalone |
| `scripts/vault-health.sh` | Obsidian vault health check | Checks plugin status, pending tasks, etc. Also runnable via `vault health`. |
| `scripts/obs-cli.sh` | Obsidian CLI wrapper (Linux) | --no-sandbox, default vault, GUI check |
| `scripts/obs-cli.ps1` | Obsidian CLI wrapper (Windows) | Default vault, GUI check |
| `scripts/knowledge-crystallize.sh` | AI knowledge maintenance | Updates MEMORY.md dates, checks health, prints crystallization checklist |
| `scripts/check-doc-paths.sh` | Doc-path guard (#916) | Fails when an instruction file names a repo path that no longer exists |
| `.zsh/aliases.zsh` | Shell aliases | Includes `oc` for opencode, `qq` for one-shot questions, `tx`/`txl`/`txa`/`txk` for tmux |
| `ai/opencode/opencode.jsonc` | OpenCode config | Provider `opencode-go` (Go subscription, catalog-restricted) + `openrouter` + MCP mirror. Deployed to `~/.config/opencode/`. |
| `AGENTS.md` (root) | Cross-agent SSOT | Canonical system prompt read by OpenCode (natively) and Claude (via this file's pointer). Per-agent files in `ai/<agent>/` and `.github/` delegate here. |
| `tmux.conf` | tmux configuration (Linux only) | Deployed via symlink to `~/.tmux.conf` by `setup-linux.sh` |

> **Editing this file: a backticked repo path *containing a slash* is a live claim.**
> `scripts/check-doc-paths.sh` fails CI when one does not resolve — that is how
> this file came to name seven files that no longer existed (#916). To mention a
> path that is gone ("the old loader lived at scripts/load-secrets.sh"), write it
> in plain text, not backticks.
>
> The slash qualifier is not a detail: bare filenames are **not** checked, by
> design — resolving them by basename flagged vault patterns and `machine.json`
> on `AGENTS.md`. So a backticked bare name like `env-mapping.conf` passes the
> guard whether or not it exists. Prefer the rooted form when you want the
> guard's protection.

## Secrets System (ADR-028)

Two tiers, and the shape matters: **secrets are never exported into the ambient
shell.** They are injected into one child process on demand. The older model —
a login-time loader that decrypted everything into the environment — is retired,
along with the scripts/load-secrets.sh loader and the sensitive/env-mapping.conf
mapping file.

- **SSOT:** Bitwarden. **DR floor:** [age](https://github.com/FiloSottile/age)-encrypted copies under `sensitive/*.secret.age`, key at `~/.config/age/key.txt`.
- **Mapping SSOT:** `secrets/registry.yaml`.
- **Facade:** `dotf secrets` — `run` (inject into a child process only), `show`,
  `ls`, `verify`, `set`, `render`, `migrate`, `sync`, `backup`.

```bash
dotf secrets ls                     # inventory: ids, plane, exposed vars (no values)
dotf secrets verify                 # resolve everything, report OK/MISSING/FAILED
dotf secrets run -- ./deploy.sh     # the only sanctioned way to hand a secret to a process
```

## Common Workflows

### Adding a new shell script
1. Create in `scripts/`, use `#!/usr/bin/env bash` shebang
2. Source `utils.sh` if you need shared functions
3. Follow the prohibited patterns table above
4. Run `shellcheck` and `bats` before committing

### Adding a new secret (ADR-028)
1. Add its entry to `secrets/registry.yaml` (the mapping SSOT — id, plane, exposed vars)
2. Write the value: `dotf secrets set <id>` (reads stdin or prompts hidden; idempotent)
3. Verify without printing it: `dotf secrets verify`
4. Consume it: `dotf secrets run -- <cmd>` — never by exporting into your shell

Run `dotf secrets <sub> --help` for the exact flags; do not reconstruct the old
`env-mapping.conf` syntax, which no longer exists.

### Updating a tool version
1. Edit `versions.conf` at repo root — change `TOOL_VERSION=X.Y.Z`
2. Verify syntax: `bash -n versions.conf && zsh -n versions.conf`
3. Run `~/.local/bin/bats tests/versions-conf.bats` to validate format
4. The new version propagates automatically to `.zshrc`/`.bashrc` via `${VAR:-fallback}` on next shell reload
5. Run `dotf doctor` to verify the versioned directory exists at `~/Applications/tool-X.Y.Z`

### Adding a new versioned tool
1. Add `NEWTOOL_VERSION=X.Y.Z` to `versions.conf`
2. Add `export NEWTOOL_HOME="$APPS_HOME/newtool-${NEWTOOL_VERSION:-X.Y.Z}"` to both `.zshrc` and `.bashrc`
3. Add `export PATH="$NEWTOOL_HOME/bin:$PATH"` in both RC files (if the tool has binaries)
4. Add the version check to `cli/internal/doctor/` — append an entry to the
   `versionMatches` table for a versioned `$APPS_HOME` dir, or call `matchPin`
   against the live binary for tools installed elsewhere (see
   `cli/internal/doctor/checks_golangci.go` for the latter shape)
5. Run tests: `~/.local/bin/bats tests/versions-conf.bats` and `cd cli && go test ./internal/doctor/`

### Running the health check
```bash
dotf doctor          # Full post-setup verification
dotf doctor --fix    # Repair what is safely repairable (junctions, hooks paths)
# Exit 0 = all pass, Exit 1 = failures detected
```

> The `healthcheck.sh` / `doctor.sh` twins were retired into this command
> (ADR-020 strangler-fig). If something tells you to run them, it is stale.

### Diagnosing shell startup time (when zsh/bash feels slow)
```bash
profile-shell                              # time-only, 5 runs of zsh, min/median/mean/max
profile-shell --shell bash                 # same for bash
profile-shell --detail                     # per-function breakdown via zprof (zsh) or xtrace (bash)
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

See vault runbook: `$VAULT_PATH/10_projects/dotfiles/40-runbooks/guide-knowledge-distillation.md`
(resolve `$VAULT_PATH` via `dotf env path VAULT_PATH` — never hardcode the literal, per ADR-025)

### Using opencode (AI coding agent — primary daily after PR2)
```bash
oc                    # TUI launcher (opencode Go subscription, $10/mo fixed)
qq "tu pregunta"      # one-shot quick-question via opencode-go/qwen3.6-plus (bash/zsh/pwsh)
```
- Default TUI model: DeepSeek V4 Pro (Go catalog). A/B candidate: Kimi K2.6 — selectable via `/models` in TUI.
- `qq` wrapper pinned to `opencode-go/qwen3.6-plus` (multilingual, fast, never-rate-limited). One-shot: each call is a fresh session. Defined in `.zsh/aliases.zsh`, `.bashrc`, and `powershell/profile.ps1`. Cross-platform name is `qq` (not `??`) because PowerShell 7+ reserves `??` as null-coalescing operator.
- Frontier on-demand: provider `openrouter` (consumes existing `OPENROUTER_API_KEY` $5 credit).
- First-time setup: launch `oc`, run `/connect` → select **OpenCode Go**, paste API key from opencode.ai/zen.
- 3-layer PAYG guardrail: (1) `opencode.jsonc` lists only Go models, (2) Zen workspace cap $0, (3) no payment method for PAYG. Runbook: `$VAULT_PATH/10_projects/dotfiles/40-runbooks/guide-opencode-go-setup.md`.
- Config: `ai/opencode/opencode.jsonc` → `~/.config/opencode/opencode.jsonc` (deployed by `setup-linux.sh`).
- ⚠ Coexistence constraint: don't run `oc` and `claude` in parallel on the same repo until hive MCP adds a lock-file to its auto-commit.

### Modifying setup scripts
1. Edit both `setup-linux.sh` AND `setup-windows.ps1` if the change is cross-platform
2. Test Linux changes by running relevant sections (MCP registration, config deployment)
3. Verify Windows parity: same MCP servers, same configs deployed

## Self-Verification Loop

After modifying ANY shell script:
1. Run `~/.local/bin/shellcheck` on the changed file
2. Run `~/.local/bin/bats tests/*.bats` to catch regressions
3. Only then claim the change is complete

After completing a session:
- If new patterns were learned, suggest updating this CLAUDE.md
- If mistakes were corrected, write the lesson to this repo's `docs/lessons.md`.
  **Not the vault** — project lessons are build/operate knowledge and live in
  the repo (Standing Order #2, `pattern-knowledge-placement`). Only
  cross-project insight goes to `00_meta/` in the store.

<claude-mem-context>
# Recent Activity

<!-- This section is auto-generated by claude-mem. Edit content outside the tags. -->

### Feb 10, 2026

| ID | Time | T | Title | Read |
|----|------|---|-------|------|
| #1016 | 6:47 PM | ○ | Project-specific permissions override allowing WebSearch tool | ~284 |

### Feb 16, 2026

| ID | Time | T | Title | Read |
|----|------|---|-------|------|
| #2360 | 5:33 PM | ○ | Claude Permission Settings Configured | ~354 |
</claude-mem-context>