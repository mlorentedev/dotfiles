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
# Base Directories (ADR-025): sourced from the generated paths.sh — source of
# truth is env-contract.json (defaults) + ~/.config/dotfiles/machine.json
# (per-machine overrides), rendered by `dotf env generate`. DOTFILES_DIR is
# bootstrapped first because it locates the file. Must be set before secrets.
export DOTFILES_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"
# Zero-touch: auto-render paths.sh on first run when missing and dotf is on PATH
# (a fresh machine self-configures without a manual `dotf env generate`). Silent +
# best-effort; the bootstrap fallback below covers any failure.
if [ ! -f "$DOTFILES_DIR/paths.sh" ] && command -v dotf >/dev/null 2>&1; then
    dotf env generate >/dev/null 2>&1 || true
fi
if [ -f "$DOTFILES_DIR/paths.sh" ]; then
    . "$DOTFILES_DIR/paths.sh"
else
    # Bootstrap fallback (paths.sh not generated yet): contract defaults inline,
    # conditional so a later source wins. A relocated repo/vault is picked up
    # once `dotf env generate` renders machine.json.
    export DOTFILES_REPO_DIR="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"
    export CLAUDE_CONFIG_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
    export SCRIPTS_DIR="${SCRIPTS_DIR:-$DOTFILES_DIR/scripts}"
    export AGY_HOME="${AGY_HOME:-$HOME/.gemini/antigravity-cli}"
    export COPILOT_HOME="${COPILOT_HOME:-$HOME/.copilot}"
    export OPENCODE_HOME="${OPENCODE_HOME:-$HOME/.config/opencode}"
fi

# AI provider endpoints — NaN community (primary, OpenAI-compatible).
# API key in $NAN_API_KEY (loaded by load-secrets.sh from sensitive/nan.api-key.secret.age).
export NAN_BASE_URL="https://api.nan.builders/v1"

# Load encrypted secrets as environment variables
[[ -f "$DOTFILES_DIR/scripts/load-secrets.sh" ]] && source "$DOTFILES_DIR/scripts/load-secrets.sh"
export APPS_HOME="$HOME/Applications"
export NINJA_HOME="$HOME/.console-ninja"

# Tool Versions (single source of truth)
[[ -f "$DOTFILES_DIR/versions.conf" ]] && . "$DOTFILES_DIR/versions.conf"

# Tool Homes (constructed from versions.conf)
export JAVA_HOME="$APPS_HOME/jdk-${JAVA_VERSION}"
export MAVEN_HOME="$APPS_HOME/apache-maven-${MAVEN_VERSION}"
export PYTHON_HOME="$APPS_HOME/python-${PYTHON_VERSION}"
export MINIKUBE_HOME="$APPS_HOME/minikube-${MINIKUBE_VERSION}"
export GO_HOME="$APPS_HOME/go-${GO_VERSION}"

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
alias g='agy'
alias c='claude'
alias obsidian='obsidian --no-sandbox'

# Gemini saved-prompt helper `gpr` lives in .zsh/functions.sh (shared bash/zsh,
# sourced below). It was renamed from `gp` to stop colliding with the
# `gp=git pull` git alias.

# qq / qf: bash quick-question wrappers. `_qq_call` lives in .zsh/functions.sh
# (REFACTOR-010, shared with zsh). Bash leaves `foo?` literal when no match
# exists (no zsh-style nomatch error), so no `noglob` wrapper is needed here.
#   qq -> nan/qwen3.6           (default daily, multilingual, 262K ctx)
#   qf -> nan/deepseek-v4-flash (long-context 500K)
qq() { _qq_call nan/qwen3.6 qq "$@"; }
qf() { _qq_call nan/deepseek-v4-flash qf "$@"; }

# oc / ocfull moved to .zsh/functions.sh (REFACTOR-010, shared bash/zsh core).

# dbg: deepseek con reasoning chain VISIBLE (opencode TUI oculta reasoning_content).
# Mirror de .zsh/aliases.zsh. Mismo script subyacente (nan-debug.sh on PATH).
dbg() { nan-debug.sh "$@"; }

# Claude Code - use slash commands inside session:
#   claude
#   > /audit src/auth.py
#   > /refactor this function

# ==========================
#    SHELL ENHANCEMENTS
# ==========================
# Portable swiss-army functions (IDEAS-002) — shared with zsh (.zsh/functions.sh)
[ -f ~/.zsh/functions.sh ] && . ~/.zsh/functions.sh

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

# opencode CLI
export PATH="$HOME/.opencode/bin:$PATH"

# Dotfiles scripts on PATH
export PATH="$HOME/.dotfiles/scripts:$PATH"

# ==========================
# Machine-local overrides (IDEAS-001) — sourced LAST so a machine-specific tweak
# can override anything above. NON-SENSITIVE config only; secrets use the age
# system (sensitive/*.secret.age). See .bashrc.local.example. (gitignored)
[ -r "$HOME/.bashrc.local" ] && [ -f "$HOME/.bashrc.local" ] && . "$HOME/.bashrc.local"

# ==========================
#    SSH AGENT (userspace, opt-in)
# ==========================
# Unlock a passphrase-protected key once per boot via a userspace ssh-agent,
# reused across interactive Git Bash sessions. For boxes where the OS ssh-agent
# isn't usable (e.g. non-admin Windows: the ssh-agent service is admin-gated).
# Opt-in: set DOTFILES_SSH_AGENT_KEY in ~/.bashrc.local (sourced just above).
# Serves Git Bash's ssh, NOT the Windows ssh.exe used by PowerShell.
if [ -n "${DOTFILES_SSH_AGENT_KEY:-}" ] && [ -f "${DOTFILES_SSH_AGENT_KEY}" ]; then
    _agent_env="$HOME/.ssh/agent.env"
    [ -f "$_agent_env" ] && . "$_agent_env" >/dev/null
    ssh-add -l >/dev/null 2>&1; _agent_state=$?   # 0=has keys 1=no keys 2=no agent
    if [ "$_agent_state" -eq 2 ]; then
        (umask 077; ssh-agent -s >"$_agent_env"); . "$_agent_env" >/dev/null
        ssh-add "${DOTFILES_SSH_AGENT_KEY}"
    elif [ "$_agent_state" -eq 1 ]; then
        ssh-add "${DOTFILES_SSH_AGENT_KEY}"
    fi
    unset _agent_env _agent_state
fi
