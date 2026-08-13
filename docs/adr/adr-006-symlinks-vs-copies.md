---
id: "dotfiles-adr-006-symlinks-vs-copies"
type: adr
adr: "006"
title: Symlinks on Linux, Copies on Windows
tags: [adr, dotfiles, cross-platform, windows, linux]
status: accepted
created: "2026-02-22"
owner: manu
---

# ADR-006: Symlinks on Linux, Copies on Windows

> **Superseded by [ADR-012](adr-012-deploy-strategy-copy-with-drift-assertion.md) (accepted).** Linux deploy no longer symlinks — `setup-linux.sh` now uses the same atomic-copy-with-drift-assertion strategy as Windows, for the reasons ADR-012 §Context lays out (cross-agent symlink fragility, silent-drift risk). Symlinks remain only for vault↔home bindings and secret files (SSH key, age key), not for the managed config files this ADR was about.

## Context

The dotfiles must deploy configuration files (shell configs, AI tools, SSH keys) to their expected locations on both Linux/macOS and Windows. The deployment strategy differs because:

| Factor | Linux/macOS | Windows |
|--------|------------|---------|
| Symlink support | Native, no special permissions | Requires Developer Mode or admin |
| Shell | bash/zsh (source from `~/.dotfiles/`) | PowerShell (copies to `$env:USERPROFILE`) |
| File paths | `~/.zshrc`, `~/.config/` | `%USERPROFILE%\.claude\`, `%USERPROFILE%\.ssh\` |
| Admin rights | Not required for symlinks | Often restricted in corporate environments |

## Decision

- **Linux/macOS (`setup-linux.sh`):** Use symlinks. Config files link back to the repo, so edits in either location are the same file.
- **Windows (`setup-windows.ps1`):** Use file copies. No admin rights required, works in restricted environments.

Examples:
```bash
# Linux: symlink
ln -sf "$DOTFILES_DIR/.zshrc" "$HOME/.zshrc"
ln -sf "$DOTFILES_DIR/ssh/config" "$HOME/.ssh/config"

# Windows: copy
Copy-Item "ai\claude\*" "$env:USERPROFILE\.claude\" -Recurse
Copy-Item "ssh\config" "$env:USERPROFILE\.ssh\config"
```

## Consequences

### Positive

- **Linux simplicity:** Symlinks mean zero drift — editing `~/.zshrc` edits the repo file directly
- **Windows compatibility:** No admin rights, no Developer Mode, works in corporate lockdown
- **No dependencies:** Both approaches use built-in OS commands

### Negative

- **Windows drift:** Copied files diverge from repo after edits. Must re-run `setup-windows.ps1` to update.
- **Two setup scripts:** `setup-linux.sh` and `setup-windows.ps1` implement the same logic differently
- **Testing gap:** PowerShell scripts not covered by BATS/ShellCheck (separate linting needed)

### Mitigations

- Windows users re-run `setup-windows.ps1` after pulling updates (documented in README)
- Both scripts register the same MCP servers and deploy the same skills (feature parity verified)
- PowerShell CI validation tracked as backlog item
