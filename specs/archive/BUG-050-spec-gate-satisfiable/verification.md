---
tags: [spec, verification, templates]
created: "2026-08-07"
---

# Verification - BUG-050-spec-gate-satisfiable

## Evidence

### The measurement that changed the design

The issue proposed counting `specs/archive/<id>/` as a spec touch. Against the
real mechanism that is insufficient, because the Discipline Gate does not test
path membership — it accumulates `SPEC_LOC` and compares it to `SPEC_FLOOR`:

```console
$ git show --numstat 9b24cce -- specs/     # PR #787, a real archive
0	0	specs/{ => archive}/HARNESS-051-copilot-native-skills/features.json
3	1	specs/{ => archive}/HARNESS-051-copilot-native-skills/proposal.md
0	0	specs/{ => archive}/HARNESS-051-copilot-native-skills/tasks.md
0	0	specs/{ => archive}/HARNESS-051-copilot-native-skills/verification.md
```

4 LOC against `SPEC_FLOOR=10`. Counting an archive's lines would have left the
gate exactly as unsatisfiable, while looking fixed. Hence `SPEC_TOUCHED=1`
outright.

This is pinned by a test rather than left as prose — `BUG-050: an archive move
alone is far under SPEC_FLOOR` asserts `Active-spec LOC (added+removed): 0` in
`--explain` output *and* a pass, so the pass is provably not coming from LOC.

### Red-before-green — `tests/spec-gate-archive.bats`

Against the unfixed gate (`git stash push scripts/check-spec-gate.sh`):

```
not ok 1 BUG-050: a large PR that archives its own spec satisfies both halves
not ok 2 BUG-050: an archive move alone is far under SPEC_FLOOR, so it cannot count by LOC
ok     3 BUG-050: a gratuitous archive-move earns no spec touch (#397 intact)
not ok 4 BUG-050: a spec created and archived in the same PR counts
ok     5 BUG-050: closing nothing leaves an archive move worthless as before
```

The split is the point. The three assertions of *new* behaviour fail before the
fix; the two asserting *preserved* protections pass before and after. A test that
went red-to-green on AC2 would mean the fix had traded #397's protection away.

After the fix: 25/25 in the file.

| AC | Case |
|---|---|
| AC1 | `a large PR that archives its own spec satisfies both halves` |
| AC2 | `a gratuitous archive-move earns no spec touch (#397 intact)` |
| AC3 | `a spec created and archived in the same PR counts` |
| AC4 | `closing nothing leaves an archive move worthless as before` |
| AC5 | the red/green split above |

### Regression

Full suite (`bats tests/*.bats`): no new failures. Two pre-existing failures on
clean `main`, both tracked — `board-pickup` (#755) and the busy-binary install
test (#807).

`shellcheck scripts/check-spec-gate.sh` clean; `bash -n` clean.

### Self-application

This PR is over the LOC threshold and closes its own issue, so it is the first
consumer of its own fix. CI checks out the pull request's merge ref, so the gate
that judges this PR is the copy contained *in* this PR. It passes only if the fix
is correct — and the archive of this very spec folder is what earns the touch.
