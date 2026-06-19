# Optional startup profiling. Enable with: DOTFILES_PROFILE=1 zsh -i -c exit
# Pairs with the zprof dump at the bottom of this file.
[[ -n "${DOTFILES_PROFILE:-}" ]] && zmodload zsh/zprof

# ==========================
#       OH MY ZSH SETUP
# ==========================
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="robbyrussell"
plugins=(git)

# Load Oh My Zsh
source $ZSH/oh-my-zsh.sh

# ==========================
#       ENVIRONMENT
# ==========================
export EDITOR=nano
# Base Directories (ADR-023): sourced from the generated paths.sh — source of
# truth is env-contract.json (defaults) + ~/.config/dotfiles/machine.json
# (per-machine overrides), rendered by `dotf env generate`. DOTFILES_DIR is
# bootstrapped first because it locates the file. Must be set before secrets.
export DOTFILES_DIR="${DOTFILES_DIR:-$HOME/.dotfiles}"
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
# COPILOT_HOME / OPENCODE_HOME now come from the ADR-023 cascade above
# (sourced paths.sh, or the bootstrap fallback) — the old unconditional
# exports here clobbered that, so they were removed.

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
# Start with system paths or current path
# export PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# Prepend Tool Paths (priority over system)
export PATH="$JAVA_HOME/bin:$PATH"
export PATH="$MAVEN_HOME/bin:$PATH"
export PATH="$PYTHON_HOME/bin:$PATH"
export PATH="$MINIKUBE_HOME:$PATH"
export PATH="$GO_HOME/bin:$PATH"
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
alias obsidian='obsidian --no-sandbox'

# Gemini Helper Function
function gp() {
    local prompt_file="$HOME/.gemini/prompts/$1.md"
    shift
    if [ ! -f "$prompt_file" ]; then
        echo "❌ Error: Prompt not found at $prompt_file"
        return 1
    fi
    agy -i "$(cat "$prompt_file")"$'\n\n'"$*"
}

# Claude Code - use slash commands inside session:
#   claude
#   > /audit src/auth.py
#   > /refactor this function

# ==========================
#    SHELL ENHANCEMENTS
# ==========================
# Load custom functions and scripts
[[ -f ~/.zsh/functions.zsh ]] && source ~/.zsh/functions.zsh
# Portable swiss-army functions (IDEAS-002) — shared with bash (.zsh/functions.sh)
[[ -f ~/.zsh/functions.sh ]] && source ~/.zsh/functions.sh
[[ -f ~/.zsh/nvm.zsh ]] && source ~/.zsh/nvm.zsh

# Initialize tools
command -v direnv >/dev/null && eval "$(direnv hook zsh)"
command -v zoxide >/dev/null && eval "$(zoxide init zsh)"

# Terraform Autocomplete
autoload -U +X bashcompinit && bashcompinit
complete -o nospace -C /usr/bin/terraform terraform

# Dump zprof results at end of startup if profiling is enabled
[[ -n "${DOTFILES_PROFILE:-}" ]] && zprof | head -25

# opencode CLI
export PATH="$HOME/.opencode/bin:$PATH"

# Dotfiles scripts on PATH
export PATH="$HOME/.dotfiles/scripts:$PATH"

# ==========================
# Machine-local overrides (IDEAS-001) — sourced LAST so a machine-specific tweak
# can override anything above. NON-SENSITIVE config only; secrets use the age
# system (sensitive/*.secret.age). See .zshrc.local.example. (gitignored)
[ -r "$HOME/.zshrc.local" ] && [ -f "$HOME/.zshrc.local" ] && . "$HOME/.zshrc.local"
