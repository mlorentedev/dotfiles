---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - BUG-042-closing-keyword-markdown

## Evidence

### Red-before-green

Against the unfixed scanner:

```
not ok 1 BUG-042: a closing keyword inside a fenced block does not fire
not ok 2 BUG-042: a closing keyword inside an inline code span does not fire
ok     3 BUG-042: a genuine closing keyword in ordinary prose still fires
ok     4 BUG-042: a fence closes only on a matching delimiter, so later prose is still scanned
not ok 5 BUG-042: the colon form is a closing declaration too
```

The split is the deliverable. Cases 1, 2 and 5 assert new behaviour and fail
before the fix. Cases 3 and 4 assert *preserved* behaviour and pass before and
after — case 3 in particular is the regression guard against the anchoring
approach the issue proposed and this spec rejects: had the keyword been anchored
to line starts, case 3 would have flipped to red and the gate would have gone
silent on declarations GitHub honours.

After the fix: 30/30 in `tests/spec-gate-archive.bats`.

| AC | Case |
|---|---|
| AC1 | a closing keyword inside a fenced block does not fire |
| AC2 | a closing keyword inside an inline code span does not fire |
| AC3 | a genuine closing keyword in ordinary prose still fires |
| AC4 | a fence closes only on a matching delimiter, so later prose is still scanned |
| AC5 | the colon form is a closing declaration too |

### Lint

`shellcheck scripts/check-spec-gate.sh` clean — with one deliberate
`SC2016` suppression on the inline-span pattern, which is single-quoted because
its backticks must reach the regex engine rather than the shell. `bash -n` clean.
`check-bats-names.sh` clean.

### Self-application

This PR's own body contains a genuine `Closes #773` **and** discusses
closing-keyword matching at length, with worked examples. CI evaluates it with the
PR's own (fixed) scanner, so the fenced examples are stripped and only the real
declaration fires. The PR is therefore a live test of its own change — the same
shape of body that broke #767.

### Rebase note

`#808` landed in the same file and appended to the same test file. The conflict
was resolved by taking main's version of `spec-gate-archive.bats` whole and
re-appending this branch's block, rather than splicing the conflict hunks — the
first attempt at splicing cut a test in half, which `bats` caught as a syntax
error rather than a silently missing test.
