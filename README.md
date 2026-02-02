# Dotfiles

Personal configuration files for development tools, shell environments, and AI coding assistants. Works with Bash and Zsh, includes encrypted secrets management with age, and provides Claude Code/Gemini CLI integration.

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

# Option 1: One-time bypass (no permanent changes)
powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1

# Option 2: Set policy for current user (recommended, persistent)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
.\setup-windows.ps1

# Restart PowerShell, then verify
project-init test-project python
```

## Features

- **Shell Configuration:** Aliases, functions, PATH management for Bash/Zsh (Linux/macOS) and PowerShell (Windows)
- **Secrets Management:** Age-encrypted secrets loaded as environment variables (Linux/macOS)
- **AI Integration:** Claude Code and Gemini CLI with custom skills
- **Project Initialization:** `project-init` command to bootstrap new projects with dual AI configuration

## Structure

```text
├── setup-linux.sh              # Linux/macOS setup script
├── setup-windows.ps1           # Windows setup script (PowerShell)
├── powershell/                 # Windows shell configs
│   └── profile.ps1             # PowerShell profile template
├── scripts/                    # Shell utilities (added to PATH)
│   ├── utils.sh                # Shared function library
│   ├── load-secrets.sh         # Secrets → env vars
│   ├── init-project.sh         # Project bootstrapper (bash)
│   ├── init-project.ps1        # Project bootstrapper (PowerShell)
│   ├── github-secrets-manager.sh
│   ├── age-encrypt-decrypt.sh
│   └── dotfiles-sync.sh
├── sensitive/                  # Encrypted secrets
│   ├── env-mapping.conf        # ENV_VAR=filename mapping
│   └── *.secret.age            # Encrypted files (tracked)
├── ai/
│   ├── claude/                 # Claude Code configuration
│   │   └── CLAUDE.md           # Master instructions
│   ├── gemini/                 # Gemini CLI configuration
│   │   └── GEMINI.md           # Master instructions
│   └── skills/                 # Shared AI skills
│       ├── audit/SKILL.md      # /audit - Security review
│       ├── refactor/SKILL.md   # /refactor - Code cleanup
│       ├── test/SKILL.md       # /test - Test generation
│       ├── doc/SKILL.md        # /doc - Documentation
│       └── docker/SKILL.md     # /docker - Containerization
├── .zsh/                       # Zsh modules
└── docs/                       # Documentation
```

## Secrets Management

Secrets are encrypted with [age](https://github.com/FiloSottile/age) and automatically loaded on shell startup.

```bash
secrets_add VAR_NAME filename   # Add new secret
secrets_rotate VAR_NAME         # Update existing
secrets_list                    # Show all secrets
secrets_check                   # Validate integrity
```

See [docs/SECRETS.md](docs/SECRETS.md) for full documentation.

## AI Tools

### Claude Code

```bash
# Initialize new project with dual AI configuration
project-init my-project python
project-init my-project go
project-init . node

# Available skills (slash commands)
/audit      # Security audit
/refactor   # Code refactoring
/test       # Test generation
/doc        # Documentation
/docker     # Containerization
```

### Gemini CLI

```bash
# Use prompts via gp function
gp audit "$(cat src/main.py)"
gp refactor "$(cat src/utils.go)"
```

See [docs/AI.md](docs/AI.md) for complete setup guide.

## Syncing

Two-directory model for stability:

- `~/.dotfiles/` - Stable local installation
- `~/Projects/dotfiles/` - Development repository

```bash
dotfiles-sync                   # Bidirectional sync + git push/pull
dotfiles-sync --secrets-only    # Only sync sensitive/
```

## Windows Setup

Windows uses PowerShell for setup (no admin rights required, no symlinks):

```powershell
# 1. Clone the repository
git clone https://github.com/mlorentedev/dotfiles.git
cd dotfiles

# 2. Run setup (one-time execution policy bypass)
powershell -ExecutionPolicy Bypass -File .\setup-windows.ps1

# 3. Restart PowerShell, then initialize projects
project-init my-project python
```

Features on Windows:
- Claude and Gemini configuration deployed to `~\.claude\` and `~\.gemini\`
- PowerShell profile with aliases (`c`, `g`, `k`) and `project-init` function
- Scripts folder added to User PATH
- Git configuration copied (if not already present)

## Requirements

**Linux/macOS:** git, bash/zsh

**Windows:** git

**Recommended:** age, gh (GitHub CLI), direnv, zoxide, eza

## Documentation

- [AI.md](docs/AI.md) - AI tools setup and workflow
- [SECRETS.md](docs/SECRETS.md) - Secrets management guide
- [GUIDE.md](docs/GUIDE.md) - General usage and customization

## Related Projects

- [Boilerplates](https://github.com/mlorentedev/boilerplates) - Project templates
- [Cheatsheets](https://github.com/mlorentedev/cheat-sheets) - Quick references
- [My list](https://mlorente.dev) - More detailed explanations

## License

MIT
