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

# Telemetry suppression (prevents agent prompts and unnecessary analytics network calls)
export ASTRO_TELEMETRY_DISABLED=1
export DO_NOT_TRACK=1

# AI provider endpoints — NaN community (primary, OpenAI-compatible).
# API key in $NAN_API_KEY - injected on demand via `dotf secrets run` (see below), not the ambient shell.
export NAN_BASE_URL="https://api.nan.builders/v1"

# Secrets are NOT auto-loaded into the ambient shell (ADR-028 "not always
# exposed"). On demand: `dotf secrets run -- <cmd>` injects the decrypted secrets
# into that child process only, so a key lives in the agent's process and never
# in this shell. Recursion-safe: dotf resolves the real binary on PATH, not this
# function.
#
# Each wrapper is scoped with `--only`, and the scope is load-bearing. The
# unscoped form resolved the WHOLE registry on every launch — deliberate parity
# with the old ambient export, and free while every secret was age-backed, which
# is a local decrypt in milliseconds. After the Bitwarden migration (#961) each
# resolution is a `bw` shell-out at ~1.5s, so an unscoped launch paid ~45s of
# latency before the agent started, and died on the first locked entry even when
# the agent had no use for it (#976). Scoping is least privilege and startup
# time at once.
#
# THE GUARD MUST NOT DEPEND ON $PATH. It runs here, but `~/.local/bin` — where
# dotf lives — is prepended ~40 lines below, and the line before that REPLACES
# PATH outright. So `command -v dotf` succeeds only when the parent process
# happened to export it already.
#
# Measured 2026-08-27: a terminal with a clean PATH skipped this whole block, so
# `pi` and `opencode` were never wrapped, ran without their injected credentials,
# and reported "No models available" — silently, because a guard that fails says
# nothing. A shell that inherited a good PATH worked, which is why it looked
# intermittent and could not be reproduced from an already-configured shell.
#
# Only the GUARD is evaluated at source time; the function BODY runs at call
# time, when PATH is complete and plain `dotf` resolves normally.
if command -v dotf >/dev/null 2>&1 || [ -x "$HOME/.local/bin/dotf" ]; then
    # Every token here must be a live registry id: `--only` fails loud on an
    # unknown one, so a stale name does not degrade the launch, it prevents it.
    # OLLAMA_API_KEY was listed for a provider slot whose registry entry was
    # never created, and opencode had stopped starting on both shells as a result.
    opencode() { dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY,OPENAI_API_KEY -- opencode "$@"; }
    pi() { dotf secrets run --only NAN_API_KEY,OPENROUTER_API_KEY -- pi "$@"; }
    # agy is deliberately NOT wrapped. It authenticates with its own stored
    # credentials and reads no variable this registry exposes — verified against
    # both its settings files and the strings of the binary itself. Wrapping it
    # injected nothing and cost a full registry resolution per launch.
fi
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
# GitHub Copilot CLI: cop -> interactive agent; cops -> single-shot prompt with
# --allow-all-tools (required by the CLI for -p mode). Same pair as aliases.zsh
# and profile.ps1 -- bash had neither (AI-036/#1296).
alias cop='copilot'
cops() { copilot -p "$*" --allow-all-tools -s; }
alias obsidian='obsidian --no-sandbox'

# Gemini saved-prompt helper `agyp` lives in .zsh/functions.sh (shared bash/zsh,
# sourced below). Named out of the `g*` namespace, which oh-my-zsh's git plugin
# owns: the earlier `gp` and `gpr` names both collided with its aliases.

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
