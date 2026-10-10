# Optional startup profiling. Enable with: DOTFILES_PROFILE=1 zsh -i -c exit
# Pairs with the zprof dump at the bottom of this file.
[[ -n "${DOTFILES_PROFILE:-}" ]] && zmodload zsh/zprof

# ==========================
#       OH MY ZSH SETUP
# ==========================
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="robbyrussell"
plugins=(git)

# Load Oh My Zsh when it is installed; without it, initialise completion
# directly, so a machine that has not run setup still gets a working shell
# (#2013 W6).
if [[ -r "$ZSH/oh-my-zsh.sh" ]]; then
    source "$ZSH/oh-my-zsh.sh"
else
    autoload -Uz compinit && compinit -C
fi
# Case-insensitive, partial-word completion.
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}' 'r:|[._-]=* r:|=*' 'l:|=* r:|=*'

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
    # Bootstrap fallback (paths.sh not generated yet): contract defaults inline.
    export DOTFILES_REPO_DIR="${DOTFILES_REPO_DIR:-$HOME/Projects/dotfiles}"
    export CLAUDE_CONFIG_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"
    export SCRIPTS_DIR="${SCRIPTS_DIR:-$DOTFILES_DIR/scripts}"
    export AGY_HOME="${AGY_HOME:-$HOME/.gemini/antigravity-cli}"
    export COPILOT_HOME="${COPILOT_HOME:-$HOME/.copilot}"
    export OPENCODE_HOME="${OPENCODE_HOME:-$HOME/.config/opencode}"
fi
export AGY_APP_DATA="$AGY_HOME"
export ANTIGRAVITY_ENDPOINT="https://cloudcode-pa.googleapis.com"
export CLOUDCODE_URL="https://cloudcode-pa.googleapis.com"
export GEMINI_DIR="$HOME/.gemini"
export GEMINI_HOME="$HOME/.gemini"
# Telemetry suppression (prevents agent prompts and unnecessary analytics network calls)
export ASTRO_TELEMETRY_DISABLED=1
export DO_NOT_TRACK=1
# COPILOT_HOME / OPENCODE_HOME now come from the ADR-025 cascade above
# (sourced paths.sh, or the bootstrap fallback) — the old unconditional
# exports here clobbered that, so they were removed.

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
# THE GUARD MUST NOT DEPEND ON $PATH, because it runs ~40 lines before the PATH
# that would satisfy it. `~/.local/bin` — where dotf lives — is prepended further
# down; here, `command -v dotf` succeeds only when the parent process happened to
# export it already.
#
# Measured 2026-08-27: a terminal launched with a clean PATH (a fresh desktop
# terminal, an IDE, an ADE) skipped this whole block, so `pi` and `opencode` were
# never wrapped, ran without their injected credentials, and reported "No models
# available" — with no error, because a guard that fails is silence. A terminal
# that inherited a good PATH worked, which is why it looked intermittent and why
# it could not be reproduced from an already-configured shell.
#
# Checking the known location fixes it without moving the PATH block: only the
# GUARD is evaluated at source time. The function BODY runs at call time, by
# which point PATH is complete and plain `dotf` resolves normally.
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

# Tool Homes (constructed from versions.conf), exported only when the
# directory exists. A JAVA_HOME naming a directory that is not there breaks
# macOS's /usr/bin/java stub, and on a machine whose toolchains come from mise
# (ADR-044) none of these exist (#2013 F-041).
_dotfiles_home() { [[ -d "$2" ]] && export "$1=$2"; }
_dotfiles_home JAVA_HOME "$APPS_HOME/jdk-${JAVA_VERSION}"
_dotfiles_home MAVEN_HOME "$APPS_HOME/apache-maven-${MAVEN_VERSION}"
_dotfiles_home MINIKUBE_HOME "$APPS_HOME/minikube-${MINIKUBE_VERSION}"
_dotfiles_home GO_HOME "$APPS_HOME/go-${GO_VERSION}"
unset -f _dotfiles_home

