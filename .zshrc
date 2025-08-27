# ==========================
#       OH MY ZSH SETUP
# ==========================
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="robbyrussell"
plugins=(git)

# Load Oh My Zsh
source $ZSH/oh-my-zsh.sh

# ==========================
#        ALIASES
# ==========================
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias python=python3
alias docker-compose='docker compose'
alias setup-gh-secrets="$HOME/.dotfiles/scripts/github-secrets-manager.sh"

# Load custom aliases if they exist
[[ -f ~/.zsh/aliases.zsh ]] && source ~/.zsh/aliases.zsh

# ==========================
#      CUSTOM SCRIPTS
# ==========================
[[ -f ~/.zsh/functions.zsh ]] && source ~/.zsh/functions.zsh
[[ -f ~/.zsh/nvm.zsh ]] && source ~/.zsh/nvm.zsh

# ==========================
#      SHELL ENHANCEMENTS
# ==========================
eval "$(direnv hook zsh)"   # Load Direnv
eval "$(zoxide init zsh)"   # Load zoxide

# ==========================
#    OPTIONAL CONFIGURATION
# ==========================
# Uncomment to enable features:
# CASE_SENSITIVE="true"                # Case-sensitive completion
# HYPHEN_INSENSITIVE="true"            # Hyphen and underscore interchangeable
# ENABLE_CORRECTION="true"             # Enable command auto-correction
# COMPLETION_WAITING_DOTS="true"       # Show waiting dots on completion
# DISABLE_AUTO_TITLE="true"            # Disable auto-setting terminal title
# DISABLE_UNTRACKED_FILES_DIRTY="true" # Speed up large Git repos
# HIST_STAMPS="yyyy-mm-dd"             # Change history timestamp format
# ZSH_CUSTOM=/path/to/custom-folder    # Change custom folder location

# ==========================
#       ENVIRONMENT
# ==========================
export JAVA_HOME=$HOME/Apps/jdk-21.0.4
export MAVEN_HOME=$HOME/Apps/apache-maven-3.9.4
export PYTHON_HOME=$HOME/Apps/python-3.12.6
export RUBY_HOME=$HOME/Apps/ruby-3.1.4
export GEM_HOME=$RUBY_HOME/gems
export MINIKUBE_HOME=$HOME/Apps/minikube-1.34.0
export GO_HOME=$HOME/Apps/go-1.23.1
export NINJA_HOME=$HOME/.console-ninja
export DOTFILES_DIR=$HOME/.dotfiles

export PATH=$HOME/.local/bin:$JAVA_HOME/bin:$MAVEN_HOME/bin:$PYTHON_HOME/bin:$RUBY_HOME/bin:$GEM_HOME/bin:$MINIKUBE_HOME:$GO_HOME/bin:/usr/bin:/bin:$NINJA_HOME/bin:$DOTFILES_DIR/scripts:$PATH