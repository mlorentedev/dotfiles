#!/bin/bash
# ~/.claude/boost.sh

alias cc="claude"
alias cca="claude analyze . --all"
alias ccf="claude fix"
alias ccg="claude generate"
alias cct="claude test"

# Auto-context
export CLAUDE_CONTEXT="README.md,**/*.yaml,**/Dockerfile"

# Pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
claude lint --staged --fix
claude test --changed
claude security --staged
EOF

chmod +x .git/hooks/pre-commit