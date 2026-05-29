---
id: "dotfiles-runbook-tool-installation"
type: runbook
status: active
tags: [runbook, dotfiles, tools, installation]
created: "2026-02-22"
owner: manu
---

# Tool Installation

Installation instructions for development tools used with the dotfiles.

## Essential

| Tool | Purpose |
|------|---------|
| `git` | Version control |
| `bash` | Shell (backup) |
| `zsh` | Primary shell |

## Shell Improvements

| Tool | Purpose |
|------|---------|
| `oh-my-zsh` | Zsh themes and plugins |
| `eza` | Better `ls` with colors |
| `zoxide` | Smart directory jumping (`z project` instead of `cd ~/work/project`) |
| `direnv` | Auto-load environment variables per project |

## Development Tools

| Tool | Purpose |
|------|---------|
| `node` + `npm` | JavaScript runtime |
| `nvm` | Node version manager |
| `python3` + `pip` | Python development |
| `docker` | Containerization |
| `docker-compose` | Multi-container apps |

## DevOps Tools

| Tool | Purpose |
|------|---------|
| `kubectl` | Kubernetes management |
| `helm` | Kubernetes packages |
| `terraform` | Infrastructure as code |
| `ansible` | Server configuration |
| `gh` | GitHub CLI |
| `minikube` | Local Kubernetes development |

## Other

| Tool | Purpose |
|------|---------|
| `age` | File encryption (secrets system) |
| `pre-commit` | Git hooks |

## Installation Commands

### Basics

```bash
# Git
sudo apt update && sudo apt install git       # Ubuntu/Debian
brew install git                               # macOS

# Zsh
sudo apt install zsh                           # Ubuntu/Debian
# macOS: already installed

# Oh My Zsh
sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
```

### Shell Improvements

```bash
# eza
sudo apt install cargo && cargo install eza    # Ubuntu/Debian
brew install eza                               # macOS

# zoxide
curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash
brew install zoxide                            # macOS

# direnv
sudo apt install direnv                        # Ubuntu/Debian
brew install direnv                            # macOS
```

### Development

```bash
# Node.js (via nvm - recommended)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install --lts

# Python
sudo apt install python3 python3-pip          # Ubuntu/Debian
brew install python3                           # macOS

# Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Docker Compose
sudo apt install docker-compose-plugin         # Ubuntu (as plugin)
```

### DevOps

```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Terraform
brew tap hashicorp/tap && brew install hashicorp/tap/terraform   # macOS

# Ansible
sudo apt install ansible                       # Ubuntu/Debian
pip3 install ansible                           # anywhere

# GitHub CLI
brew install gh                                # macOS
# Ubuntu: see https://cli.github.com for apt setup
```

### Encryption

```bash
# age
brew install age                               # macOS
go install filippo.io/age/cmd/...@latest       # with Go

# pre-commit
pip3 install pre-commit
```

### Language-Specific

```bash
# Java (via SDKMAN)
curl -s "https://get.sdkman.io" | bash
sdk install java 21.0.1-open
sdk install maven 3.9.4

# Go
brew install go                                # macOS
sudo apt install golang-go                     # Ubuntu

# Ruby (via rbenv)
git clone https://github.com/rbenv/rbenv.git ~/.rbenv
git clone https://github.com/rbenv/ruby-build.git ~/.rbenv/plugins/ruby-build
rbenv install 3.1.4 && rbenv global 3.1.4
```

## Post-Installation

```bash
# Make Zsh default shell
chsh -s $(which zsh)

# Add to ~/.zshrc
eval "$(zoxide init zsh)"     # smart cd
eval "$(direnv hook zsh)"     # auto env loading

# Configure Git
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

## Check What's Installed

```bash
for tool in git zsh node docker kubectl terraform minikube age gh pre-commit; do
    if command -v $tool >/dev/null 2>&1; then
        echo "+ $tool"
    else
        echo "- $tool"
    fi
done
```

## Related

- [AI Tools Setup](ai-tools-setup.md) — Claude Code and Gemini installation
- [Secrets Management](secrets-management.md) — Requires `age`
- Project overview — see the repo `README.md` (strategic context lives in the maintainer's knowledge store)
