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
    xterm*|rxvt*) PS1="\[\e]0;${debian_chroot:+($debian_chroot)}\u@\h: \w\a\]$PS1" ;;
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
alias setup-gh-secrets="$HOME/.dotfiles/scripts/setup-gh-secrets.sh"

# Load custom aliases if available
[[ -f ~/.bash/bash_aliases ]] && source ~/.bash/bash_aliases

# ==========================
#      AUTOCOMPLETION
# ==========================
if [[ -r /usr/share/bash-completion/bash_completion ]]; then
    source /usr/share/bash-completion/bash_completion
fi

# ==========================
#    ENVIRONMENT VARIABLES
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

export PATH="$JAVA_HOME/bin:$MAVEN_HOME/bin:$PYTHON_HOME/bin:$RUBY_HOME/bin:$GEM_HOME/bin:$MINIKUBE_HOME:$GO_HOME/bin:/usr/bin:/bin:$NINJA_HOME/bin:$DOTFILES_DIR/scripts:$PATH"
