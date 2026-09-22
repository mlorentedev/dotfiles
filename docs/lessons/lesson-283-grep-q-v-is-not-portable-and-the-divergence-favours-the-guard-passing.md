---
id: lesson-283
type: lesson
status: active
created: "2026-09-21"
owner: manu
tags: [lesson, shell, portability, grep, verification, silent-failure, guards]
---

# 283 — `grep -q -v` is not portable, and the divergence lands on the side that lets a guard pass

## What happened

While verifying HARNESS-136 (#1594), the machine-readable check for AC1 was
written the obvious way: run `dotf doctor`, slice out the section, and assert no
line reports a finding.

```sh
... | awk '/^\[Model limit drift\]/{f=1;next} /^\[/{f=0} f' | grep -qvE 'FAIL|WARN'
```

Run locally, it failed on the broken tree and passed on the fixed one — exactly
what a discriminating check should do. The reasoning behind it was wrong, and the
right answer was a coincidence.

`grep -q -v` asks "is there **any** line that does *not* match?". The section on
the broken tree looks like this:

```
  [FAIL] nan/glm5.3-flash contextWindow is 1048576, above the provider's 1000000
    The provider rejects a request this config invites.
  [WARN] nan/glm5.3-flash maxTokens is 65536, below the provider's 131072
    Capability forfeited silently; nothing breaks.
```

Half those lines contain neither `FAIL` nor `WARN`, so the honest answer is
"yes, there are non-matching lines" — exit 0, **the guard passes on a broken
tree**. That is what GNU grep answers. The local run disagreed because `grep` in
this shell is not GNU grep:

```
$ type grep
grep is a shell function ...
$ grep --version
ugrep 7.8.4

$ grep    -qvE 'FAIL|WARN' broken-section.txt ; echo $?   # ugrep
1
$ /usr/bin/grep -qvE 'FAIL|WARN' broken-section.txt ; echo $?   # GNU grep 3.12
0
```

So the check was **correct on the developer's machine and vacuous in CI**, where
`grep` is GNU. The divergence points the wrong way: the implementation that
disagrees is the one the guard is eventually graded by, and it disagrees by
passing.

## Why it matters

This is the shape the repo's prohibited-pattern table is built around — *"a shell
incompatibility that answers wrongly beats one that fails"* — with a new twist.
The existing rows are about `bash` vs `zsh`, where the wrong answer is usually an
empty result that reads as "nothing found". This one is about **two
implementations of the same command name**, and neither shell is at fault.

Two things make it hard to catch:

1. **Testing it locally proves nothing**, because local is the implementation
   that answers the way you expected. The check "worked"; it was graded by the
   wrong grader.
2. **`type grep` is not something anyone runs.** A shell function or alias
   shadowing a core utility is invisible to every script that just calls `grep`,
   and modern replacements — ugrep, ripgrep — are normally installed precisely
   *because* they behave a bit differently.

## What to do instead

**Never write `grep -q -v`.** Negate the pipeline, not the match:

```sh
# wrong: asks "is some line clean?", and implementations disagree
... | grep -qvE 'FAIL|WARN'

# right: asks "is any line dirty?", and `!` inverts the answer
! ( ... | grep -qE 'FAIL|WARN' )
```

`-q` without `-v` means the same thing everywhere, so the negation belongs in the
shell where it is unambiguous. Note `!` legally negates a **whole pipeline**;
`a | ! b` is a syntax error, which is its own small trap.

**Better still, assert the positive.** The rewrite above was still not good
enough. Asserting *absence of findings* passes when the check never ran at all —
against `main`, the doctor binary had no such section, so there were no findings
and the guard went green. The final form asserts the section **ran and reported
agreement**:

```sh
... | grep -q 'models match the provider catalog'
```

Which is the same rule the check itself enforces one level up: *"nothing to
compare" and "everything agrees" print the same clean report and mean opposite
things.* A verification that can be satisfied by absence will eventually be
satisfied by absence.

## Where this applies

- `specs/*/features.json` verification commands — they are run by a harness on
  some other machine, so they must be portable and positively-asserting.
- Any CI step, pre-commit hook or guard script asserting that something is
  **clean**. Prefer a positive marker the passing path emits over the absence of
  a failing one.
- When a sweep returns an unexpected result, run `type grep` (and `type ls`,
  `type find`) before believing it.

## See also

- `.claude/CLAUDE.md`, the prohibited-pattern table — the `bash`/`zsh` siblings
  of this failure, and the "a script whose failure path reports 'found nothing'
  needs a preflight" note directly below it.
- `docs/lessons/lesson-088-gh-project-item-list-truncates-to-limit-silently-c.md`
  and `lesson-282-...` — the same family: a tool answering a narrower question
  than the one asked, without saying so.
