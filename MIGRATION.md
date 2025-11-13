# Migration Guide

This guide helps you migrate from the old flat dotfiles structure to the new GNU Stow-based modular structure.

## Overview

**Old Structure** (Flat):
```
dotfiles/
├── .bashrc
├── .zshrc
├── .gitconfig
├── .zsh/
├── scripts/
└── install.sh
```

**New Structure** (Stow Modules):
```
dotfiles/
├── bash/
│   ├── .bashrc
│   └── .bash_profile
├── zsh/
│   ├── .zshrc
│   └── .zsh/
├── git/
│   └── .gitconfig
├── scripts/
│   └── .local/bin/
└── bootstrap.sh
```

## Benefits of Migration

- **Modular**: Install only what you need
- **Testable**: Comprehensive test suite with CI/CD
- **Automated**: Tool installation included
- **Standard**: Follows XDG Base Directory specification
- **Safe**: Backup creation and conflict detection

## Migration Steps

### Step 1: Backup Current Setup

```bash
# Backup existing dotfiles
mkdir -p ~/dotfiles-backup-$(date +%Y%m%d)
cp ~/.bashrc ~/dotfiles-backup-$(date +%Y%m%d)/ 2>/dev/null || true
cp ~/.zshrc ~/dotfiles-backup-$(date +%Y%m%d)/ 2>/dev/null || true
cp ~/.gitconfig ~/dotfiles-backup-$(date +%Y%m%d)/ 2>/dev/null || true
cp -r ~/.zsh ~/dotfiles-backup-$(date +%Y%m%d)/ 2>/dev/null || true

echo "Backup created in: ~/dotfiles-backup-$(date +%Y%m%d)"
```

### Step 2: Clone New Repository

```bash
# If you haven't already, clone to a new location
cd ~
git clone https://github.com/mlorentedev/dotfiles dotfiles-new
cd dotfiles-new
```

### Step 3: Review Changes

```bash
# Check what's different
make check          # See what will be installed
cat README.md       # Read the new documentation
ls -la bash/        # Examine the bash module
ls -la zsh/         # Examine the zsh module
```

### Step 4: Remove Old Symlinks

If you used the old install.sh:

```bash
# Remove old symlinks
rm ~/.bashrc ~/.zshrc ~/.gitconfig 2>/dev/null || true

# Or if they're real files, back them up first
for file in .bashrc .zshrc .gitconfig; do
    if [ -f ~/$file ] && [ ! -L ~/$file ]; then
        mv ~/$file ~/${file}.old
    fi
done
```

### Step 5: Install New Structure

```bash
# Minimal installation
make minimal

# Or install with tools
make tools

# Or install everything
make all
```

### Step 6: Verify Installation

```bash
# Run tests
make test

# Check symlinks
ls -la ~/.bashrc ~/.zshrc

# Verify PATH
echo $PATH | tr ':' '\n' | grep .local/bin

# Test shell
bash -c "alias ll"  # Should show alias
zsh -c "alias k"    # Should show kubectl alias
```

### Step 7: Migrate Custom Configurations

If you had custom configurations in the old setup:

#### For Bash Customizations

```bash
# Create local override file
vim ~/.bashrc.local

# Add your custom aliases, functions, etc.
```

#### For Zsh Customizations

```bash
# Create local override file
vim ~/.zshrc.local

# Add your custom aliases, functions, etc.
```

#### For Git Customizations

```bash
# Create local git config
vim ~/.gitconfig.local

# Add machine-specific git config
```

### Step 8: Migrate Scripts

If you had custom scripts in the old `scripts/` directory:

```bash
# Copy custom scripts to new location
cp ~/dotfiles-old/scripts/my-custom-script.sh ~/.local/bin/my-custom-script
chmod +x ~/.local/bin/my-custom-script
```

### Step 9: Clean Up

```bash
# Once everything works, you can remove the old dotfiles
# (Keep the backup just in case!)
mv ~/dotfiles ~/dotfiles-old-$(date +%Y%m%d)
mv ~/dotfiles-new ~/dotfiles

# Update any aliases or scripts that reference the old location
```

## Common Migration Scenarios

### Scenario 1: You Have Many Custom Aliases

**Old way** (mixed in .zshrc):
```zsh
# In .zshrc
alias myproject='cd ~/work/my-project'
alias deploy='./deploy.sh'
```

**New way** (in local override):
```bash
# Create ~/.zshrc.local
cat >> ~/.zshrc.local << 'EOF'
alias myproject='cd ~/work/my-project'
alias deploy='./deploy.sh'
EOF
```

