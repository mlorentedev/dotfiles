---
id: "IDEAS-003-sourcing-loop"
type: spec
status: abandoned
created: "2026-05-25"
tags: [spec, proposal, ideas-003, shell, refactor, dotfiles-survey, tier-1]
template_version: "1.0"
---

# IDEAS-003: Sourcing loop refactor

> **ABANDONED 2026-06-05 — premise invalid against current code; superseded by REFACTOR-010.**
> The spec assumed `.zshrc` and `.bashrc` carried parallel contiguous `source` blocks
> over a shared file set, so a single brace-expanded loop would deduplicate them. Verified
> reality before implementing: (1) `.zshrc` sourcing is intentionally scattered across two
> semantic sections (`aliases.zsh` in ALIASES at L86, the rest in SHELL ENHANCEMENTS at
> L115-118), so a single loop would reorder `aliases.zsh` relative to the inline AI aliases —
> a behavioral change the spec itself forbids; (2) `.bashrc` sources exactly ONE `.zsh/` file
> (`functions.sh`, L157) — bash and zsh deliberately diverge (bash uses `~/.bash_aliases` +
> inline wrappers), so there is no parallel duplication to remove; (3) the graceful-missing
> tolerance the loop would add already exists (every line has its own `[[ -f ]]` guard).
> Net effect of implementing as specced: reorder risk for zero/negative benefit (cf. POLISH-001
> WONTFIX; lesson "verify-before-act on agent audits"). The genuine duplication is elsewhere —
> the `_qq_call`/`oc`/`ocfull` wrappers shared bash↔zsh — captured in REFACTOR-010. See
> `verification.md` for the full finding.

