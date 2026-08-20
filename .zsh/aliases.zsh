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
alias dch="dotf doctor"                           # Repo/deploy drift check now lives in dotf doctor (CLI-019)
alias cl="changelog-gen.sh"                       # Regenerate CHANGELOG.md from git log
alias profile-shell="shell-profile.sh"            # Measure shell startup (use --detail for per-function breakdown via zprof/xtrace)

# OpenCode (primary AI coding agent). `oc`/`ocfull` moved to .zsh/functions.sh
# (REFACTOR-010, shared bash/zsh core). `oclog` is zsh-only (not duplicated in bash).
alias oclog='tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)" | grep --line-buffered -vE "file\.watcher\.updated|bus type=message\.part\.delta"'  # live tail of newest opencode log, filtered

# qq / qf: zsh quick-question wrappers. `_qq_call` lives in .zsh/functions.sh
# (REFACTOR-010, shared with bash). Wrapped in `noglob` so `qq por que tardas?`
# works without quotes -- zsh would otherwise glob-expand the trailing `?`.
#   qq -> nan/qwen3.6           (default daily, multilingual, ES-friendly, 262K ctx)
#   qf -> nan/deepseek-v4-flash (long-context 500K, autocomplete / large transforms)
alias qq='noglob _qq_call nan/qwen3.6 qq'
alias qf='noglob _qq_call nan/deepseek-v4-flash qf'

# dbg: deepseek-v4-flash con reasoning chain VISIBLE (opencode TUI lo oculta
# porque NaN devuelve el campo en `reasoning_content` non-OpenAI). Usa
# scripts/nan-debug.sh que parsea + colorea reasoning aparte del answer.
alias dbg='noglob nan-debug.sh'

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

# Modern Python (uv) shortcuts
alias uvr="uv run"
alias uvt="uv run pytest"
alias uvl="uv run ruff check"

# Go shortcuts
alias gtr="go test -v -race ./..."
alias gci="golangci-lint run"

# Platform & Cloud shortcuts
alias tk="poetry run toolkit"
alias sops-edit="sops"

# Astro web shortcuts
alias astrod="npm run dev"
alias astrob="npm run build"
alias astroc="npx astro check"