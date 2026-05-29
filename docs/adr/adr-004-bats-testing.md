---
id: "dotfiles-adr-004-bats-testing"
type: adr
adr: "004"
title: BATS for Shell Testing
tags: [adr, dotfiles, testing, bats]
status: accepted
created: "2026-02-22"
owner: manu
---

# ADR-004: BATS for Shell Testing

## Context

The dotfiles contain ~1500 lines of shell code across `utils.sh`, `load-secrets.sh`, `dotfiles-sync.sh`, and setup scripts. Shell functions handle secrets decryption, environment variable loading, file deployment, and bidirectional sync — operations where silent failures cause real damage (missing API tokens, stale configs).

Alternatives evaluated:

| Approach | Pros | Cons |
|----------|------|------|
| Manual testing | No setup needed | Not repeatable, no CI |
| shUnit2 | Mature, xUnit-style | Heavier setup, less readable |
| BATS | TAP output, `@test` syntax, `run` helper | Requires bats-core install |
| ShellCheck only | Catches syntax issues | No runtime behavior testing |

## Decision

Use [BATS](https://github.com/bats-core/bats-core) (Bash Automated Testing System) as the primary testing framework. ShellCheck is used alongside for static analysis, but BATS covers runtime behavior.

BATS installed at `~/.local/bin/bats`. Test files in `tests/*.bats`.

## Consequences

### Positive

- **106 tests covering:** `utils.sh` functions, `load-secrets.sh` commands, env-mapping parsing, file secret deployment, sync logic, edge cases
- **CI integration:** GitHub Actions runs `bats tests/*.bats` + `shellcheck scripts/*.sh` on every push/PR
- **Readable syntax:** `@test "secrets_list shows all mapped secrets"` is self-documenting
- **TAP output:** Standard test protocol, parseable by CI tools
- **Dual-shell coverage:** Tests verify behavior in both bash and zsh contexts

### Negative

- **Requires installation:** `bats-core` not available in all package managers (installed manually to `~/.local/bin/`)
- **Mocking is manual:** No built-in mock/stub framework (tests use temp dirs and fake age keys)
- **Shell-only:** Can't test PowerShell scripts (`setup-windows.ps1`)

### Mitigations

- Setup script could install bats automatically (currently manual)
- Test fixtures use isolated temp directories (`setup()` / `teardown()` in BATS)
- PowerShell testing tracked as separate backlog item (PSScriptAnalyzer)