# ==========================
#    PATH CONFIGURATION
# ==========================
# Start with system paths or current path
# export PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# Homebrew (macOS): its shellenv when brew is installed, Apple Silicon or Intel,
# and not already loaded. A nested shell inherits HOMEBREW_PREFIX with the PATH
# shellenv built, so it does not run brew again.
if [[ -z "${HOMEBREW_PREFIX:-}" ]]; then
    for _brew in /opt/homebrew/bin/brew /usr/local/bin/brew; do
        [[ -x "$_brew" ]] && eval "$("$_brew" shellenv)" && break
    done
    unset _brew
fi

# Prepend Tool Paths (priority over system), for the homes that exist
[[ -n "${JAVA_HOME:-}" ]] && export PATH="$JAVA_HOME/bin:$PATH"
[[ -n "${MAVEN_HOME:-}" ]] && export PATH="$MAVEN_HOME/bin:$PATH"
[[ -n "${MINIKUBE_HOME:-}" ]] && export PATH="$MINIKUBE_HOME:$PATH"
[[ -n "${GO_HOME:-}" ]] && export PATH="$GO_HOME/bin:$PATH"
export PATH="$HOME/go/bin:$PATH"          # Go workspace bin
export PATH="$NINJA_HOME/.bin:$PATH"      # Console Ninja
export PATH="$DOTFILES_DIR/scripts:$PATH"

# Prepend User Local Bin (highest priority)
export PATH="$HOME/.local/bin:$PATH"

# bun completions
[ -s "$HOME/.bun/_bun" ] && source "$HOME/.bun/_bun"

# bun
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"



# ==========================
#        ALIASES
# ==========================
# Load consolidated aliases
[[ -f ~/.zsh/aliases.zsh ]] && source ~/.zsh/aliases.zsh



# AI Tool Aliases
alias g='agy'
alias c='claude'
# --no-sandbox is for the Linux AppImage's Electron sandbox; the macOS CLI
# reads it as a command and fails every call (obsidian 1.14.4). Everywhere
# but darwin, as `dotf vault health` does.
case "$OSTYPE" in darwin*) ;; *) alias obsidian='obsidian --no-sandbox' ;; esac

# Gemini saved-prompt helper `agyp` lives in .zsh/functions.sh (shared bash/zsh,
# sourced below). Named out of the `g*` namespace, which oh-my-zsh's git plugin
# owns: the earlier `gp` and `gpr` names both collided with its aliases.

# Claude Code - use slash commands inside session:
#   claude
#   > /test src/auth.py
#   > /refactor this function

# ==========================
#    SHELL ENHANCEMENTS
# ==========================
# Load custom functions and scripts
[[ -f ~/.zsh/functions.zsh ]] && source ~/.zsh/functions.zsh
# Portable swiss-army functions (IDEAS-002) — shared with bash (.zsh/functions.sh)
[[ -f ~/.zsh/functions.sh ]] && source ~/.zsh/functions.sh
[[ -f ~/.zsh/nvm.zsh ]] && source ~/.zsh/nvm.zsh

# Initialize tools. mise first: the pinned CLIs it installs (direnv, zoxide,
# ...) reach PATH through it (ADR-044, #2013 W2b).
command -v mise >/dev/null && eval "$(mise activate zsh)"
command -v direnv >/dev/null && eval "$(direnv hook zsh)"
command -v zoxide >/dev/null && eval "$(zoxide init zsh)"

# Terraform Autocomplete, from whichever terraform is on PATH (#2013 F-042)
if command -v terraform >/dev/null; then
    autoload -U +X bashcompinit && bashcompinit
    complete -o nospace -C "$(command -v terraform)" terraform
fi

# Dump zprof results at end of startup if profiling is enabled
[[ -n "${DOTFILES_PROFILE:-}" ]] && zprof | head -25

# Dotfiles scripts on PATH
export PATH="$HOME/.dotfiles/scripts:$PATH"

# ==========================
# Machine-local overrides (IDEAS-001) — sourced LAST so a machine-specific tweak
# can override anything above. NON-SENSITIVE config only; secrets use the age
# system (sensitive/*.secret.age). See .zshrc.local.example. (gitignored)
[ -r "$HOME/.zshrc.local" ] && [ -f "$HOME/.zshrc.local" ] && . "$HOME/.zshrc.local"
