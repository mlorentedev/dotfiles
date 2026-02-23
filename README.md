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
- **AI integration** — Claude Code + Gemini CLI with 21 custom skills
- **Cross-platform** — Symlinks on Linux/macOS, copies on Windows (no admin required)
- **Tested** — 106 BATS tests + ShellCheck in CI

## Structure

```text
├── setup-linux.sh              # Linux/macOS setup (symlinks)
├── setup-windows.ps1           # Windows setup (copies)
├── scripts/                    # Shell utilities (added to PATH)
│   ├── utils.sh                # Shared function library
│   ├── load-secrets.sh         # Secrets → env vars (sourced at login)
│   ├── dotfiles-sync.sh        # Bidirectional sync between dirs
│   ├── init-project.sh         # Project bootstrapper (bash)
│   ├── init-project.ps1        # Project bootstrapper (PowerShell)
│   ├── github-secrets-manager.sh
│   └── age-encrypt-decrypt.sh
├── sensitive/                  # Encrypted secrets
│   ├── env-mapping.conf        # ENV_VAR=filename mapping
│   └── *.secret.age            # Encrypted files (tracked)
├── ai/
│   ├── claude/CLAUDE.md        # Master Claude instructions
│   ├── gemini/GEMINI.md        # Master Gemini instructions
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
```

### Sync

```bash
dotfiles-sync                       # Bidirectional sync + git push/pull
dotfiles-sync --secrets-only        # Only sync sensitive/ files
```

## Requirements

**Linux/macOS:** git, bash/zsh

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
