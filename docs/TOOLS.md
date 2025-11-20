# Tool Installation Guide

Installation instructions for the tools I use in my dotfiles. Some are required, others make life easier.

## What You Need

### Essential Stuff

| Tool | What it does |
|------|-------------|
| `git` | Version control |
| `bash` | Shell (backup) |
| `zsh` | Better shell |

### Makes Things Nicer

| Tool | What it does | Why I like it |
|------|-------------|---------------|
| `oh-my-zsh` | Zsh themes and plugins | Makes the shell pretty |
| `eza` | Better `ls` with colors | Easier to read file listings |
| `zoxide` | Smart directory jumping | Type `z project` instead of `cd ~/work/project` |
| `direnv` | Auto-load environment variables | Different .env per project |

### Development Tools  

| Tool | What it does | When you need it |
|------|-------------|-----------------|
| `node` + `npm` | JavaScript runtime | Working with JavaScript/Node.js |
| `nvm` | Node version manager | Multiple Node versions |
| `python3` + `pip` | Python | Python development |
| `docker` | Containerization | Running containers |
| `docker-compose` | Multi-container apps | Complex Docker setups |

### DevOps Tools

| Tool | What it does |
|------|-------------|
| `kubectl` | Kubernetes management |
| `helm` | Kubernetes packages |
| `terraform` | Infrastructure as code |
| `ansible` | Server configuration |
| `gh` | GitHub CLI |
| `minikube` | Local Kubernetes development |

### Other Useful Stuff

| Tool | What it does |
|------|-------------|
| `age` | File encryption |
| `pre-commit` | Git hooks |
| `console-ninja` | VS Code debugging extension CLI |

## Installing Stuff

### The Basics

#### Git

```bash
# Ubuntu/Debian
sudo apt update && sudo apt install git

# macOS
brew install git

# RHEL/Fedora/CentOS
sudo dnf install git
```

#### Zsh

```bash
# Ubuntu/Debian  
sudo apt install zsh

# macOS - already there

# RHEL/Fedora
sudo dnf install zsh
```

### Shell Improvements

#### Oh My Zsh

```bash
sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
```

#### eza (better ls)  

```bash
# Ubuntu/Debian - need rust first
sudo apt install cargo
cargo install eza

# macOS
brew install eza

# Arch
sudo pacman -S eza

# Manual install
wget https://github.com/eza-community/eza/releases/latest/download/eza_x86_64-unknown-linux-gnu.tar.gz
tar -xzf eza_x86_64-unknown-linux-gnu.tar.gz
sudo mv eza /usr/local/bin/
```

#### zoxide (smart cd)

```bash
# Easy way
curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash

# Package managers
brew install zoxide              # macOS
sudo apt install zoxide         # Ubuntu/Debian  
sudo pacman -S zoxide           # Arch
```

#### direnv (auto env vars)

```bash
sudo apt install direnv         # Ubuntu/Debian
brew install direnv              # macOS
curl -sfL https://direnv.net/install.sh | bash  # anywhere
```

### Development Stuff

#### Node.js

```bash
# Best way - use nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
# restart terminal
nvm install --lts

# Quick way
sudo apt install nodejs npm     # Ubuntu/Debian
brew install node npm           # macOS
```

#### Python

```bash
sudo apt install python3 python3-pip    # Ubuntu/Debian
brew install python3                     # macOS
sudo dnf install python3 python3-pip    # RHEL/Fedora
```

#### Docker  

```bash
# Linux - easy way
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# macOS
brew install --cask docker
```

#### Docker Compose

```bash
sudo apt install docker-compose-plugin    # Ubuntu - as plugin

# Or standalone version
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### DevOps Stuff

#### kubectl  

```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Easier ways
brew install kubectl             # macOS
sudo apt install kubectl        # Ubuntu
```

#### Helm

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash  # anywhere
brew install helm                                                                 # macOS
```

#### Minikube

```bash
# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# macOS
brew install minikube

# Windows
choco install minikube
```

#### Terraform

```bash
# macOS - easy
brew tap hashicorp/tap && brew install hashicorp/tap/terraform

# Linux - manual way
wget https://releases.hashicorp.com/terraform/1.5.0/terraform_1.5.0_linux_amd64.zip
unzip terraform_1.5.0_linux_amd64.zip
sudo mv terraform /usr/local/bin/
```

#### Ansible

```bash
sudo apt install ansible        # Ubuntu/Debian
brew install ansible            # macOS
pip3 install ansible           # anywhere with pip
```

#### GitHub CLI

```bash
brew install gh                 # macOS - easiest

# Ubuntu - official way
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update && sudo apt install gh
```

### Other Stuff

#### age (encryption)

```bash
brew install age                # macOS
go install filippo.io/age/cmd/...@latest  # if you have go

# Manual Linux install
wget https://github.com/FiloSottile/age/releases/latest/download/age-v1.1.1-linux-amd64.tar.gz
tar -xzf age-v1.1.1-linux-amd64.tar.gz
sudo mv age/age* /usr/local/bin/
```

#### pre-commit

```bash
pip3 install pre-commit         # easiest
brew install pre-commit         # macOS
```

#### Console Ninja

```bash
# VS Code extension - install via VS Code extensions
# Or use npm for CLI tools
npm install -g @console-ninja/cli

# The CLI gets automatically added to PATH by the VS Code extension
```

## Quick Setup

Want to install the essentials quickly? Here's a simple script:

```bash
#!/bin/bash
# Install the basics

# Linux
if [[ "$OSTYPE" == linux* ]]; then
    sudo apt update
    sudo apt install -y git zsh curl
    
# macOS  
elif [[ "$OSTYPE" == darwin* ]]; then
    # Install Homebrew if missing
    if ! command -v brew &> /dev/null; then
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi
    brew install git
fi

# Oh My Zsh
sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended

# Nice extras
curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash

echo "Basic setup done!"
```

## Language-Specific Setup

### Java

```bash
# SDKMAN is easiest for Java versions
curl -s "https://get.sdkman.io" | bash
source "$HOME/.sdkman/bin/sdkman-init.sh"
sdk install java 21.0.1-open
sdk install maven 3.9.4
```

### Ruby  

```bash
# rbenv for version management
git clone https://github.com/rbenv/rbenv.git ~/.rbenv
git clone https://github.com/rbenv/ruby-build.git ~/.rbenv/plugins/ruby-build
rbenv install 3.1.4
rbenv global 3.1.4
```

### Go

```bash
brew install go             # macOS
sudo apt install golang-go  # Ubuntu
```

## Check What's Installed

Quick script to see what you have:

```bash
#!/bin/bash

for tool in git zsh node docker kubectl terraform minikube age gh pre-commit; do
    if command -v $tool &> /dev/null; then
        echo "+ $tool"
    else
        echo "- $tool"
    fi
done
```

## After Installing

### Make Zsh your default shell

```bash
chsh -s $(which zsh)
```

### Set up the integrations  

Add these to your shell config:

```bash
eval "$(zoxide init zsh)"     # smart cd
eval "$(direnv hook zsh)"     # auto env loading
```

### Configure Git

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

## Common Issues

- **Permission errors**: Make sure you can use `sudo`
- **Command not found**: Restart your terminal  
- **Docker permission**: Run `sudo usermod -aG docker $USER`
- **PATH problems**: Check `/usr/local/bin` is in your PATH

Most tools have `--help` if you get stuck.
