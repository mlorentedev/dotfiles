#!/bin/bash
# ~/.claude/boost.sh

alias cc="claude-code"
alias cca="claude-code analyze . --all"
alias ccf="claude-code fix"
alias ccg="claude-code generate"
alias cct="claude-code test"

# Auto-context
export CLAUDE_CONTEXT="README.md,**/*.yaml,**/Dockerfile"

# Pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
claude-code lint --staged --fix
claude-code test --changed
claude-code security --staged
EOF

chmod +x .git/hooks/pre-commit