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
alias g='gemini'
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
    gemini -i "$(cat "$prompt_file")"$'\n\n'"$*"
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
[[ -f ~/.zsh/nvm.zsh ]] && source ~/.zsh/nvm.zsh

# Initialize tools
command -v direnv >/dev/null && eval "$(direnv hook zsh)"
command -v zoxide >/dev/null && eval "$(zoxide init zsh)"

# Terraform Autocomplete
autoload -U +X bashcompinit && bashcompinit
complete -o nospace -C /usr/bin/terraform terraform

# Dump zprof results at end of startup if profiling is enabled
[[ -n "${DOTFILES_PROFILE:-}" ]] && zprof | head -25