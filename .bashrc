# ~/.bashrc: executed by bash(1) for non-login shells.

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
# Base Directories (must be set before loading secrets)
export DOTFILES_DIR="$HOME/.dotfiles"

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
# Ensure system paths are present
export PATH="/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin"

# Append Tool Paths
export PATH="$PATH:$JAVA_HOME/bin"
export PATH="$PATH:$MAVEN_HOME/bin"
export PATH="$PATH:$PYTHON_HOME/bin"
export PATH="$PATH:$RUBY_HOME/bin"
export PATH="$PATH:$GEM_HOME/bin"
export PATH="$PATH:$MINIKUBE_HOME"
export PATH="$PATH:$GO_HOME/bin"
export PATH="$PATH:$HOME/go/bin"
export PATH="$PATH:$NINJA_HOME/.bin"
export PATH="$PATH:$DOTFILES_DIR/scripts"

# Prepend User Local Bin
export PATH="$HOME/.local/bin:$PATH"

# ==========================
#        ALIASES
# ==========================
if [ -f ~/.bash_aliases ]; then
    source ~/.bash_aliases
fi

# AI Tool Aliases
alias g='gemini'
alias c='claude'

function gp() {
    local prompt_file="$HOME/.gemini/prompts/$1.md"
    shift
    if [ ! -f "$prompt_file" ]; then
        echo "❌ Error: Prompt not found at $prompt_file"
        return 1
    fi
    gemini -i "$(cat "$prompt_file")"$'\n\n'"$*"
}

function cp() {
    local prompt_file="$HOME/.claude/prompts/$1.md"
    shift
    if [ ! -f "$prompt_file" ]; then
        echo "❌ Error: Prompt not found at $prompt_file"
        return 1
    fi
    claude -i "$(cat "$prompt_file")"$'\n\n'"$*"
}

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