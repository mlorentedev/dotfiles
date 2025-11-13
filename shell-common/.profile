# ~/.profile: executed by the command interpreter for login shells.
# This file is not read by bash(1), if ~/.bash_profile or ~/.bash_login
# exists.
# see /usr/share/doc/bash/examples/startup-files for examples.
# the files are located in the bash-doc package.

# the default umask is set in /etc/profile; for setting the umask
# for ssh logins, install and configure the libpam-umask package.
#umask 022

# if running bash
if [ -n "$BASH_VERSION" ]; then
    # include .bashrc if it exists
    if [ -f "$HOME/.bashrc" ]; then
	. "$HOME/.bashrc"
    fi
fi

# set PATH so it includes user's private bin if it exists
if [ -d "$HOME/bin" ] ; then
    PATH="$HOME/bin:$PATH"
fi

# set PATH so it includes user's private bin if it exists
if [ -d "$HOME/.local/bin" ] ; then
    PATH="$HOME/.local/bin:$PATH"
fi

# set Utils file
UTILS_FILE="$HOME/.dotfiles/scripts/utils.sh"
if [ -f "$UTILS_FILE" ]; then
    # Add the sourcing of utils.sh to the .bashrc or .zshrc file
    if ! grep -q "source $UTILS_FILE" "$HOME/.bashrc" && ! grep -q "source $UTILS_FILE" "$HOME/.zshrc"; then
        echo "source $UTILS_FILE" >> "$HOME/.bashrc"
        echo "source $UTILS_FILE" >> "$HOME/.zshrc"
    fi
fi

# set PATH and EDITOR
export PATH="$HOME/bin:$PATH"
export EDITOR="vim"