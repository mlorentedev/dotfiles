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

> The last two rows fail **silently**: they return an empty or single-element result instead of an
> error, and empty reads as a finding. Every row above them breaks loudly. Before believing an
> empty result from a shell sweep, re-run it in the other shell — see `docs/lessons.md`,
> *"a shell incompatibility that answers wrongly beats one that fails"*.

## Verification Commands

Run these before claiming any shell script change is complete:

```bash
# Lint all scripts
~/.local/bin/shellcheck scripts/*.sh setup-linux.sh

# Run full test suite (147 tests, bash+zsh)
~/.local/bin/bats tests/*.bats

# Syntax check
for f in scripts/*.sh setup-linux.sh; do bash -n "$f"; done
```

## Key Files

| File | Role | Notes |
|---|---|---|
| `versions.conf` | Central version manifest | Single source of truth for tool versions |
| `scripts/utils.sh` | Foundation library | Sourced by ALL other scripts |
| `scripts/healthcheck.sh` | Post-setup verification | 6-section tool/version/symlink checks |
| `scripts/load-secrets.sh` | Secrets loader | Sourced in `.zshrc`/`.bashrc` at login |
| `sensitive/env-mapping.conf` | Secret-to-env mapping | Maps `.secret.age` files to env vars |
| `setup-linux.sh` | Linux bootstrap | Installs tools, deploys configs, registers MCP servers |
| `setup-windows.ps1` | Windows bootstrap | PowerShell equivalent (copies instead of symlinks) |
| `ai/claude/CLAUDE.md` | Master Claude instructions | Deployed to `~/.claude/CLAUDE.md` by setup |
| `ai/skills/*/SKILL.md` | Claude skills (20 total) | Deployed to `~/.claude/skills/` |
| `scripts/claude-session-start.sh` | SessionStart hook | Injects vault health + knowledge staleness into Claude context |
| `scripts/vault.sh` | Vault tooling dispatcher (REFACTOR-005) | `vault {health, maintenance, check-escapes}` — single discoverable entry point; each subcommand still runnable standalone |
| `scripts/vault-health.sh` | Obsidian vault health check | Checks plugin status, pending tasks, etc. Also runnable via `vault health`. |
| `scripts/obs-cli.sh` | Obsidian CLI wrapper (Linux) | --no-sandbox, default vault, GUI check |
| `scripts/obs-cli.ps1` | Obsidian CLI wrapper (Windows) | Default vault, GUI check |
| `scripts/knowledge-crystallize.sh` | AI knowledge maintenance | Updates MEMORY.md dates, checks health, prints crystallization checklist |
| `ai/aider/aider.conf.yml` | Aider global config | Deployed to `~/.aider.conf.yml` — 3 tiers via OpenRouter |
| `ai/aider/aider.model.settings.yml` | Aider model settings | Deployed to `~/.aider.model.settings.yml` |
| `.zsh/aliases.zsh` | Shell aliases | Includes `oc` for opencode, `ai`/`aic`/`aia` for aider (legacy, sunset PR2), `tx`/`txl`/`txa`/`txk` for tmux |
| `ai/opencode/opencode.jsonc` | OpenCode config | Provider `opencode-go` (Go subscription, catalog-restricted) + `openrouter` + MCP mirror. Deployed to `~/.config/opencode/`. |
| `AGENTS.md` (root) | Cross-agent SSOT | Canonical system prompt read by OpenCode (natively) and Claude (via this file's pointer). Per-agent files in `ai/<agent>/` and `.github/` delegate here. |
| `tmux.conf` | tmux configuration (Linux only) | Deployed via symlink to `~/.tmux.conf` by `setup-linux.sh` |

## Secrets System

- Encryption: [age](https://github.com/FiloSottile/age) with key at `~/.config/age/key.txt`
- Encrypted files: `sensitive/*.secret.age`
- Mapping: `sensitive/env-mapping.conf` maps filenames to environment variable names
- Loader: `scripts/load-secrets.sh` decrypts and exports at shell startup
- Add a secret: encrypt with `age -r $(age-keygen -y ~/.config/age/key.txt)`, add mapping to `env-mapping.conf`

## Common Workflows

### Adding a new shell script
1. Create in `scripts/`, use `#!/usr/bin/env bash` shebang
2. Source `utils.sh` if you need shared functions
3. Follow the prohibited patterns table above
4. Run `shellcheck` and `bats` before committing

### Adding a new secret
1. Create `sensitive/KEYNAME.secret.age` with age encryption
2. Add mapping line to `sensitive/env-mapping.conf`
3. Test with `. scripts/load-secrets.sh && echo $VAR_NAME`

### Adding a file secret
1. Run `secrets_add_file VAR_NAME filename dest_path` (e.g. `secrets_add_file KUBECONFIG kubelab.kubeconfig ~/.kube/kubelab.config`)
2. It will prompt for the source file path, encrypt it, and add `@VAR=filename>dest` to `env-mapping.conf`
3. Or manually: encrypt with age, add `@VAR=filename>~/.kube/dest.config` to `env-mapping.conf`
4. Test with `secrets_refresh && echo $VAR_NAME` (should point to deployed file path)

### Updating a tool version
1. Edit `versions.conf` at repo root — change `TOOL_VERSION=X.Y.Z`
2. Verify syntax: `bash -n versions.conf && zsh -n versions.conf`
3. Run `~/.local/bin/bats tests/versions-conf.bats` to validate format
4. The new version propagates automatically to `.zshrc`/`.bashrc` via `${VAR:-fallback}` on next shell reload
5. Run `./scripts/healthcheck.sh` to verify the versioned directory exists at `~/Applications/tool-X.Y.Z`

### Adding a new versioned tool
1. Add `NEWTOOL_VERSION=X.Y.Z` to `versions.conf`
2. Add `export NEWTOOL_HOME="$APPS_HOME/newtool-${NEWTOOL_VERSION:-X.Y.Z}"` to both `.zshrc` and `.bashrc`
3. Add `export PATH="$NEWTOOL_HOME/bin:$PATH"` in both RC files (if the tool has binaries)
4. Add version check to `scripts/healthcheck.sh` (sections 2, 3, and 5)
5. Run tests: `~/.local/bin/bats tests/versions-conf.bats tests/healthcheck.bats`

### Running the health check
```bash
./scripts/healthcheck.sh     # Full post-setup verification
# Checks: core tools, versioned paths, version match, symlinks, env vars, optional tools
# Exit 0 = all pass, Exit 1 = failures detected
```

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

See vault runbook: `~/Projects/knowledge/10_projects/dotfiles/40-runbooks/guide-knowledge-distillation.md`

### Using opencode (AI coding agent — primary daily after PR2)
```bash
oc                    # TUI launcher (opencode Go subscription, $10/mo fixed)
qq "tu pregunta"      # one-shot quick-question via opencode-go/qwen3.6-plus (bash/zsh/pwsh)
```
- Default TUI model: DeepSeek V4 Pro (Go catalog). A/B candidate: Kimi K2.6 — selectable via `/models` in TUI.
- `qq` wrapper pinned to `opencode-go/qwen3.6-plus` (multilingual, fast, never-rate-limited). One-shot: each call is a fresh session. Defined in `.zsh/aliases.zsh`, `.bashrc`, and `powershell/profile.ps1`. Cross-platform name is `qq` (not `??`) because PowerShell 7+ reserves `??` as null-coalescing operator.
- Frontier on-demand: provider `openrouter` (consumes existing `OPENROUTER_API_KEY` $5 credit).
- First-time setup: launch `oc`, run `/connect` → select **OpenCode Go**, paste API key from opencode.ai/zen.
- 3-layer PAYG guardrail: (1) `opencode.jsonc` lists only Go models, (2) Zen workspace cap $0, (3) no payment method for PAYG. Runbook: `~/Projects/knowledge/10_projects/dotfiles/40-runbooks/guide-opencode-go-setup.md`.
- Config: `ai/opencode/opencode.jsonc` → `~/.config/opencode/opencode.jsonc` (deployed by `setup-linux.sh`).
- ⚠ Coexistence constraint: don't run `oc` and `claude` in parallel on the same repo until hive MCP adds a lock-file to its auto-commit.

### Using aider (legacy — sunset in PR2)
```bash
ai                    # daily — DeepSeek V3.2 ($0.25/$0.40 per M tokens)
aic                   # coding — Qwen3 Coder ($0.12/$0.75 per M tokens)
aia                   # architecture — DeepSeek Speciale + architect mode ($0.40/$1.20 per M tokens)
```
- Budget target: ~$2/month at heavy use (60 interactions/day)
- Requires: `uv`, Python 3.12 (audioop removed in 3.13), `OPENROUTER_API_KEY`
- To change models: edit `ai/aider/aider.conf.yml` + `.zsh/aliases.zsh`, re-run setup
- **Replaced by opencode in PR2.** Kept here during PR1 coexistence (see spec `specs/AI-011-opencode-bootstrap/`).

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
- If mistakes were corrected, update vault `~/Projects/knowledge/10_projects/dotfiles/lessons.md`

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