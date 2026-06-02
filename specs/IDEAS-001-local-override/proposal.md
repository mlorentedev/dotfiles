---
id: "IDEAS-001-local-override"
type: spec
status: active
created: "2026-05-25"
tags: [spec, proposal, ideas-001, shell, dotfiles-survey, tier-1]
template_version: "1.0"
---

# IDEAS-001: Local override pattern

> **Naming**: file lives at `<repo>/specs/IDEAS-001-local-override/proposal.md`.
> Origin: dotfiles-survey research (worktree `research+dotfiles-survey`), Tier-1 idea (#1) from mathiasbynens (`.extra`) + holman (`gitconfig.local.symlink`).

## Why

<!-- from research/dotfiles-survey.md §"Top 6 ideas a aplicar" #1: .extra/.local override pattern, Tier-1 ROI -->

Hardcoding machine-specific paths/aliases (work VPN routes, `/mnt/...` on one VM only, host-specific GPU flags) in versioned `.zshrc`/`.bashrc` forces a bad choice: commit private state (security risk + visual noise in `git status`) or fork the dotfiles per machine (drift). The age system in `sensitive/` is for **secrets** — encrypted, decrypted at login, mapped to env vars. It is *not* the right tool for non-sensitive but machine-local shell config (an alias that only makes sense on the laptop, a `PATH` prepend for a manually-installed tool on a VM). Both mathiasbynens (`.extra`) and holman (`gitconfig.local.symlink`) solve this with a gitignored override file sourced LAST so it can override anything earlier. If we don't ship this, the next "but this command only runs on my Wayland desktop" addition either pollutes the repo or stays uncommitted in a one-off file the user forgets about.

## What

Two trailing source lines, one each at the END of `.zshrc` and `.bashrc` (deployed via symlink by `setup-linux.sh`):

```bash
# Machine-local overrides — last so they can override anything above.
[ -r "$HOME/.zshrc.local" ] && [ -f "$HOME/.zshrc.local" ] && . "$HOME/.zshrc.local"
```

Concrete additions:

1. Trailing source-with-guard in `.zshrc` and `.bashrc`.
2. `.zshrc.local` and `.bashrc.local` added to `.gitignore`.
3. A committed `.zshrc.local.example` (and `.bashrc.local.example`) showing 2-3 common use-cases (host-specific PATH prepend, VM-only alias, work-only env var).
4. Documentation in `.claude/CLAUDE.md` "Common Workflows" section explaining when to use `.local` overrides vs the age secrets system.

The mechanism is purely additive: shell behavior unchanged when no `.local` files exist.

## Out of scope

- **Windows PowerShell equivalent** — `profile.ps1` has different idioms (no `[ -r ]` test, uses `Test-Path`). Tracked as IDEAS-001b for the Windows VM session.
- **Migrating existing secrets to this mechanism** — the age system stays canonical for anything sensitive. `.local` is for *non-sensitive* machine-local tweaks. The CLAUDE.md doc update must make this distinction crystal-clear so users don't accidentally put API keys in `.zshrc.local`.
- **`.gitconfig.local`** — holman's pattern for git author overrides is git-specific and orthogonal; if desired, opened separately. This spec covers only the shell-rc pattern.

## Risks / open questions

- **R1**: Source order. The `.local` file MUST load LAST so it can override prompt, aliases, exports, functions, etc. Confirm both `.zshrc` and `.bashrc` end with the source line — no late `setopt` or prompt re-init after it that could clobber user overrides. Verification: read both files post-change, ensure the source line is the final non-blank line.
- **R2**: Drift detector interaction. The repo's drift detector (PR #93 / BUG-024) watches `.zshrc` and `.bashrc` byte-for-byte against the repo copies. Adding the trailing source line modifies both files in the repo. The drift baseline updates automatically because the file *in repo* changes (drift detector compares deployed-vs-repo). Confirm post-change that `drift-detector` exit code is 0.
- **R3 (open question, non-blocking)**: Should we provide a third file `.shellrc.local` sourced by BOTH `.zshrc` AND `.bashrc` for cross-shell overrides? Pro: less duplication for users who maintain parity. Con: extra file, complicates the model. **Recommendation: defer to user-feedback. Ship only `.zshrc.local` + `.bashrc.local` first; add `.shellrc.local` only if a real use-case emerges.**

## Acceptance criteria

- [ ] `.zshrc` ends with a conditional source of `$HOME/.zshrc.local` guarded by `[ -r ] && [ -f ]`.
- [ ] `.bashrc` ends with a conditional source of `$HOME/.bashrc.local` guarded by `[ -r ] && [ -f ]`.
- [ ] `.gitignore` includes `.zshrc.local` and `.bashrc.local` (and any sibling local files added).
- [ ] `.zshrc.local.example` and `.bashrc.local.example` exist in the repo with at least 2 commented use-cases each.
- [ ] Bats test: when `~/.zshrc.local` exists with `export TEST_IDEAS001_LOCAL=42`, a zsh subshell sourcing `.zshrc` exports `$TEST_IDEAS001_LOCAL=42`.
- [ ] Bats test: when `~/.zshrc.local` does NOT exist, sourcing `.zshrc` exits 0 with no errors (graceful no-op via the guard).
- [ ] Same two bats tests for `.bashrc.local` in bash.
- [ ] Drift detector (`drift-detector.sh` or equivalent) passes after deployment.

## Completeness review

Standard items considered:

- **Rate limit / cost guard** — N/A (local shell mechanism, no API).
- **Idempotency** — yes; the `[ -r ] && [ -f ]` guard makes repeat sourcing safe and the absence of the file a graceful no-op.
- **Regression test** — covered by bats tests for both present and absent cases.
- **Cert provisioning** — N/A.
- **Rollback** — single-commit revert restores prior state; `.local` files left on disk are harmless (won't be sourced by reverted rc files).

Adding (not in template, load-bearing here):

- **Onboarding doc**: README and/or `.claude/CLAUDE.md` must document when to use `.local` vs the age secrets system. Without this, the pattern degrades into "users put their secrets in `.local`" anti-use over time.
- **Cross-OS gap note**: this spec is Linux-only. IDEAS-001b (Windows mirror via `profile.ps1`) tracked separately.

## References

- Research source: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #1 (Tier 1).
- mathiasbynens pattern: `.bash_profile` itera con `for file in ~/.{path,bash_prompt,exports,aliases,functions,extra}; do [ -r ] && [ -f ] && source "$file"; done` — `.extra` cargado al final.
- holman pattern: `git/gitconfig.local.symlink.example` committed, `gitconfig.local.symlink` gitignored.
- Related: IDEAS-003 (sourcing loop refactor) makes this trivial to integrate cleanly.
- Distinct from: `sensitive/env-mapping.conf` + age (that's for **secrets**, this is for **non-sensitive machine-local config**).

## LOC estimate

~10 LOC source lines + 2 lines `.gitignore` + ~30 LOC example files + ~40 LOC bats = **~80 LOC total**. Borderline `skip-sdd` candidate, but specced for traceability to the survey research.
