# Dotfiles

Personal configuration files for development tools, shell environments, and AI coding assistants. Works with Bash and Zsh, includes encrypted secrets management with age, and provides Claude Code/Gemini CLI integration.

## Quick Start

```bash
git clone https://github.com/mlorentedev/dotfiles.git ~/.dotfiles
cd ~/.dotfiles
./install.sh
source ~/.zshrc
```

## Features

- **Shell Configuration:** Aliases, functions, PATH management for Bash/Zsh
- **Secrets Management:** Age-encrypted secrets loaded as environment variables
- **AI Integration:** Claude Code and Gemini CLI with custom skills
- **Project Initialization:** `claude-init` command to bootstrap new projects

## Structure

```text
├── scripts/                    # Shell utilities (added to PATH)
│   ├── utils.sh                # Shared function library
│   ├── load-secrets.sh         # Secrets → env vars
│   ├── init-project.sh         # Project bootstrapper (bash)
│   ├── init-project.bat        # Project bootstrapper (Windows)
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
# Initialize new project with Claude configuration
claude-init my-project python
claude-init my-project go
claude-init . node

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

Dotfiles (shell config, secrets) are Linux/macOS specific. On Windows, only Claude Code is configured:

```batch
:: 1. Clone the repository
git clone https://github.com/mlorentedev/dotfiles.git
cd dotfiles

:: 2. Run setup (copies skills to %USERPROFILE%\.claude\)
scripts\windows-setup.bat

:: 3. Initialize projects
%USERPROFILE%\.claude\init-project.bat my-project python
```

**Alternative with Git Bash:** If you have Git for Windows, you can use the bash scripts directly:

```bash
# In Git Bash
./scripts/init-project.sh my-project python
```

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