### Scenario 2: You Have Environment Variables

**Old way** (in .zshrc):
```zsh
export MY_API_KEY="secret"
export MY_PROJECT_PATH="/path/to/project"
```

**New way** (use direnv or local override):
```bash
# Option 1: Use direnv for project-specific vars
cd ~/work/my-project
cat > .envrc << 'EOF'
export MY_API_KEY="secret"
export MY_PROJECT_PATH="$PWD"
EOF
direnv allow

# Option 2: Use ~/.zshrc.local for global vars
echo 'export MY_PROJECT_PATH="/path/to/project"' >> ~/.zshrc.local

# Option 3: Use secrets-wrapper for sensitive data
```

### Scenario 3: You Have Custom Functions

**Old way** (in .zshrc):
```zsh
myfunc() {
    echo "Hello $1"
}
```

**New way** (in functions file or local override):
```bash
# Option 1: Add to zsh/.zsh/functions.zsh
vim ~/.dotfiles/zsh/.zsh/functions.zsh

# Option 2: Add to ~/.zshrc.local
cat >> ~/.zshrc.local << 'EOF'
myfunc() {
    echo "Hello $1"
}
EOF
```

### Scenario 4: You Have Secrets

**Old way** (various locations):
```bash
# Secrets scattered in various files
```

**New way** (centralized with encryption):
```bash
# Use the secrets-wrapper
secrets-wrapper encrypt

# Or setup direnv integration
secrets-wrapper setup-direnv
```

## Verification Checklist

After migration, verify:

- [ ] All aliases work: `alias` command shows your aliases
- [ ] All functions work: Try running your custom functions
- [ ] PATH is correct: `echo $PATH` includes `~/.local/bin`
- [ ] Shell prompt works: Git branch, k8s context shown
- [ ] Scripts are executable: `which my-script` finds your scripts
- [ ] Tests pass: `make test` completes successfully
- [ ] No broken symlinks: `find ~ -maxdepth 1 -type l -exec test ! -e {} \; -print`

## Rollback Plan

If something goes wrong:

```bash
# Stop using new dotfiles
cd ~/.dotfiles
make clean

# Restore from backup
cp ~/dotfiles-backup-TIMESTAMP/.bashrc ~/.bashrc
cp ~/dotfiles-backup-TIMESTAMP/.zshrc ~/.zshrc
cp ~/dotfiles-backup-TIMESTAMP/.gitconfig ~/.gitconfig

# Restart shell
exec $SHELL
```

## Troubleshooting

### Problem: Stow complains about conflicts

**Solution**:
```bash
# Backup and remove conflicting files
make backup
rm ~/.bashrc ~/.zshrc  # etc.

# Then retry
make minimal
```

### Problem: Aliases don't work after migration

**Solution**:
```bash
# Restart shell
exec $SHELL

# Or source config manually
source ~/.bashrc  # for bash
source ~/.zshrc   # for zsh
```

### Problem: Scripts not found in PATH

**Solution**:
```bash
# Verify .local/bin is in PATH
echo $PATH | grep .local/bin

# If not, add to shell config
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc

# Restart shell
exec $SHELL
```

### Problem: Oh My Zsh not working

**Solution**:
```bash
# Install Oh My Zsh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# Or use bootstrap which installs it automatically
./bootstrap.sh --minimal
```

## Getting Help

If you encounter issues:

1. Check the [README.md](README.md) for detailed documentation
2. Run `make check` to verify dependencies
3. Run `make test` to see if tests pass
4. Open an issue on GitHub with details

## Post-Migration Recommendations

After successful migration:

1. **Test thoroughly**: Use your shell for a day to ensure everything works
2. **Update documentation**: Document any custom configurations
3. **Setup CI/CD**: If you forked, enable GitHub Actions
4. **Regular updates**: Run `make update` periodically
5. **Explore new features**: Try out the new tools (eza, bat, fzf, etc.)

## Key Differences Reference

| Feature | Old | New |
|---------|-----|-----|
| Structure | Flat | Modular (Stow) |
| Installation | `install.sh` | `bootstrap.sh` or `make minimal` |
| Scripts location | `~/dotfiles/scripts` | `~/.local/bin` |
| Customization | Edit main files | Use `.local` override files |
| Tool installation | Manual | Automated (`make tools`) |
| Testing | None | Comprehensive suite |
| CI/CD | None | GitHub Actions |

## Migration Complete

Once you've verified everything works, you now have a modern, tested, and automated dotfiles setup.
