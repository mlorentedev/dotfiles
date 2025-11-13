# ============================================================================
# Oh My Zsh Setup
# ============================================================================
export ZSH="$HOME/.oh-my-zsh"

# Theme
ZSH_THEME="robbyrussell"

# Plugins
plugins=(
    git
    docker
    kubectl
    terraform
    ansible
    colored-man-pages
    command-not-found
    sudo
    history-substring-search
)

# Load Oh My Zsh (skip if not installed)
if [ -f "$ZSH/oh-my-zsh.sh" ]; then
    source "$ZSH/oh-my-zsh.sh"
fi

# ============================================================================
# History Configuration
# ============================================================================
HISTFILE=~/.zsh_history
HISTSIZE=10000
SAVEHIST=20000
setopt HIST_IGNORE_ALL_DUPS     # Remove older duplicate entries
setopt HIST_REDUCE_BLANKS       # Remove blank lines
setopt INC_APPEND_HISTORY       # Save commands immediately
setopt SHARE_HISTORY            # Share history between sessions
setopt HIST_VERIFY              # Show command before executing from history

# ============================================================================
# Zsh Options
# ============================================================================
setopt AUTO_CD                  # cd by typing directory name
setopt AUTO_PUSHD               # Make cd push old directory onto directory stack
setopt PUSHD_IGNORE_DUPS        # Don't push duplicates
setopt CORRECT                  # Command correction
setopt INTERACTIVE_COMMENTS     # Allow comments in interactive shell

# ============================================================================
# Environment Variables
# ============================================================================
export EDITOR=vim
export VISUAL=vim
export PAGER=less

# Dotfiles directory
export DOTFILES_DIR="$HOME/.dotfiles"

# Add ~/.local/bin to PATH if it exists
if [ -d "$HOME/.local/bin" ]; then
    PATH="$HOME/.local/bin:$PATH"
fi

# Smart PATH management - remove duplicates
typeset -U PATH

# ============================================================================
# Load Custom Configurations
# ============================================================================
# Load custom aliases
if [ -f ~/.zsh/aliases.zsh ]; then
    source ~/.zsh/aliases.zsh
fi

# Load custom functions
if [ -f ~/.zsh/functions.zsh ]; then
    source ~/.zsh/functions.zsh
fi

# Load NVM configuration
if [ -f ~/.zsh/nvm.zsh ]; then
    source ~/.zsh/nvm.zsh
fi

# ============================================================================
# Tool Integrations
# ============================================================================
# direnv (auto-load .envrc files)
if command -v direnv &> /dev/null; then
    eval "$(direnv hook zsh)"
fi

# zoxide (smart cd replacement)
if command -v zoxide &> /dev/null; then
    eval "$(zoxide init zsh)"
fi

# fzf (fuzzy finder)
if command -v fzf &> /dev/null; then
    if [ -f ~/.fzf.zsh ]; then
        source ~/.fzf.zsh
    fi
fi

# Starship prompt (overrides Oh My Zsh theme if installed)
if command -v starship &> /dev/null; then
    eval "$(starship init zsh)"
fi

# kubectl completion
if command -v kubectl &> /dev/null; then
    source <(kubectl completion zsh)
fi

# helm completion
if command -v helm &> /dev/null; then
    source <(helm completion zsh)
fi

# ============================================================================
# Enhanced Prompt (if not using Starship)
# ============================================================================
if ! command -v starship &> /dev/null; then
    # Add kubernetes context to prompt
    kube_ps1() {
        if command -v kubectl &> /dev/null; then
            local context=$(kubectl config current-context 2>/dev/null)
            local namespace=$(kubectl config view --minify --output 'jsonpath={..namespace}' 2>/dev/null)
            if [ -n "$context" ]; then
                if [ -n "$namespace" ]; then
                    echo " %F{cyan}[☸ $context/$namespace]%f"
                else
                    echo " %F{cyan}[☸ $context]%f"
                fi
            fi
        fi
    }

    # Add terraform workspace to prompt
    tf_ps1() {
        if [ -d .terraform ] && command -v terraform &> /dev/null; then
            local workspace=$(terraform workspace show 2>/dev/null)
            if [ -n "$workspace" ] && [ "$workspace" != "default" ]; then
                echo " %F{magenta}[tf:$workspace]%f"
            fi
        fi
    }

    # Custom right prompt
    RPS1='$(kube_ps1)$(tf_ps1)'
fi

# ============================================================================
# Language Version Managers
# ============================================================================
# Add language-specific paths if they exist
# These are examples - customize based on your setup

# Java
if [ -d "$HOME/Apps/jdk-21.0.4" ]; then
    export JAVA_HOME="$HOME/Apps/jdk-21.0.4"
    PATH="$JAVA_HOME/bin:$PATH"
fi

# Maven
if [ -d "$HOME/Apps/apache-maven-3.9.4" ]; then
    export MAVEN_HOME="$HOME/Apps/apache-maven-3.9.4"
    PATH="$MAVEN_HOME/bin:$PATH"
fi

# Python
if [ -d "$HOME/Apps/python-3.12.6" ]; then
    export PYTHON_HOME="$HOME/Apps/python-3.12.6"
    PATH="$PYTHON_HOME/bin:$PATH"
fi

# Ruby
if [ -d "$HOME/Apps/ruby-3.1.4" ]; then
    export RUBY_HOME="$HOME/Apps/ruby-3.1.4"
    export GEM_HOME="$RUBY_HOME/gems"
    PATH="$RUBY_HOME/bin:$GEM_HOME/bin:$PATH"
fi

# Go
if [ -d "$HOME/Apps/go-1.23.1" ]; then
    export GO_HOME="$HOME/Apps/go-1.23.1"
    export GOPATH="$HOME/go"
    PATH="$GO_HOME/bin:$GOPATH/bin:$PATH"
fi

# ============================================================================
# Custom Local Configuration
# ============================================================================
# Source local customizations (not tracked by git)
if [ -f ~/.zshrc.local ]; then
    source ~/.zshrc.local
fi

# ============================================================================
# Performance Optimization
# ============================================================================
# Lazy load NVM for faster shell startup
# NVM is loaded in ~/.zsh/nvm.zsh when needed
