# Dotfiles - Lessons Learned

> Patterns learned, mistakes avoided, and best practices discovered during development.
>
> **Protocol**: Update after every correction or important discovery.

---

## Format

```markdown
### [YYYY-MM-DD] Title

**Context**: What I was doing
**Problem**: What went wrong or what I discovered
**Solution**: How I resolved it
**Rule**: Pattern to follow going forward
```

---

## Entries

### [2025-12-15] echo -e breaks in zsh

**Context**: Shell scripts used `echo -e "\033[32mDone\033[0m"` for colored output
**Problem**: zsh does not support the `-e` flag on `echo`. Color codes printed as literal text instead of being interpreted.
**Solution**: Replace all `echo -e` with `printf '%b'` which handles escape sequences portably.
**Rule**: Never use `echo -e`. Always use `printf '%b' "..."` for colored or escaped output.

### [2025-12-15] &>/dev/null is not POSIX

**Context**: Scripts used `&>/dev/null` to suppress both stdout and stderr
**Problem**: `&>` is a bash-ism. Not POSIX-compliant and can fail in strict zsh configurations or when scripts are sourced in unexpected contexts.
**Solution**: Replace all `&>/dev/null` with `>/dev/null 2>&1` across every script.
**Rule**: Always use `>/dev/null 2>&1` for output suppression. Never use `&>`.

### [2025-12-15] ((count++)) exits with code 1 when count is 0

**Context**: Counter variables used `((count++))` inside scripts with `set -e`
**Problem**: In bash, `((0))` evaluates to false and returns exit code 1. When `count=0`, `((count++))` evaluates the pre-increment value (0), triggering `set -e` to abort the script.
**Solution**: Replace `((count++))` with `count=$((count + 1))`. The `$((...))` form always returns exit code 0.
**Rule**: Never use `((...))` for arithmetic that might evaluate to 0 under `set -e`. Use `var=$((expr))` assignment form instead.

### [2025-12-15] ${BASH_SOURCE[0]} is empty in zsh

**Context**: Scripts used `${BASH_SOURCE[0]}` to determine their own file path for relative directory resolution
**Problem**: zsh does not populate `BASH_SOURCE`. Scripts that relied on it for path resolution silently got empty strings, breaking relative path calculations.
**Solution**: Use `${BASH_SOURCE[0]:-$0}` which falls back to `$0` (populated by zsh) when `BASH_SOURCE` is empty.
**Rule**: Always use `${BASH_SOURCE[0]:-$0}` when a script needs to know its own path. Test path resolution in both bash and zsh.

---

*More entries will be added as the project evolves.*
