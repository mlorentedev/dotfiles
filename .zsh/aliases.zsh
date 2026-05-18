# Custom Aliases
# ==============
# DevOps/Infrastructure tools
alias k="kubectl"             # Kubernetes CLI
alias h="helm"                # Helm package manager
alias tf="terraform"          # Infrastructure as code
alias a="ansible"             # Configuration management
alias ap="ansible-playbook"   # Ansible playbooks

# Project navigation
alias gprj="cd $HOME/Projects"                    # Go to Projects directory
alias gcs="cd $HOME/Projects/cheat-sheets"        # Go to cheat-sheets
alias gbp="cd $HOME/Projects/boilerplates"        # Go to boilerplates

# Enhanced file listing (requires eza)
alias ls="eza --group-directories-first"          # Basic listing with eza
alias ll="eza --group-directories-first -l"       # Long format listing
alias lla="eza --group-directories-first -la"     # Long format with hidden files

# Git shortcuts
alias gs="git status"
alias gd="git diff"
alias gl="git log --oneline -10"
alias gp="git pull"

# AI knowledge maintenance
alias kc="knowledge-crystallize.sh"               # Stamp current project MEMORY.md
alias kca="knowledge-crystallize.sh --all"        # Stamp all projects at once

# Repo maintenance
alias dch="diff-check.sh"                         # Detect drift between repo and ~/.dotfiles
alias cl="changelog-gen.sh"                       # Regenerate CHANGELOG.md from git log

# OpenCode (primary AI coding agent — replaces aider)
alias oc="opencode"                                                                       # TUI: opencode Go subscription
alias oclog='tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)" | grep --line-buffered -vE "file\.watcher\.updated|bus type=message\.part\.delta"'  # live tail of newest opencode log, filtered

# tmux session management
alias tx='tmux new -A -s'         # attach-or-create by name: tx dotfiles
alias txl='tmux ls'               # list sessions
alias txa='tmux a'                # attach to most recent
alias txk='tmux kill-session -t'  # kill named session: txk dotfiles