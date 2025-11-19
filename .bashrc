# ~/.bashrc: executed by bash(1) for non-login shells.

# ==========================
#    EXIT IF NOT INTERACTIVE
# ==========================
[[ $- != *i* ]] && return

# ==========================
#      HISTORY SETTINGS
# ==========================
HISTCONTROL=ignoreboth
HISTSIZE=1000
HISTFILESIZE=2000

# ==========================
#      LESS ENHANCEMENTS
# ==========================
command -v lesspipe &>/dev/null && eval "$(SHELL=/bin/sh lesspipe)"

# ==========================
#      PROMPT CONFIGURATION
# ==========================
case "$TERM" in
    xterm-color|*-256color) color_prompt=yes ;;
esac

if [[ $color_prompt == yes ]]; then
    PS1='${debian_chroot:+($debian_chroot)}\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ '
else
    PS1='${debian_chroot:+($debian_chroot)}\u@\h:\w\$ '
fi
unset color_prompt

# Set terminal title for xterm-like terminals
case "$TERM" in
    xterm*|rxvt*) 
        PS1="\[\e]0;${debian_chroot:+($debian_chroot)}\u@\h: \w\a\]$PS1" 
        ;;
esac

# ==========================
#      COLOR SUPPORT
# ==========================
if command -v dircolors &>/dev/null; then
    eval "$(dircolors -b ~/.dircolors 2>/dev/null || dircolors -b)"
    alias ls='ls --color=auto'
    alias grep='grep --color=auto'
    alias fgrep='fgrep --color=auto'
    alias egrep='egrep --color=auto'
fi

# ==========================
#       USEFUL ALIASES
# ==========================
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
alias python=python3
alias docker-compose='docker compose'

# ==========================
#   AI
# ==========================
alias g='gemini'

function gp() {
    local prompt_name="$1"
    local prompt_file="$HOME/.gemini/prompts/${prompt_name}.md"
    shift # Remove prompt name from arguments

    if [ ! -f "$prompt_file" ]; then
        echo "❌ Error: Prompt '$prompt_name' not found in ~/.gemini/prompts/"
        echo "Available prompts:"
        ls -1 ~/.gemini/prompts/ | sed 's/\.md//'
        return 1
    fi

    # Pass the content of the prompt file as system instruction
    local full_prompt="$(cat "$prompt_file")"$'\n\n'"$*"
    gemini -i "$full_prompt"
}

# Load custom aliases if available
[[ -f ~/.bash_aliases ]] && source ~/.bash_aliases

# ==========================
#      AUTOCOMPLETION
# ==========================
if [[ -r /usr/share/bash-completion/bash_completion ]]; then
    source /usr/share/bash-completion/bash_completion
fi

# ==========================
#    ENVIRONMENT VARIABLES
# ==========================
# Development Tools Home
export APPS_HOME="$HOME/Apps"

# Define individual tool homes
export JAVA_HOME="$APPS_HOME/jdk-21.0.4"
export MAVEN_HOME="$APPS_HOME/apache-maven-3.9.4"
export PYTHON_HOME="$APPS_HOME/python-3.12.6"
export RUBY_HOME="$APPS_HOME/ruby-3.1.4"
export GEM_HOME="$RUBY_HOME/gems"
export MINIKUBE_HOME="$APPS_HOME/minikube-1.34.0"
export GO_HOME="$APPS_HOME/go-1.23.1"

# ==========================
#       PATH CONFIGURATION
# ==========================
# Ensure system paths are first
export PATH="/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin"

# Add development tool paths
export PATH="$PATH:$JAVA_HOME/bin"
export PATH="$PATH:$MAVEN_HOME/bin"
export PATH="$PATH:$PYTHON_HOME/bin"
export PATH="$PATH:$RUBY_HOME/bin"
export PATH="$PATH:$GEM_HOME/bin"
export PATH="$PATH:$MINIKUBE_HOME"
export PATH="$PATH:$GO_HOME/bin"

# Add personal script directories
export PATH="$PATH:$HOME/.console-ninja/.bin"
export PATH="$PATH:$HOME/.dotfiles/scripts"

# Debugging PATH function
path_info() {
    echo "=== CURRENT PATH ==="
    echo "$PATH" | tr ':' '\n'
    
    echo -e "\n=== COMMAND LOCATIONS ==="
    for cmd in java mvn python ruby gem go; do
        location=$(command -v "$cmd" 2>/dev/null)
        if [[ -n "$location" ]]; then
            echo "$cmd: $location"
        else
            echo "$cmd: Not found"
        fi
    done
}
