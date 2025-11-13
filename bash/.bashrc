# ~/.bashrc: executed by bash(1) for non-login shells.

# If not running interactively, don't do anything
case $- in
    *i*) ;;
      *) return;;
esac

# ============================================================================
# History Configuration
# ============================================================================
HISTCONTROL=ignoreboth          # Ignore duplicates and commands starting with space
HISTSIZE=10000                  # Commands in memory
HISTFILESIZE=20000              # Commands in history file
shopt -s histappend             # Append to history, don't overwrite

# ============================================================================
# Shell Options
# ============================================================================
shopt -s checkwinsize           # Update LINES and COLUMNS after each command
shopt -s globstar 2>/dev/null   # Enable ** pattern (bash 4+)
shopt -s cdspell                # Auto-correct minor typos in cd
shopt -s dirspell               # Auto-correct directory names

# ============================================================================
# Colors and Prompt
# ============================================================================
# Enable color support
if [ -x /usr/bin/dircolors ]; then
    test -r ~/.dircolors && eval "$(dircolors -b ~/.dircolors)" || eval "$(dircolors -b)"
    alias ls='ls --color=auto'
    alias grep='grep --color=auto'
    alias fgrep='fgrep --color=auto'
    alias egrep='egrep --color=auto'
fi

# Colored GCC warnings and errors
export GCC_COLORS='error=01;31:warning=01;35:note=01;36:caret=01;32:locus=01:quote=01'

# Enhanced prompt with git branch and exit status
parse_git_branch() {
    git branch 2>/dev/null | grep '^*' | sed 's/* //'
}

# Prompt: [user@host dir] (git_branch) $
PS1='\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]'
PS1="$PS1"'\[\033[01;33m\]$(git branch 2>/dev/null | grep "^*" | sed "s/* / (/")$([ -n "$(git branch 2>/dev/null)" ] && echo ")")\[\033[00m\]'
PS1="$PS1"'\n\$ '

# ============================================================================
# Environment Variables
# ============================================================================
export EDITOR=vim
export VISUAL=vim
export PAGER=less

# Add ~/.local/bin to PATH if it exists
if [ -d "$HOME/.local/bin" ]; then
    export PATH="$HOME/.local/bin:$PATH"
fi

# ============================================================================
# Aliases
# ============================================================================
# Basic aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'

# Safety aliases
alias rm='rm -i'
alias cp='cp -i'
alias mv='mv -i'

# Navigation
alias ..='cd ..'
alias ...='cd ../..'
alias ....='cd ../../..'

# DevOps tools
alias k='kubectl'
alias h='helm'
alias tf='terraform'
alias d='docker'
alias dc='docker-compose'

# Git shortcuts
alias g='git'
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git log --oneline --graph --decorate'

# Check if eza is available, use it instead of ls
if command -v eza &> /dev/null; then
    alias ls='eza --group-directories-first'
    alias ll='eza --group-directories-first -l'
    alias la='eza --group-directories-first -la'
    alias tree='eza --tree'
fi

# Check if bat is available
if command -v bat &> /dev/null; then
    alias cat='bat --paging=never'
fi

# ============================================================================
# Functions
# ============================================================================
# Source utility functions if available
if [ -f "$HOME/.local/bin/utils.sh" ]; then
    source "$HOME/.local/bin/utils.sh"
fi

# Create directory and cd into it
mkcd() {
    mkdir -p "$1" && cd "$1"
}

# Extract various archive types
extract() {
    if [ -f "$1" ]; then
        case "$1" in
            *.tar.bz2)   tar xjf "$1"    ;;
            *.tar.gz)    tar xzf "$1"    ;;
            *.bz2)       bunzip2 "$1"    ;;
            *.rar)       unrar x "$1"    ;;
            *.gz)        gunzip "$1"     ;;
            *.tar)       tar xf "$1"     ;;
            *.tbz2)      tar xjf "$1"    ;;
            *.tgz)       tar xzf "$1"    ;;
            *.zip)       unzip "$1"      ;;
            *.Z)         uncompress "$1" ;;
            *.7z)        7z x "$1"       ;;
            *)           echo "'$1' cannot be extracted via extract()" ;;
        esac
    else
        echo "'$1' is not a valid file"
    fi
}

# Quick search in history
hs() {
    history | grep "$@"
}

# Find process by name
psgrep() {
    ps aux | grep -v grep | grep -i -e VSZ -e "$@"
}

# ============================================================================
# Tool Integrations
# ============================================================================
# zoxide (smart cd)
if command -v zoxide &> /dev/null; then
    eval "$(zoxide init bash)"
fi

# direnv (auto-load .envrc)
if command -v direnv &> /dev/null; then
    eval "$(direnv hook bash)"
fi

# fzf (fuzzy finder)
if command -v fzf &> /dev/null; then
    if [ -f ~/.fzf.bash ]; then
        source ~/.fzf.bash
    fi
fi

# Starship prompt (if installed, overrides default prompt)
if command -v starship &> /dev/null; then
    eval "$(starship init bash)"
fi

# ============================================================================
# Kubernetes Context in Prompt
# ============================================================================
kube_ps1() {
    if command -v kubectl &> /dev/null; then
        local context=$(kubectl config current-context 2>/dev/null)
        [ -n "$context" ] && echo " [☸ $context]"
    fi
}

# Add kube context to prompt if not using starship
if ! command -v starship &> /dev/null; then
    PS1="$PS1"'\[\033[01;36m\]$(kube_ps1)\[\033[00m\]\n\$ '
fi

# ============================================================================
# Custom Configurations
# ============================================================================
# Source local customizations if they exist
if [ -f ~/.bashrc.local ]; then
    source ~/.bashrc.local
fi
