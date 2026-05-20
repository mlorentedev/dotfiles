# ~/.bashrc: executed by bash(1) for non-login shells.

# Optional startup profiling. Enable with: DOTFILES_PROFILE=1 bash -i -c exit
# Bash has no built-in profiler — emit timestamped xtrace to a tmp file.
if [[ -n "${DOTFILES_PROFILE:-}" ]]; then
    PS4='+ $EPOCHREALTIME ${BASH_SOURCE}:${LINENO}: '
    exec 3>&2 2>"/tmp/bashrc-profile.$$.log"
    set -x
fi

# ==========================
#    INTERACTIVE CHECK
# ==========================
[[ $- != *i* ]] && return

# ==========================
#    HISTORY & DISPLAY
# ==========================
HISTCONTROL=ignoreboth
HISTSIZE=1000
HISTFILESIZE=2000
shopt -s histappend
shopt -s checkwinsize

# Prompt Configuration
case "$TERM" in
    xterm-color|*-256color) color_prompt=yes ;;
esac

if [[ $color_prompt == yes ]]; then
    PS1='${debian_chroot:+($debian_chroot)}\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
else
    PS1='${debian_chroot:+($debian_chroot)}\u@\h:\w\$ '
fi
unset color_prompt

# Color Support
if command -v dircolors >/dev/null; then
    eval "$(dircolors -b ~/.dircolors 2>/dev/null || dircolors -b)"
    alias ls='ls --color=auto'
    alias grep='grep --color=auto'
fi
# ==========================
#       ENVIRONMENT
# ==========================
export EDITOR=nano
# Base Directories (must be set before loading secrets)
export DOTFILES_DIR="$HOME/.dotfiles"
export DOTFILES_REPO_DIR="$HOME/Projects/dotfiles"
export CLAUDE_CONFIG_DIR="$HOME/.claude"
# Per-agent install dirs + scripts deploy target (REFACTOR-002).
# Declared in env-contract.json; doctor.{sh,ps1} validates on every run.
export SCRIPTS_DIR="$DOTFILES_DIR/scripts"
export GEMINI_HOME="$HOME/.gemini"
export COPILOT_HOME="$HOME/.copilot"
export OPENCODE_HOME="$HOME/.config/opencode"

# Load encrypted secrets as environment variables
[[ -f "$DOTFILES_DIR/scripts/load-secrets.sh" ]] && source "$DOTFILES_DIR/scripts/load-secrets.sh"
export APPS_HOME="$HOME/Applications"
export NINJA_HOME="$HOME/.console-ninja"

# Tool Versions (single source of truth)
[[ -f "$DOTFILES_DIR/versions.conf" ]] && . "$DOTFILES_DIR/versions.conf"

# Tool Homes (constructed from versions.conf)
export JAVA_HOME="$APPS_HOME/jdk-${JAVA_VERSION:-21.0.4}"
export MAVEN_HOME="$APPS_HOME/apache-maven-${MAVEN_VERSION:-3.9.4}"
export PYTHON_HOME="$APPS_HOME/python-${PYTHON_VERSION:-3.12.6}"
export MINIKUBE_HOME="$APPS_HOME/minikube-${MINIKUBE_VERSION:-1.34.0}"
export GO_HOME="$APPS_HOME/go-${GO_VERSION:-1.26.0}"

# ==========================
#    PATH CONFIGURATION
# ==========================
# Ensure system paths are present
export PATH="/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin"

# Prepend Tool Paths (priority over system)
export PATH="$JAVA_HOME/bin:$PATH"
export PATH="$MAVEN_HOME/bin:$PATH"
export PATH="$PYTHON_HOME/bin:$PATH"
export PATH="$MINIKUBE_HOME:$PATH"
export PATH="$GO_HOME/bin:$PATH"
export PATH="$HOME/go/bin:$PATH"
export PATH="$NINJA_HOME/.bin:$PATH"
export PATH="$DOTFILES_DIR/scripts:$PATH"

# Prepend User Local Bin (highest priority)
export PATH="$HOME/.local/bin:$PATH"

# bun
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"



# ==========================
#        ALIASES
# ==========================
if [ -f ~/.bash_aliases ]; then
    source ~/.bash_aliases
fi



# AI Tool Aliases
alias g='gemini'
alias c='claude'
alias obsidian='obsidian --no-sandbox'

function gp() {
    local prompt_file="$HOME/.gemini/prompts/$1.md"
    shift
    if [ ! -f "$prompt_file" ]; then
        echo "❌ Error: Prompt not found at $prompt_file"
        return 1
    fi
    gemini -i "$(cat "$prompt_file")"$'\n\n'"$*"
}

# qq / qf: one-shot opencode wrappers. Mirrors .zsh/aliases.zsh for bash users.
# Bash leaves `foo?` literal when no match exists (no zsh-style nomatch error),
# so no `noglob` wrapper is needed here.
#   qq -> qwen3.6-plus     (multilingual, ES-friendly, balanced)
#   qf -> deepseek-v4-flash (faster, never-rate-limited per opencode-go docs)
_qq_call() {
    local model="$1" name="$2"; shift 2
    [ $# -eq 0 ] && { printf 'usage: %s <consulta libre>\n' "$name" >&2; return 1; }
    opencode run -m "$model" "$*"
}
qq() { _qq_call opencode-go/qwen3.6-plus qq "$@"; }
qf() { _qq_call opencode-go/deepseek-v4-flash qf "$@"; }

# Claude Code - use slash commands inside session:
#   claude
#   > /audit src/auth.py
#   > /refactor this function

# ==========================
#    SHELL ENHANCEMENTS
# ==========================
# Enable direnv and zoxide
command -v direnv >/dev/null && eval "$(direnv hook bash)"
command -v zoxide >/dev/null && eval "$(zoxide init bash)"

# Bash Completion
if [ -f /usr/share/bash-completion/bash_completion ]; then
    . /usr/share/bash-completion/bash_completion
fi
complete -o nospace -C /usr/bin/terraform terraform

# Stop xtrace and surface the profile log path if profiling was enabled
if [[ -n "${DOTFILES_PROFILE:-}" ]]; then
    set +x
    exec 2>&3 3>&-
    echo "bashrc profile written to /tmp/bashrc-profile.$$.log"
fi