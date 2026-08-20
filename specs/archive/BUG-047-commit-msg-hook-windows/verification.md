---
tags: [spec, verification, templates]
created: "2026-08-07"
---

# Verification - BUG-047-commit-msg-hook-windows

## Evidence

- [x] AC1 (scoped accepted) -> `accepts a scoped Conventional Commit`, `accepts every scoped subject currently on main`, `accepts a breaking-change marker, scoped and unscoped`, `accepts a scope containing dots, slashes and dashes`
- [x] AC2 (malformed rejected) -> `rejects invalid commit message`, `rejects a missing space after the colon`, `rejects an empty subject after the colon`
- [x] AC3 (subject only) -> `validates the subject only: a conforming body cannot rescue it`
- [x] AC4 (git-generated exempt) -> `exempts git-generated merge and revert subjects`, `exempts rebase fixup and squash subjects`
- [x] AC5 (runs on Windows) -> `the shebang resolves via env so the hook can run on Windows`, plus the live commit below
- [x] AC6 (spec-gate runs on Windows) -> `bats tests/check-spec-gate.bats` 25/25 after the shebang change; the gate then executed and returned a real verdict on this branch
- [x] AC7 (cross-shell contract preserved, green without zsh) -> `validate-commit-msg.sh valid sh syntax`, `... bash syntax`, `... zsh syntax` (skipped where zsh is absent)

## Test status

```
$ bats tests/validate-commit-msg.bats
18 tests, 0 failures, 3 skipped     # 3 skipped = zsh not installed on this Windows host

$ bats tests/check-bats-names.bats
7 tests, 0 failures                 # includes "the repo's own tests/ pass"

$ bats tests/check-spec-gate.bats
25 tests, 0 failures                # unchanged by the shebang fix
```

**Live proof on Windows (AC5).** The fix validates itself: this branch's own commit was made on native Windows with the new hook active, and its subject is scoped (`fix(hooks): ...`) — precisely the form the old pattern rejected and the old shebang could not even evaluate.

```
Detect hardcoded secrets.................................................Passed
Run dotfiles tests......................................................Skipped
Validate message format..................................................Passed
```

Before the fix, on the same host:

```
Validate message format..................................................Failed
- hook id: validate-commit-msg
- exit code: 1
Executable `/bin/sh` not found
```

**Live proof for AC6.** Before the `check-spec-gate.sh` shebang fix, `git push` died with `Executable /bin/bash not found`. After it, the gate ran and returned a substantive verdict — it rejected this very branch for 57 LOC of production diff with no spec folder touched, which is why this spec exists. A gate that can fail you is a gate that was absent before.

No regressions: `dotfiles-test` was skipped throughout (finding 1 of #794, unrelated and pre-existing).

## Decisions made during implementation

- **Kept the hook POSIX instead of rewriting it in bash.** The first draft used `[[ ]]`, arrays and `=~`. Reading `tests/validate-commit-msg.bats` first showed a deliberate, tested cross-shell contract (sh + bash + zsh). The real defect was never the language — it was the absolute path in the shebang, so `#!/usr/bin/env sh` is both the smaller and the correct fix.
- **Permissive type, not an allow-list.** An allow-list would reject `wip:`, which the repo uses. The gate's job here is shape; release-please already ignores unknown types.
- **Skipped the zsh cases rather than deleting them.** Deleting would have made the suite green by removing coverage. Skipping keeps the contract enforced wherever zsh exists.
- **Took the spec path over the `skip-sdd` label.** The gate offers both. At 57 LOC with multiple findings and an existing ticket, a written spec is the honest answer.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — a quality gate that cannot execute on a supported platform is indistinguishable from no gate, and it hides a second defect behind the first: the commit-msg validator had been rejecting the repo's own commit convention for as long as it was installed, invisible because squash-merges never run local hooks.
- [ ] ADR-worthy decision? no — no architectural choice here; the shebang convention is already the repo's majority (22 `env`-based vs 12 absolute).
- [ ] New pattern candidate for `00_meta/patterns/`? no — single-repo hook mechanics.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-047-commit-msg-hook-windows/` -> `specs/archive/BUG-047-commit-msg-hook-windows/`
- [ ] Bitácora board ticket moved to Done / closed with PR link (ADR-018) — **only findings 3/4**; #794 stays open for findings 1/2
- [ ] Promotions above executed (if any)
