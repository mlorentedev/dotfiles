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
alias profile-shell="shell-profile.sh"            # Measure shell startup (use --detail for per-function breakdown via zprof/xtrace)

# OpenCode (primary AI coding agent — replaces aider)
alias oc="opencode"                                                                       # TUI: opencode Go subscription
alias oclog='tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)" | grep --line-buffered -vE "file\.watcher\.updated|bus type=message\.part\.delta"'  # live tail of newest opencode log, filtered

# qq / qf: cross-platform one-shot quick-question wrappers. Each invocation is
# a fresh session; for follow-ups use `opencode run -c` directly or the TUI.
#   qq -> qwen3.6-plus     (multilingual, ES-friendly, balanced)
#   qf -> deepseek-v4-flash (faster, never-rate-limited per opencode-go docs)
# Aliases are wrapped in `noglob` so `qq por que tardas tanto?` works without
# quotes -- zsh would otherwise try to glob-expand the trailing `?`.
_qq_call() {
  local model="$1" name="$2"; shift 2
  [ $# -eq 0 ] && { printf 'usage: %s <consulta libre>\n' "$name" >&2; return 1; }
  opencode run -m "$model" "$*"
}
alias qq='noglob _qq_call opencode-go/qwen3.6-plus qq'
alias qf='noglob _qq_call opencode-go/deepseek-v4-flash qf'

# GitHub Copilot CLI v2 (BUG-003: standalone agentic CLI, replaces ghcs/ghce wrappers)
# cop  -> interactive agent (tool use requires confirmation, safe default)
# cops -> single-shot non-interactive prompt with --allow-all-tools (required by CLI for -p mode)
alias cop="copilot"
cops() { copilot -p "$*" --allow-all-tools -s; }

# tmux session management
alias tx='tmux new -A -s'         # attach-or-create by name: tx dotfiles
alias txl='tmux ls'               # list sessions
alias txa='tmux a'                # attach to most recent
alias txk='tmux kill-session -t'  # kill named session: txk dotfiles