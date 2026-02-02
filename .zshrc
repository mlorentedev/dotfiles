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
# Base Directories (must be set before loading secrets)
export DOTFILES_DIR="$HOME/.dotfiles"
export DOTFILES_REPO_DIR="$HOME/Projects/dotfiles"

# Load encrypted secrets as environment variables
[[ -f "$DOTFILES_DIR/scripts/load-secrets.sh" ]] && source "$DOTFILES_DIR/scripts/load-secrets.sh"
export APPS_HOME="$HOME/Applications"
export NINJA_HOME="$HOME/.console-ninja"

# Tool Homes
export JAVA_HOME="$APPS_HOME/jdk-21.0.4"
export MAVEN_HOME="$APPS_HOME/apache-maven-3.9.4"
export PYTHON_HOME="$APPS_HOME/python-3.12.6"
export RUBY_HOME="$APPS_HOME/ruby-3.1.4"
export GEM_HOME="$RUBY_HOME/gems"
export MINIKUBE_HOME="$APPS_HOME/minikube-1.34.0"
export GO_HOME="$APPS_HOME/go-1.23.1"

# ==========================
#    PATH CONFIGURATION
# ==========================
# Start with system paths or current path
# export PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# Append Tool Paths
export PATH="$PATH:$JAVA_HOME/bin"
export PATH="$PATH:$MAVEN_HOME/bin"
export PATH="$PATH:$PYTHON_HOME/bin"
export PATH="$PATH:$RUBY_HOME/bin"
export PATH="$PATH:$GEM_HOME/bin"
export PATH="$PATH:$MINIKUBE_HOME"
export PATH="$PATH:$GO_HOME/bin"
export PATH="$PATH:$HOME/go/bin"        # Go workspace bin
export PATH="$PATH:$NINJA_HOME/.bin"    # Console Ninja
export PATH="$PATH:$DOTFILES_DIR/scripts"

# Prepend User Local Bin (High Priority)
export PATH="$HOME/.local/bin:$PATH"



# ==========================
#        ALIASES
# ==========================
# Load consolidated aliases
[[ -f ~/.zsh/aliases.zsh ]] && source ~/.zsh/aliases.zsh



# AI Tool Aliases
alias g='gemini'
alias c='claude'

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