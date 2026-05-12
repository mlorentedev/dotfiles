# Custom Functions
# ================

# Display terminal color palette (useful for theming)
function colormap() {
  for i in {0..255}; do print -Pn "%K{$i}  %k%F{$i}${(l:3::0:)i}%f " ${${(M)$((i%6)):#3}:+$'\n'}; done
}

# sshmux: SSH to a host and attach-or-create a tmux session there.
# Usage: sshmux <host> [session_name]   (session defaults to "main")
sshmux() {
    if [ -z "${1:-}" ]; then
        printf 'Usage: sshmux <host> [session_name]\n' >&2
        return 1
    fi
    ssh -t "$1" "tmux new-session -A -s ${2:-main}"
}

# Source utility functions from dotfiles
if [ -f "$HOME/.dotfiles/scripts/utils.sh" ]; then
    source "$HOME/.dotfiles/scripts/utils.sh"
fi