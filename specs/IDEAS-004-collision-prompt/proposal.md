---
id: "IDEAS-004-collision-prompt"
type: spec
status: draft
created: "2026-05-25"
tags: [spec, proposal, ideas-004, setup, ux, dotfiles-survey, tier-2]
template_version: "1.0"
---

# IDEAS-004: Interactive collision prompt on symlink deploy

> **Naming**: file lives at `<repo>/specs/IDEAS-004-collision-prompt/proposal.md`.
> Origin: dotfiles-survey research, Tier-2 idea (#4) from holman (`script/bootstrap` collision prompt).

## Why

<!-- from research/dotfiles-survey.md §"Top 6 ideas a aplicar" #4: holman collision prompt with [s]/[o]/[b]/[*-all] modes, Tier-2 ROI -->

When `setup-linux.sh` deploys symlinks and a destination already exists as a regular file (e.g., a fresh OS install with default `~/.bashrc`, or a re-run after a manual edit), the current behavior is heuristic: in some places it overwrites silently, in others it aborts. holman's `script/bootstrap` solves this with a four-option interactive prompt:

```
File already exists: ~/.zshrc, what do you want to do?
[s]kip, [S]kip all, [o]verwrite, [O]verwrite all, [b]ackup, [B]ackup all
```

This is materially better UX for users running setup on a machine with existing dotfiles — they see what would be lost and choose per-file (or globally via `*-all`). The `*-all` modes are the load-bearing UX: nobody wants to answer 30 prompts for 30 files. Without this, the choice is "destructive overwrite" or "abort and fix manually" — both worse than the third option of "backup first, proceed".

## What

A new helper `prompt_collision()` in `scripts/utils.sh`:

- Inputs: destination path, source path
- Behavior: prints colliding file info + the 6-choice prompt; reads a single character
- Returns: choice as enum-string (`skip`, `overwrite`, `backup`)
- State: persists `skip_all` / `overwrite_all` / `backup_all` flags via a global (or returned tuple) so subsequent collisions short-circuit
- Force mode: when `DOTFILES_SETUP_FORCE=1`, skip prompt entirely and apply configurable default (default: `backup` — safest)
- Backup naming: `~/.zshrc.bak.<UTC-timestamp>` to avoid clobbering prior backups

Integration:

- `link_file()` (or equivalent symlink helper used by `setup-linux.sh`) calls `prompt_collision()` on collision instead of its current heuristic.
- CI test suite sets `DOTFILES_SETUP_FORCE=1` so bats integration tests don't hang.
- Bats unit tests stdin-mock the 6 paths (`s`, `S`, `o`, `O`, `b`, `B`) against `prompt_collision()` directly.

## Out of scope

- **Windows PowerShell mirror** — `setup-windows.ps1` has its own `link_file` equivalent and PowerShell-idiomatic `Read-Host`. Tracked as IDEAS-004b for the Windows VM session.
- **Redefining "collision"** — current spec triggers only on regular files at the destination. Symlinks pointing elsewhere are still handled by existing logic (typically: remove + re-link). Out of scope to change that.
- **Bulk migration of existing user dotfiles** — this spec changes behavior on collision; it does not proactively scan for collisions outside `setup-linux.sh` invocations.
- **Restoring from backup** — users restore manually (`mv ~/.zshrc.bak.20260525-143022 ~/.zshrc`). A `dotfiles-restore-backup` helper is a separate spec if ever needed.

## Risks / open questions

- **R1 (BLOCKER for CI)**: interactive prompt hangs CI. Mitigation: `DOTFILES_SETUP_FORCE=1` MUST be set in `.github/workflows/*.yml` AND verified by an intentional-collision bats test that asserts force-mode applies the default action without prompting. Without this, the first PR merge to main locks up CI.
- **R2**: `-all` state management. Bash globals are fragile across subshells. Decision: keep state in a single function-scoped global (`__DOTFILES_COLLISION_MODE`) set at the top of `setup-linux.sh` and respected by `prompt_collision()`. If `link_file()` runs in a subshell, the state is lost — confirm it does NOT subshell.
- **R3 (BLOCKER for cross-shell)**: zsh `read -r -k 1` vs bash `read -r -n 1`. Both shells support `read` but flag names differ. Use POSIX `read -r` reading one char via stty manipulation, OR detect shell and branch.
- **R4**: backup naming collision. If two runs happen within the same second (unlikely but possible in test loops), `bak.<timestamp>` could collide. Mitigation: append `$$` (PID) or use higher-resolution `date +%s%N`.
- **R5 (open question, non-blocker)**: what is the default action under `DOTFILES_SETUP_FORCE=1`? Candidates: `overwrite` (matches current implicit behavior, breaks user data), `backup` (safe, leaves orphan files), `skip` (safe, but setup doesn't take effect). **Recommendation: `backup` — non-destructive AND setup proceeds.** Decide before tasks.md freeze.

## Acceptance criteria

- [ ] `prompt_collision()` exists in `scripts/utils.sh` with a documented signature.
- [ ] `link_file()` (or equivalent in `setup-linux.sh`) calls `prompt_collision()` on regular-file destinations.
- [ ] Environment variable `DOTFILES_SETUP_FORCE=1` skips the prompt and applies the chosen default (per R5).
- [ ] Bats test for each of 6 input paths (`s`, `S`, `o`, `O`, `b`, `B`) → expected end-state on FS.
- [ ] Bats test for force-mode: intentional collision + `DOTFILES_SETUP_FORCE=1` → no hang, default action applied.
- [ ] Bats test for backup naming: timestamp suffix prevents collision across rapid invocations.
- [ ] CI workflow updated to set `DOTFILES_SETUP_FORCE=1` for `setup-linux.sh` invocations.
- [ ] Cross-shell: `prompt_collision()` works under both bash and zsh (matrix test).
- [ ] No regressions in existing `setup-linux.sh` integration tests.

## Completeness review

Standard items considered:

- **Rate limit / cost guard** — N/A.
- **Idempotency** — backup mode is idempotent in the sense that re-running creates a new backup; skip mode is idempotent (no state change); overwrite mode is idempotent (final state same). Document that "backup" creates accumulating `.bak.*` files (user can clean periodically).
- **Regression test** — covered by criteria 4-6 and existing integration tests.
- **Cert provisioning** — N/A.
- **Rollback** — single-commit revert reverts both helper and `link_file()` integration; users with `.bak.*` files may want to manually restore.

Adding (not in template, load-bearing here):

- **Documentation**: README has a "First-time setup" section that mentions the prompt and force-mode env var. Without this, users assume setup is destructive and avoid running it.
- **Migration**: existing tests + healthcheck must NOT regress. Run full bats matrix + drift detector pre-merge.

## References

- Research source: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #4 (Tier 2).
- Upstream: holman/dotfiles `script/bootstrap` — `link_files()` collision prompt.
- Project rules: `.claude/CLAUDE.md` Prohibited Patterns — `read` portability call-out.
- Related: IDEAS-005 (curl bootstrap) — first-time `install.sh` invocation will hit collisions on existing dotfiles. Force-mode default needs to be safe for this path.

## LOC estimate

~60 LOC helper + ~80 LOC bats + ~10 LOC README + ~5 LOC CI yml = **~155 LOC total**. Above the 50-LOC threshold; full SDD discipline applies.
