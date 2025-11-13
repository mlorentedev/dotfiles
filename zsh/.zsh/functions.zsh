# Custom Functions
# ================

# Display terminal color palette (useful for theming)
function colormap() {
  for i in {0..255}; do print -Pn "%K{$i}  %k%F{$i}${(l:3::0:)i}%f " ${${(M)$((i%6)):#3}:+$'\n'}; done
}

# Source utility functions from dotfiles
if [ -f "$HOME/.dotfiles/scripts/utils.sh" ]; then
    source "$HOME/.dotfiles/scripts/utils.sh"
fi