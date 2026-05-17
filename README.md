# Dotfiles

Personal development environment: shell configs, AI tool integration, and encrypted secrets management. Works across Linux, macOS, and Windows.

## Quick Start

### Linux / macOS

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

## Features

- **Dual-shell support** — All scripts work in both bash and zsh (POSIX-compatible)
- **Encrypted secrets** — Age-encrypted tokens and file secrets, auto-loaded at login
- **AI integration** — Claude Code (primary) + OpenCode (secondary, Go subscription) + Gemini CLI with 21 custom skills, unified by `AGENTS.md` SSOT
- **Cross-platform** — Symlinks on Linux/macOS, copies on Windows (no admin required)
- **Tested** — 316 BATS tests + ShellCheck + PSScriptAnalyzer in CI

## Structure

```text
├── setup-linux.sh              # Linux/macOS setup (symlinks)
├── setup-windows.ps1           # Windows setup (copies)
├── scripts/                    # Shell utilities (added to PATH)
│   ├── utils.sh                # Shared function library
│   ├── load-secrets.sh         # Secrets → env vars (Linux, sourced at login)
│   ├── load-secrets.ps1        # Secrets → env vars (Windows)
│   ├── dotfiles-sync.sh        # Bidirectional sync (Linux)
│   ├── dotfiles-sync.ps1       # Bidirectional sync (Windows)
│   ├── claude-session-start.sh # Claude SessionStart hook (Linux)
│   ├── claude-session-start.ps1# Claude SessionStart hook (Windows)
│   ├── init-project.sh         # Project bootstrapper (bash)
│   ├── init-project.ps1        # Project bootstrapper (PowerShell)
│   ├── github-secrets-manager.sh
│   └── age-encrypt-decrypt.sh
├── sensitive/                  # Encrypted secrets
│   ├── env-mapping.conf        # ENV_VAR=filename mapping
│   └── *.secret.age            # Encrypted files (tracked)
├── AGENTS.md                   # Cross-agent SSOT (canonical system prompt)
├── ai/
│   ├── claude/CLAUDE.md        # Claude Code extensions (pointer to AGENTS.md)
│   ├── gemini/GEMINI.md        # Gemini extensions (pointer to AGENTS.md)
│   ├── copilot/                # Copilot extensions (pointer to AGENTS.md)
│   ├── opencode/opencode.jsonc # OpenCode config (Go + OpenRouter providers + MCP)
│   └── skills/                 # 21 shared AI skills
├── ssh/                        # SSH config + public key
├── powershell/profile.ps1      # Windows PowerShell profile
├── tests/*.bats                # BATS test suite
└── .zsh/                       # Zsh modules
```

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

### AI Tools

```bash
project-init my-project python      # Bootstrap project with dual AI config
claude                               # Start Claude Code session
> /audit src/auth.py                 # Use skills via slash commands
gp audit "$(cat src/main.py)"       # Gemini prompt function
oc                                   # OpenCode TUI (Go subscription, DeepSeek V4 Pro default)
```

### Sync

```bash
dotfiles-sync                       # Bidirectional sync + git push/pull
dotfiles-sync --secrets-only        # Only sync sensitive/ files
```

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

Full reference and pane-layout recipes: `~/Projects/knowledge/10_projects/dotfiles/40-runbooks/guide-tmux.md`.

## Requirements

**Linux/macOS:** git, bash/zsh, tmux (`sudo apt install tmux`)

**Windows:** git, PowerShell

**Recommended:** age, gh (GitHub CLI), direnv, zoxide, eza

## Documentation

Detailed documentation lives in the private knowledge vault:

- **Runbooks** — Secrets management, AI tools setup, tool installation
- **Troubleshooting** — Common issues with secrets and AI tools
- **ADRs** — Architecture decisions (age encryption, dual-shell, BATS testing, two-directory sync, symlinks vs copies)

## Related Projects

- [Boilerplates](https://github.com/mlorentedev/boilerplates) — Project templates
- [Cheatsheets](https://github.com/mlorentedev/cheat-sheets) — Quick references

## License

[MIT License](LICENSE) — Free to use and modify with attribution.
