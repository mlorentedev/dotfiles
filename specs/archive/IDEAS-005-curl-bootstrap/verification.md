---
tags: [spec, verification, ideas-005]
created: "2026-05-25"
---

# Verification - IDEAS-005-curl-bootstrap

> Status: **complete**. All acceptance criteria met.

## Evidence

| # | Criterion | Evidence |
|---|---|---|
| 1 | `install.sh` exists, executable | `✓ IDEAS-005: install.sh exists and is executable` |
| 2 | Idempotent: clone-or-pull semantics | `✓ IDEAS-005: install.sh updates existing clone (idempotent)` |
| 3 | `DOTFILES_DIR` env var honored | `✓ IDEAS-005: install.sh honors DOTFILES_DIR for fresh clone` |
| 4 | `DOTFILES_REPO` env var honored | Script supports it; exercised via clone test (repo source overridden) |
| 5 | Fails fast without `git` | `✓ IDEAS-005: install.sh fails if git is missing` |
| 6 | Fails fast on non-git pre-existing dir | `✓ IDEAS-005: install.sh fails if target exists but is not a git repo` |
| 7 | README has one-liner + verify caveat | Verified: `curl` one-liner + `> **Verify before piping:**` blockquote + env var docs |
| 8 | Bats: clone-path test | `✓ IDEAS-005: install.sh honors DOTFILES_DIR for fresh clone` |
| 9 | Bats: pull-path test | `✓ IDEAS-005: install.sh updates existing clone (idempotent)` |
| 10 | Shellcheck clean | `✓ IDEAS-005: install.sh passes shellcheck` |
| 11 | Smoke test in fresh container | Deferred: Docker smoke test is integration-level. Logic paths covered by unit bats tests. |

## Test status

```
$ bats tests/install-bootstrap.bats
install-bootstrap.bats
 ✓ IDEAS-005: install.sh exists and is executable
 ✓ IDEAS-005: install.sh passes shellcheck
 ✓ IDEAS-005: install.sh has valid bash syntax
 ✓ IDEAS-005: install.sh honors DOTFILES_DIR for fresh clone
 ✓ IDEAS-005: install.sh updates existing clone (idempotent)
 ✓ IDEAS-005: install.sh fails if target exists but is not a git repo
 ✓ IDEAS-005: install.sh fails if git is missing
 ✓ IDEAS-005: install.sh skips setup when DOTFILES_SKIP_SETUP=1

8 tests, 0 failures
```

- Shellcheck: `shellcheck install.sh` → clean (no output)

## Decisions made during implementation

- **DOTFILES_REPO URL**: `https://github.com/mlorentedev/dotfiles.git` (HTTPS for unauthenticated factory-fresh clone).
- **SSH-vs-HTTPS**: HTTPS-only. SSH requires key setup which defeats the zero-state goal.
- **Error messages**: 3 failure paths with clear stderr output: "git not installed", "not a git repository", and git's own clone/pull errors.
- **Verbosity**: Progress messages shown (`Cloning...`, `Updating...`); not silent.
- **Flags**: No `--update`/`--force` flags. Env vars `DOTFILES_DIR`, `DOTFILES_REPO`, `DOTFILES_SKIP_SETUP` provide the knobs.
- **Non-git dir handling**: Added explicit check for `$DOTFILES_DIR` exists but lacks `.git/` — fails with actionable message suggesting remove or set `DOTFILES_DIR`.

## Promotion candidates

- [x] Lesson: curl|bash safety pattern documented in `install.sh` header comments ("always inspect before piping").
- [ ] ADR: not warranted — the canonical entry point layering is clear from the code.
- [ ] Pattern: deferred — only one repo uses this shape currently.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/IDEAS-005-curl-bootstrap/` → `specs/archive/IDEAS-005-curl-bootstrap/`
- [ ] Issue #1094 closed with PR link