> **Naming**: file lives at `<repo>/specs/IDEAS-003-sourcing-loop/proposal.md`.
> Origin: dotfiles-survey research, Tier-1 idea (#3) from mathiasbynens (`.bash_profile` source loop).

## Why

<!-- from research/dotfiles-survey.md §"Top 6 ideas a aplicar" #3: sourcing loop with `[ -r ]` guards, Tier-1 ROI -->

`.zshrc` and `.bashrc` currently source `.zsh/*.zsh` files via explicit blocks like `source ~/.zsh/aliases.zsh`, `source ~/.zsh/exports.zsh`. Adding a new concern (IDEAS-002 `functions.zsh`) or an optional one (IDEAS-001 `.zshrc.local`) means editing the block in two files in parallel. mathiasbynens's pattern compresses this into:

```bash
for f in $HOME/.zsh/{path,exports,aliases,functions,prompt,completions}.zsh; do
  [ -r "$f" ] && [ -f "$f" ] && . "$f"
done
unset f
```

Benefits: (1) tolerates missing files (a future `prompt.zsh` not yet committed sources cleanly when absent), (2) one place to add a new concern, (3) idiomatic to the canonical `.bash_profile` shape. If we don't ship this, IDEAS-001 and IDEAS-002 each have to add explicit source lines in two rc files — gradually re-adding the verbose pattern this refactor eliminates.

## What

Replace the current explicit `source` blocks in `.zshrc` and `.bashrc` with a single brace-expanded loop, preserving exact current ordering. Files in the loop list MUST match the current set; this is a pure refactor with zero behavioral change beyond the `[ -r ]` graceful-missing tolerance.

Pseudo-shape:

```bash
# Existing line block:
#   source ~/.zsh/path.zsh
#   source ~/.zsh/aliases.zsh
#   source ~/.zsh/exports.zsh
#   source ~/.zsh/functions.zsh    # if IDEAS-002 has landed
#   source ~/.zsh/prompt.zsh
#   source ~/.zsh/completions.zsh

# Becomes:
for f in $HOME/.zsh/{path,aliases,exports,functions,prompt,completions}.zsh; do
  [ -r "$f" ] && [ -f "$f" ] && . "$f"
done
unset f
```

The set of files in the brace expansion is the FROZEN INPUT to the spec — match current contents exactly. Reordering or adding/removing files is out of scope for this PR.

## Out of scope

- **PowerShell `profile.ps1` refactor** — has different idioms (`Test-Path` + `. file.ps1`); separate spec IDEAS-003b if desired.
- **`load-secrets.sh` invocation timing** — that call stays explicit, NOT in the loop. The loop is for `.zsh/*.zsh` source files only.
- **Changing the set of sourced files** — pure refactor, no add/remove.
- **Migrating the explicit set into a manifest file** — possible future, but this PR keeps the brace expansion inline (auditable in one place).

## Risks / open questions

- **R1**: brace-expansion order across shells. Bash and zsh both expand `{a,b,c}` left-to-right deterministically — verify in both via explicit bats test (`echo {a,b,c} | tr ' ' '\n'`). If a non-default `setopt` somehow disabled this in zsh, the parity test catches it.
- **R2**: `$f` namespace leak. The trailing `unset f` covers it, but if the user already has `$f` defined (unlikely, but possible) we destroy it. Mitigation: `unset f 2>/dev/null` or use a less common variable name (`__src_f`). Decision: use `__src_f` — clear intent, unlikely to collide.
- **R3 (drift detector)**: as with IDEAS-001, this modifies `.zshrc` and `.bashrc`. The drift detector compares deployed-vs-repo; since BOTH change atomically (same commit), no drift is introduced. Confirm with explicit post-deploy run.
- **R4 (performance)**: `for` + 6-7 stat calls per shell startup. Negligible (~1ms), but verify with `profile-shell` before/after — must stay within ±10% of pre-refactor baseline.
- **R5 (parity)**: post-refactor, the env state (exported vars, aliases, functions) MUST be identical to pre-refactor. Bats test: dump `compgen -A export`, `alias`, `compgen -A function` pre and post, assert sets match.

## Acceptance criteria

- [ ] `.zshrc` and `.bashrc` both use a single brace-expanded `for` loop for sourcing `.zsh/*.zsh` files.
- [ ] Each loop iteration is guarded by `[ -r "$__src_f" ] && [ -f "$__src_f" ]` (or similar non-colliding var name).
- [ ] Trailing `unset __src_f` (or equivalent) — no namespace leak.
- [ ] Bats parity test: pre/post `compgen -A export` + `alias` + `compgen -A function` sets are identical (no env state regression).
- [ ] `profile-shell` shows shell startup time within ±10% of pre-refactor baseline (both bash and zsh).
- [ ] Drift detector exits 0 post-deploy.
- [ ] No existing bats test regresses (full matrix runs green).

## Completeness review

Standard items considered:

- **Rate limit / cost guard** — N/A.
- **Idempotency** — sourcing the same files via loop is idempotent (same semantics as explicit `source`).
- **Regression test** — covered by parity test (criterion 4) + existing bats matrix.
- **Cert provisioning** — N/A.
- **Rollback** — single-commit revert.

Adding (not in template, load-bearing here):

- **Performance baseline capture** — `profile-shell` BEFORE any change must be recorded in the PR description for the ±10% threshold to be enforceable. Take 5-run min/median/mean/max per shell.
- **Drift detector co-update**: the drift baseline (if hash-pinned) updates in the same commit that changes the rc files. Verify drift-detector exit 0 *after* deploy.

## References

- Research source: `research/dotfiles-survey.md` § "Top 6 ideas a aplicar" #3 (Tier 1).
- Upstream: mathiasbynens/dotfiles `.bash_profile` source loop.
- Related: IDEAS-001 (local override) — benefits from this pattern; could extend the loop to include `~/.zshrc.local` if desired (though the spec keeps that line explicit and trailing).
- Related: IDEAS-002 (shell functions) — adds `functions.zsh` to the loop's input set.
- Project rule: `.claude/CLAUDE.md` Prohibited Patterns — no `source` shorthand; use `.` per POSIX rule.

## LOC estimate

~10 LOC refactor in `.zshrc` + ~10 LOC in `.bashrc` + ~50 LOC bats parity test = **~70 LOC total**. Borderline `skip-sdd` candidate, but specced for traceability and to lock the parity-test invariant.
