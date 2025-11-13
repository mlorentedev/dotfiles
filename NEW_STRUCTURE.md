# New Stow-based Directory Structure

## Design Philosophy
- Each module is self-contained and can be installed independently
- GNU Stow creates symlinks from module subdirectories to $HOME
- Modules follow XDG Base Directory specification where appropriate
- Easy to test, maintain, and extend

## Directory Structure

```
dotfiles/
├── bash/                    # Bash shell configuration
│   ├── .bashrc
│   ├── .bash_profile
│   └── README.md
├── zsh/                     # Zsh shell configuration
│   ├── .zshrc
│   ├── .zsh/
│   │   ├── aliases.zsh
│   │   ├── functions.zsh
│   │   └── nvm.zsh
│   └── README.md
├── git/                     # Git configuration
│   ├── .gitconfig
│   └── README.md
├── shell-common/            # Common shell configs (bash + zsh)
│   ├── .profile
│   └── README.md
├── scripts/                 # Utility scripts (installed to ~/.local/bin)
│   ├── .local/
│   │   └── bin/
│   │       ├── utils.sh
│   │       ├── age-encrypt-decrypt
│   │       ├── github-secrets-manager
│   │       ├── secrets-wrapper
│   │       └── install-precommit
│   └── README.md
├── direnv/                  # direnv configuration
│   ├── .config/
│   │   └── direnv/
│   │       └── direnvrc
│   └── README.md
├── starship/                # Starship prompt
│   ├── .config/
│   │   └── starship.toml
│   └── README.md
├── docker/                  # Docker shortcuts
│   ├── .docker/
│   │   └── config.json
│   └── README.md
├── kubernetes/              # K8s configuration
│   ├── .kube/
│   │   └── config
│   └── README.md
├── terraform/               # Terraform configuration
│   ├── .terraform.d/
│   └── README.md
├── secrets/                 # Encrypted secrets (optional install)
│   ├── sensitive/
│   └── README.md
├── test/                    # Comprehensive test suite
│   ├── run-all-tests.sh
│   ├── test-stow.sh
│   ├── test-bootstrap.sh
│   ├── test-aliases.sh
│   ├── test-path.sh
│   ├── test-secrets.sh
│   ├── bats/
│   │   ├── test-utils.bats
│   │   └── test-scripts.bats
│   └── docker/
│       ├── Dockerfile.ubuntu-20.04
│       ├── Dockerfile.ubuntu-22.04
│       └── Dockerfile.ubuntu-24.04
├── .github/
│   ├── workflows/
│   │   └── test.yml
│   └── hooks/
│       └── validate-commit-msg.sh
├── bootstrap.sh             # Main installation script
├── tools/                   # Tool installation scripts
│   ├── install-devops.sh
│   ├── install-containers.sh
│   ├── install-kubernetes.sh
│   ├── install-iac.sh
│   └── install-shell.sh
├── Makefile                 # Common operations
├── .pre-commit-config.yaml
├── .gitignore
├── README.md
└── MIGRATION.md             # Migration guide from old structure
```

## Installation Flow

1. **bootstrap.sh --minimal**: Installs only shell configs (bash, zsh, git, shell-common)
2. **bootstrap.sh --tools**: Installs configs + DevOps toolchain
3. **bootstrap.sh --all**: Everything including secrets setup

## Testing Strategy

1. Unit tests for utility functions (BATS)
2. Integration tests for symlinks (test-stow.sh)
3. Functional tests for aliases and PATH (test-aliases.sh, test-path.sh)
4. Docker-based tests on Ubuntu 20.04, 22.04, 24.04
5. GitHub Actions CI/CD for automated testing
