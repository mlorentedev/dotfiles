# Dotfiles

Personal development environment: shell configs, AI tool integration, and encrypted secrets management. Supported today on **Linux** and **Windows**. **macOS is planned** (roadmap) — there is no `setup-macos.sh` yet, so the Linux bootstrap is unverified on macOS.

| Platform | Status | Bootstrap |
|---|---|---|
| Linux | Supported | `setup-linux.sh` |
| Windows | Supported | `setup-windows.ps1` |
| macOS | Planned (not yet implemented) | — |

## Quick Start

### Linux

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
./setup-linux.sh
source ~/.zshrc
```

### Windows (PowerShell)

```powershell
git clone https://github.com/mlorentedev/dotfiles.git
cd dotfiles
powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1
# Restart PowerShell after setup
```

Optional: add `-WithDefaults` to also apply ~15 HKCU engineering defaults
(show file extensions/hidden files, disable advertising ID and Bing-in-Start,
dark mode — the [mathiasbynens `.macos`](https://github.com/mathiasbynens/dotfiles/blob/main/.macos)
analog, see `scripts/windows-defaults.ps1`). Off by default; HKCU only, no
admin needed; some changes show after an Explorer restart.

## Features

- **Dual-shell support** — All scripts work in both bash and zsh (POSIX-compatible)
- **Encrypted secrets** — Age-encrypted tokens and file secrets, auto-loaded at login
- **AI integration** — Claude Code (primary) + OpenCode (secondary, Go subscription) + Gemini CLI with 21 custom skills, unified by `AGENTS.md` SSOT
- **Cross-platform** — Symlinks on Linux, copies on Windows (no admin required); macOS planned
- **Editor & shell ergonomics** — `.editorconfig` for cross-IDE consistency + `.inputrc` for case-insensitive tab-completion and arrow-key history search
- **Tested** — 316 BATS tests + ShellCheck + PSScriptAnalyzer in CI

## Structure

```text
├── setup-linux.sh              # Linux setup (symlinks); macOS planned
├── setup-windows.ps1           # Windows setup (copies)
├── scripts/                    # Shell utilities (NOT on PATH — see Human entrypoints below)
│   ├── utils.sh                # Shared function library (sourced by other scripts)
│   ├── load-secrets.sh / .ps1  # Secrets → env vars (sourced at login)
│   ├── init-project.sh / .ps1  # Project bootstrapper
│   ├── healthcheck.ps1         # Windows post-setup verification (Linux: `dotf doctor`)
│   ├── vault.sh                # Vault tooling dispatcher
│   └── …                       # 31 more scripts (hooks, CI helpers, secret tools)
├── sensitive/                  # Encrypted secrets
│   ├── env-mapping.conf        # ENV_VAR=filename mapping
│   └── *.secret.age            # Encrypted files (tracked)
├── AGENTS.md                   # Cross-agent SSOT (canonical system prompt)
├── ai/
│   ├── claude/CLAUDE.md        # Claude Code extensions (pointer to AGENTS.md)
│   ├── agy/AGY.md              # Gemini/AGY extensions (pointer to AGENTS.md)
│   ├── copilot/                # Copilot extensions (pointer to AGENTS.md)
│   ├── opencode/opencode.jsonc # OpenCode config (Go + OpenRouter providers + MCP)
│   └── harness/skills/         # 31 shared AI skills (deployed by compile-harness)
├── ssh/                        # SSH config + public key
├── powershell/profile.ps1      # Windows PowerShell profile
├── tests/*.bats                # BATS test suite
└── .zsh/                       # Zsh modules
```

## Human entrypoints

Scripts in `scripts/` are **not on PATH**. They are invoked via shell aliases (defined in
`.zsh/aliases.zsh` and `.bashrc`) or sourced at login. The table below lists the ~8
scripts that a human ever runs directly — everything else is a library, hook, or CI helper.

| Alias / command | Script | What it does |
|---|---|---|
| `./setup-linux.sh` | `setup-linux.sh` | Bootstrap Linux: install tools, deploy configs, register MCPs |
| `.\setup-windows.ps1` | `setup-windows.ps1` | Bootstrap Windows: same, via PowerShell |
| `dotf doctor` | `dotf` CLI | Post-setup verification (versions, paths, symlinks, env vars, env-contract) |
| `project-init <name> <stack>` | `scripts/init-project.sh` | Bootstrap a new project with dual AI config |
| `vault <subcommand>` | `scripts/vault.sh` | Vault tooling: `vault health`, `vault maintenance`, `vault check-escapes` |
| `profile-shell` | `scripts/shell-profile.sh` | Measure shell startup time (zsh/bash, --detail for per-function) |
| `obs` | `scripts/obs-cli.sh` | Open Obsidian vault (Linux, --no-sandbox, GUI check) |
| `. scripts/load-secrets.sh` | `scripts/load-secrets.sh` | Decrypt age secrets → env vars (auto-sourced at login; manual when adding a new secret) |

## Key Commands

### Secrets

```bash
secrets_add VAR_NAME filename       # Add new env var secret
secrets_add_file VAR FILE DEST      # Add file secret (kubeconfig, SSH keys)
secrets_rotate VAR_NAME             # Rotate existing secret
secrets_show VAR_NAME               # Show value (memory/disk/.age fallback)
secrets_list                        # List all secrets and status
secrets_check                       # Validate mapping integrity
```

### Machine-local overrides

Non-sensitive, per-machine shell config (a host-only `PATH` prepend, a VM-only alias) goes in `~/.zshrc.local` / `~/.bashrc.local` — gitignored, sourced **last** so it can override anything above. Copy from the committed `.zshrc.local.example` / `.bashrc.local.example`.

> **`.local` is not for secrets.** API keys, tokens and credentials always go through the age system above (`sensitive/*.secret.age` + `env-mapping.conf`), never a `.local` file.

### AI Tools

```bash
project-init my-project python      # Bootstrap project with dual AI config
claude                               # Start Claude Code session
> /audit src/auth.py                 # Use skills via slash commands
gp audit "$(cat src/main.py)"       # Gemini prompt function
oc                                   # OpenCode TUI (Go subscription, DeepSeek V4 Pro default)
qq por que tardas tanto?             # one-shot question (no quotes needed in zsh) -> qwen3.6-plus (ES-friendly)
qf explain the C10k problem         # one-shot question -> deepseek-v4-flash (faster, technical)
```

**AI skills** are edited in the vault (`00_meta/skills/<name>/`), compiled to committed
records under `harness/skills/`, and deployed per-agent by `scripts/compile-harness.sh`
(Claude, OpenCode, Gemini/AGY, Copilot). Do not add skill directories to the repo — edit
in the vault and re-run setup. Pipeline details: the vault's `pattern-cross-agent-skill-pipeline.md`.

### Sync

```bash
dotfiles-sync                       # Bidirectional sync + git push/pull
dotfiles-sync --secrets-only        # Only sync sensitive/ files
```

### Diagnostics

```bash
hc                                  # Run healthcheck (versions, paths, symlinks, env vars)
dch                                 # Drift check: repo vs ~/.dotfiles deploy dir
profile-shell                       # Measure shell startup time (zsh default)
profile-shell --shell bash --detail # Per-function breakdown via zprof/xtrace
vault help                          # Vault tooling dispatcher (health / maintenance / check-escapes)
```

### Shell helpers

Portable swiss-army functions in `.zsh/functions.sh`, sourced by **both** bash and
zsh (curated from [mathiasbynens/dotfiles](https://github.com/mathiasbynens/dotfiles)):

```bash
mkd <dir>            # mkdir -p <dir> && cd into it
gz <file>            # show original vs gzipped size + ratio (read-only)
dataurl <file>       # print a base64 data: URI (MIME auto-detected)
targz <file|dir>     # create <input>.tar.gz (zopfli > pigz > gzip by availability)
server [port]        # serve the current dir over HTTP (default 8000) + open browser
getcertnames host[:port]  # print a TLS cert's Common Name + Subject Alt Names
```

The names `mkd`, `gz`, `server` are short and may shadow a binary on `$PATH`. If one
conflicts, re-alias it in `~/.zshrc.local` / `~/.bashrc.local` (see *Machine-local
overrides*).

### tmux

Two use cases this setup is tuned for: **(1) split-pane multiplexing** (editor + AI agent + tests side by side) and **(2) session persistence** (close the laptop / drop SSH and come back to the same state).

```bash
# --- The 6 commands you actually need ---

tx dotfiles                # Start (or re-attach) a session named "dotfiles"
                           # Inside tmux now: prompt shows [dotfiles]

# Split for editor + AI + tests:
#   C-b %                  Split vertically  (editor | agent)
#   C-b "                  Split horizontally (... above tests)
#   C-b h/j/k/l            Move between panes (vim-style)
#   C-b z                  Zoom current pane fullscreen (toggle)

# Pause / resume:
#   C-b d                  Detach — session keeps running in background
tx dotfiles                # Re-attach later (same command). Layout preserved.

# --- The rest (use occasionally) ---

txl                        # List all sessions
txa                        # Attach to most recent (no name needed)
txk <name>                 # Kill a named session
sshmux <host> [session]    # SSH + attach-or-create remote tmux (survives drops)

# Inside tmux:
#   C-b r                  Reload ~/.tmux.conf after editing
#   C-b x                  Close current pane
#   C-b [                  Scroll mode (q to exit, / to search)
```

Full reference and pane-layout recipes: [`docs/runbooks/guide-tmux.md`](docs/runbooks/guide-tmux.md).

## Requirements

**Linux:** git, bash/zsh, tmux (`sudo apt install tmux`)

**Windows:** git, PowerShell

**macOS:** planned — not yet supported (no `setup-macos.sh`)

**Recommended:** age, gh (GitHub CLI), direnv, zoxide, eza

## Contributing

PRs ≥50 LOC of production diff must include an active `specs/<feature-id>/` folder (Spec-Driven Development). The `spec-gate` CI check enforces this; failures link back to `AGENTS.md` "Discipline Gate". Escape hatch: add the `skip-sdd` label AND a non-empty `## SDD skip rationale` section in the PR body. Optional local pre-push hook: `./scripts/install-precommit.sh --with-sdd-gate`.

## Documentation

Project-bound knowledge lives in [`docs/`](docs/) (docs-as-code):

- [`docs/architecture.md`](docs/architecture.md) — **where does X live**: the normative repo tree, the `dotf` CLI layout, and the language boundary pointers (drift-guarded by CI)
- [`docs/adr/`](docs/adr/) — Architecture Decision Records (age encryption, dual-shell, BATS testing, two-directory sync, symlinks vs copies, multi-agent runtime, model-tier policy, …) plus the repo audits and architecture map
- [`docs/runbooks/`](docs/runbooks/) — operational procedures (secrets management, AI tools setup, tool installation, tmux, OpenCode, self-deploy timer)
- [`docs/troubleshooting/`](docs/troubleshooting/) — known issues and their fixes (secrets, AI tools, Hive MCP, claude-mem)
- [`docs/lessons.md`](docs/lessons.md) — accumulated gotchas and post-mortems

Strategic context, roadmap, and session memory live in the maintainer's cross-project knowledge store and are intentionally not committed here.

## Related Projects

- [Boilerplates](https://github.com/mlorentedev/boilerplates) — Project templates
- [Cheatsheets](https://github.com/mlorentedev/cheat-sheets) — Quick references

## License

[MIT License](LICENSE) — Free to use and modify with attribution.
