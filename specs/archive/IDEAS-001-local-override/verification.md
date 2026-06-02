---
tags: [spec, verification, ideas-001]
created: "2026-05-25"
---

# Verification — IDEAS-001-local-override

```bash
~/.local/bin/bats tests/local-override.bats        # 6/6
zsh -n .zshrc && bash -n .bashrc                    # rc files parse
```

| AC | Evidence |
|---|---|
| `.zshrc` ends with guarded source of `~/.zshrc.local` | bats "as the last non-blank line" ✅ |
| `.bashrc` ends with guarded source of `~/.bashrc.local` | bats "as the last non-blank line" ✅ |
| `.gitignore` includes both local files | bats ".gitignore excludes both" ✅ |
| `.zshrc.local.example` + `.bashrc.local.example` with use-cases | bats "example files exist…" ✅ |
| present → sourced (zsh + bash) | bats "guard sources … when present" ✅ |
| absent → graceful no-op (zsh + bash) | bats "guard is a graceful no-op" ✅ |
| doc: `.local` vs age system | added under README "Machine-local overrides" (repo `.claude/CLAUDE.md` is gitignored → README is the tracked home; spec's own completeness note allows "README and/or") ✅ |
| drift detector passes after deployment | **deferred** — verified post-merge + redeploy (drift compares deployed-vs-repo; expected drift until `setup-linux.sh` re-runs). CI has no user `$HOME`, so not gated there. |

Note on test design: the functional ACs exercise the guard's semantics in an
isolated subshell rather than sourcing the full `.zshrc` (which pulls in plugins
/ tools and is environment-fragile in CI). The structural test proves the real
rc carries the exact guarded line as its last statement; together they are
equivalent to "sourcing rc honors the override" without the fragility.

## Out of scope (tracked)

- **IDEAS-001b** — Windows `profile.ps1` equivalent (`Test-Path` idiom). Windows session.
- `.shellrc.local` cross-shell file — deferred to user feedback (proposal R3).
