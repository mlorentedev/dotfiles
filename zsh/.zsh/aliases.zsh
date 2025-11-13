# Custom Aliases
# ==============
# DevOps/Infrastructure tools
alias k="kubectl"             # Kubernetes CLI
alias h="helm"                # Helm package manager
alias tf="terraform"          # Infrastructure as code
alias a="ansible"             # Configuration management
alias ap="ansible-playbook"   # Ansible playbooks

# Project navigation
alias gprj="cd $HOME/Projects"                    # Go to Projects directory
alias gcs="cd $HOME/Projects/cheat-sheets"        # Go to cheat-sheets
alias gbp="cd $HOME/Projects/boilerplates"        # Go to boilerplates

# Enhanced file listing (requires eza)
alias ls="eza --group-directories-first"          # Basic listing with eza
alias ll="eza --group-directories-first -l"       # Long format listing
alias lla="eza --group-directories-first -la"     # Long format with hidden files