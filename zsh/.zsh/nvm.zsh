# NVM (Node Version Manager) Configuration
# -----------------------------------------

# Set NVM directory
export NVM_DIR="$HOME/.nvm"

# This loads nvm
if [ -s "$NVM_DIR/nvm.sh" ]; then
  # Unset any existing nvm alias first to avoid the "defining function based on alias" error
  unalias nvm 2>/dev/null || true
  
  # Load NVM
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
  
  # Load bash completion
  [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
fi