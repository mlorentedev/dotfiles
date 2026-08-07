---
id: "BUG-049-test-sh-repo-root"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-07"
issue: "mlorentedev/dotfiles#794"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-049-test-sh-repo-root

> **Naming**: issue #794 is *titled* `BUG-047`, but that id was already claimed by
> `specs/BUG-047-commit-msg-hook-windows/` (a different problem, landed as #795).
> This spec takes the next free id rather than deepening the collision. The
> id-collision class itself is tracked by #770.

## Why

<!-- from issue #794: BUG-047: the pre-commit test hook tests the deployed tree, not the committed one — and cannot pass on Windows -->

`scripts/test.sh` runs as the `dotfiles-test` pre-commit hook, but it honoured an
ambient `$DOTFILES_DIR` and so asserted against `~/.dotfiles` — the **deploy mirror** —
instead of the tree being committed. On Windows that mirror carries only a subset of
`scripts/` (a single file, `load-secrets.ps1`), so the run died at step 2/15 with
`utils.sh not found` and **blocked every commit on the machine**. CI never saw it
because the variable is unset there, which is exactly why it survived so long.

## What

`scripts/test.sh` tests the tree it ships in, and is runnable as a pre-commit hook on
both platforms. Two concepts that shared one variable are separated:

| | Means | Source |
|---|---|---|
| `REPO_DIR` | the tree **under test** | this script's own repo, always |
| `DOTFILES_DIR` | the **deploy environment** | read, never overwritten |

`test.sh` is not itself deployed, so running it at all means running it from a
checkout — the source files it asserts on exist only there. Separately, the suite
asserted POSIX-only facts unconditionally (symlink semantics, mode bits, and the
`~/.zshrc` / `~/.bashrc` / `~/.zsh/*.zsh` targets that `setup-windows.ps1`
deliberately does not create); those become explicit skips on Windows, carrying the
same reason the setup itself prints.

## Out of scope

- The other Windows hook blockers found the same day: the local hook stack's
  executable bit (#795, landed) and the scoped-Conventional-Commits validator
  (`specs/BUG-047-commit-msg-hook-windows/`). Different hooks, same family.
- Porting `test.sh` to `dotf` (ADR-020 strangler-fig). It is a test entrypoint, not a
  user-facing twin; not triggered by this change.
- The 4 checks that still fail in a bare Linux container (no deployed shell config).
  They fail identically before and after; making the container representative is a
  separate concern.

## Risks / open questions

- **Resolved — does this weaken the suite?** No. Two assertions were previously
  *vacuous*: the resolution block assigned `DOTFILES_DIR`, and section 14/15 then
  asserted the value it had just written, so they could never fail. Reading the
  variable instead makes them real.
- **Resolved — making them real turns them red in CI.** An un-provisioned
  environment genuinely has no deploy mirror. That is not a code defect, so it is an
  explicit `skip` with a stated reason — never a silent pass, never a blocking red.
- **Accepted — skips could hide a real Windows regression.** Mitigated by using the
  existing `skip()` helper, which prints and counts every skip, so the output states
  what was not checked rather than quietly shrinking the suite.

## Acceptance criteria

- [ ] **AC1 — Tests the committed tree:** with `$DOTFILES_DIR` exported and pointing at the deploy mirror, the suite still reports `Testing from: <repo checkout>`.
- [ ] **AC2 — Unblocks Windows commits:** `bash scripts/test.sh` exits 0 on Windows, and the `dotfiles-test` pre-commit hook reports `Passed`.
- [ ] **AC3 — No Linux regression:** the set of failing checks in a bare Linux container is identical before and after the change.
- [ ] **AC4 — Skips are visible:** every POSIX-only assertion skipped on Windows prints a reason and is counted, not dropped.
- [ ] **AC5 — Environment assertions are honest:** `DOTFILES_DIR` is never assigned by the script before being asserted.

## References

- Bitácora: [#794](https://github.com/mlorentedev/dotfiles/issues/794)
- Same-day siblings: #795 (hook stack executable bit, landed), `specs/BUG-047-commit-msg-hook-windows/`, #797 (closed, superseded by #795)
- Id-collision class: [#770](https://github.com/mlorentedev/dotfiles/issues/770) (SDD-039)
