# 252 - A git pathspec is resolved against the CWD, so the same argument means two different things and one of them is silently empty

**Date:** 2026-09-01
**Area:** git, guards, verification

## What happened

GUARD-007 (#1158) added a `dotf doctor` check that asks how far the deployed
`dotf` binary is behind the checkout, scoped to `cli/` so that docs and spec
commits do not read as staleness. The count came from:

```sh
git rev-list --count <stamp>..HEAD -- cli
```

While verifying it end to end, the check and the hand-run command disagreed. The
check reported **4 cli/ commits behind**; the command I ran to confirm it
reported **0**. The check was right.

```
from repo ROOT:              git rev-list --count 11a68b1..HEAD -- cli          → 4
from cli/ subdir:            git rev-list --count 11a68b1..HEAD -- cli          → 0
from cli/ with :(top) magic: git rev-list --count 11a68b1..HEAD -- ':(top)cli'  → 4
```

A git pathspec is resolved **relative to the current working directory**, not to
the repository root. From inside `cli/`, the argument `cli` means `cli/cli`,
which does not exist. Git does not complain: a pathspec that matches nothing is
a legitimate query with an empty answer.

## Why it matters more than a wrong number

The check happened to be correct because it passes `cfg.RepoDir` as git's working
directory. That is correctness by circumstance. `RepoDir` is resolved by a
`.git` walk-up with a `DOTFILES_REPO_DIR` override (#1358 is already open about
its worktree behaviour), so the day that resolution changes — or the day someone
reuses the helper from a check that runs elsewhere — the count becomes `0`.

And `0` is the **clean** answer. The check would report "current with HEAD",
which is exactly the failure mode #1158 exists to end: a guard that reports
health it never established. The bug would have been reproduced inside its own
fix, and nothing would have said so.

This is the same class as `docs/lessons.md`'s *"a shell incompatibility that
answers wrongly beats one that fails"*, and the same class as the repo's
`.claude/CLAUDE.md` table of silent-failure shell patterns: the dangerous
constructs are not the ones that error, they are the ones that return an empty
or single-element result that reads as a finding.

## What to do instead

Use the `:(top)` pathspec magic prefix, which is always resolved from the
repository root regardless of CWD:

```sh
git rev-list --count <stamp>..HEAD -- ':(top)cli'
```

Quote it — the parentheses are shell metacharacters — and note the prefix works
anywhere a pathspec is accepted (`log`, `diff`, `grep`, `ls-files`).

The general rule: **any git invocation whose CWD is not fixed by construction
must use a root-relative pathspec.** In a tool this means every call built from
a resolved directory rather than a hard-coded one — which is every call in
`dotf`.

## Guard

`TestCheckDotfProvenanceUsesRootRelativePathspec` in
`cli/internal/doctor/checks_dotf_provenance_test.go` asserts the rev-list argv
contains `:(top)cli`. Proven fail-first: reintroducing the bare `cli` pathspec
fails it with the argv it actually built.

Assert on the **argument**, not on the count. A test that only checks the number
passes against a fixture whose CWD happens to be the root, which is the
circumstance that hid this in the first place.

## Related

- #1158 — the check this was found inside
- #1358 — `env.RepoDir` and doctor do not recognise a git worktree as a checkout
- Lesson 248 — a broken parser is a defect only where something reads it
