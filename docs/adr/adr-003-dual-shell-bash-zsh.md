---
id: "dotfiles-adr-003-dual-shell-bash-zsh"
type: adr
adr: "003"
title: Dual Shell Support (Bash + Zsh)
tags: [adr, dotfiles, shell, bash, zsh]
status: accepted
created: "2026-02-22"
owner: manu
---

# ADR-003: Dual Shell Support (Bash + Zsh)

## Context

The dotfiles originally targeted bash only. When zsh became the primary interactive shell (default on macOS, preferred on Linux), scripts started failing silently due to bash-specific syntax.

Key incompatibilities discovered:

| Bash-ism | Failure in zsh | Fix |
|----------|---------------|-----|
| `echo -e "\033[32m..."` | Prints literal `-e` flag | `printf '%b' "..."` |
| `&>/dev/null` | Syntax error in strict mode | `>/dev/null 2>&1` |
| `${BASH_SOURCE[0]}` | Empty string | `${BASH_SOURCE[0]:-$0}` |
| `declare -g VAR` | Not supported | `eval "VAR=value"` |
| `((count++))` with `set -e` | Exits when count=0 | `count=$((count + 1))` |
| `${!var}` (indirect) | Different syntax | Branch: bash `${!var}` / zsh `${(P)var}` |

All scripts are sourced in both `.bashrc` and `.zshrc`, so they must work in both shells.

## Decision

All shell scripts must be compatible with both bash and zsh. A prohibited patterns table is enforced in `.claude/CLAUDE.md` and verified by ShellCheck + BATS tests in CI.

## Consequences

### Positive

- **Works everywhere:** Same scripts on macOS (zsh default), Linux (bash or zsh), and CI (bash)
- **Caught early:** ShellCheck flags most bash-isms; BATS tests run in both shells (106 tests)
- **Documented patterns:** `.claude/CLAUDE.md` has an explicit prohibited/replacement table
- **Lessons captured:** Each incompatibility is recorded in `lessons.md` with context and rule

### Negative

- **POSIX subset:** Can't use convenient bash-only features (associative arrays, `mapfile`, process substitution)
- **Indirect expansion complexity:** `${!var}` vs `${(P)var}` requires shell detection branches in `utils.sh`
- **Testing cost:** Every script change requires verifying in both shells

### Mitigations

- CI runs both `shellcheck` and `bats` on every push/PR
- `utils.sh` contains the shell-detection logic centrally (other scripts just source it)
- `.claude/CLAUDE.md` prohibited patterns table prevents AI-generated bash-isms
