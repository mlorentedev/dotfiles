# shellcheck shell=sh
# ~/.zshenv: the one file every zsh reads, interactive or not, so `zsh -c` from
# an agent, a script or an ssh command gets the same PATH head as a terminal
# (#2013 W2b). Keep it to that: everything else belongs in ~/.zshrc, which only
# interactive shells read, and anything slow here slows every zsh.
# shellcheck source=/dev/null
if [ -f "$HOME/.zsh/path.sh" ]; then . "$HOME/.zsh/path.sh"; fi
