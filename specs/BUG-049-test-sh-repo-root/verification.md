---
tags: [spec, verification, templates]
created: "2026-08-07"
---

# Verification - BUG-049-test-sh-repo-root

## Evidence

Two layers: an executable regression guard for the resolution contract, plus
differential execution of the suite itself on both platforms.

### Red-before-green — `tests/test-sh-resolution.bats`

The guard asserts the contract directly rather than inferring it from suite counts.
Run in `ubuntu:26.04` with bats `1.13.0` against two trees that differ only in
`scripts/test.sh`:

```
# against the PRE-FIX script (main):        # against this branch:
1..4                                        1..4
not ok 1 resolves the repo root even ...    ok 1 resolves the repo root even ...
not ok 2 never reports the deploy mirror    ok 2 never reports the deploy mirror
ok 3 resolves the repo root when unset      ok 3 resolves the repo root when unset
not ok 4 does not announce a custom ...     ok 4 does not announce a custom ...
```

3 of 4 are red before the fix and green after. Test 3 passes in both: with
`DOTFILES_DIR` unset the pre-fix script already fell through to the script's own
parent, so that case was never broken — it is a guard against a future regression,
not a reproduction of this bug. Stating that rather than claiming 4/4 red.

### Differential — the suite itself

| | Linux (bare `ubuntu:26.04` container) | Windows (Git Bash) |
|---|---|---|
| `main` (`7c83df7`) | 4 failed / 4 skipped / 68 total | aborts at step 2/15, `Cannot continue without utils.sh` |
| this branch | **the same 4** failed / 6 skipped / 68 total | **68 passed / 0 failed / 8 skipped, exit 0** |

- [x] **AC1 — Tests the committed tree** -> header changed from `Using custom DOTFILES_DIR: <deploy-mirror>` + `Testing from: <deploy-mirror>` to `Testing from: <repo-checkout>`, with `$DOTFILES_DIR` still exported in the shell.
- [x] **AC2 — Unblocks Windows commits** -> `bash scripts/test.sh` exits 0; the commit that introduced this fix ran its own fixed script and the hook reported `Run dotfiles tests.......Passed`.
- [x] **AC3 — No Linux regression** -> failing set is byte-identical before and after: `.zshrc: missing`, `aliases.zsh: missing`, `functions.zsh: missing`, `nvm.zsh: missing`. All four are the bare container having no deployed shell config.
- [x] **AC4 — Skips are visible** -> 8 skips on Windows, each printed with a reason by the existing `skip()` helper and counted in the summary (`SKIPPED: 8`).
- [x] **AC5 — Environment assertions are honest** -> `DOTFILES_DIR` no longer appears on the left of an assignment; section 14/15 reads it. The 2 extra skips in the Linux column are precisely these two checks, which were vacuous before and are now real (and correctly skipped in an un-provisioned container).
- [x] **AC6 — Regression guard** -> `tests/test-sh-resolution.bats`, 4 cases, red-before-green table above.

## Test status

- `bash -n scripts/test.sh` -> clean
- `shellcheck -S warning scripts/test.sh` -> clean
- Windows: `PASSED: 68 / FAILED: 0 / SKIPPED: 8 / TOTAL: 76`, exit 0
- Linux container: `PASSED: 58 / FAILED: 4 / SKIPPED: 6 / TOTAL: 68` — same 4 failures as `main`
- Self-applying check: the `dotfiles-test` pre-commit hook runs the working-tree copy, so this commit was gated by the fixed script itself

## Decisions made during implementation

- **Separated the two concepts instead of overriding the variable.** Forcing
  `DOTFILES_DIR` to the repo root would have fixed the symptom and left section 14/15
  asserting a value the script had just written — the vacuity that hid this for so long.
- **Un-provisioned environment is a `skip`, not a `fail`.** A missing deploy mirror is
  a fact about the machine, not a defect in the code being committed. Blocking a commit
  for it is the exact pathology #794 is about.
- **Windows skips reuse the setup's own wording.** `setup-windows.ps1` already prints
  `[SKIP] .zshrc (POSIX-only; Windows uses $PROFILE)`; the suite now agrees with the
  installer instead of contradicting it.

## Environment findings surfaced while verifying

Recorded because each cost real time and will recur.

1. **A linked worktree cannot be bind-mounted into a Linux container for git work** —
   its `.git` is a *file* holding `gitdir: C:/Users/...`, an absolute Windows path that
   resolves nowhere in the container, so every `git` call under the mount aborts with
   `fatal: not a git repository`, including `git config --global`. Container verification
   must run against a copy with the `.git` pointer removed. Same family as #776.
2. **`robocopy /XD .git` does not exclude a worktree's `.git`** — `/XD` excludes
   *directories*, and a linked worktree's `.git` is a file. Needs an explicit removal.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "a test suite must test the tree
      it ships in, not the deployed mirror", plus the two container/worktree findings above.
- [ ] ADR-worthy decision? no.
- [ ] New pattern candidate for `00_meta/patterns/`? no — repo-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-049-test-sh-repo-root/` -> `specs/archive/BUG-049-test-sh-repo-root/`
- [ ] Bitácora ticket #794 closed with PR link (ADR-018)
- [ ] Promotions above executed
