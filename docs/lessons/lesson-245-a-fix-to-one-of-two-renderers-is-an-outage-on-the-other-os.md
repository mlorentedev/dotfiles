# Lesson 245 — A fix applied to one of two renderers is an outage on the other OS, and nothing on the fixed side can see it

**Date:** 2026-08-29
**Context:** HARNESS-095 — #1080 taught `render_skill` in `scripts/compile-harness.sh` to drop `paths:` from deployed skill frontmatter, because Claude Code reads a top-level `paths:` as a *conditional* skill and holds it dormant until a matching file is touched. Its Windows twin, `Convert-SkillRecord` in `setup-windows.ps1`, never got the same rule. Measured on the Windows box 2026-08-29: 34 of 43 deployed skills invisible at session start, and the nine that survived were the nine whose records carry no `paths:` at all.

## What happened

`paths:` entered the skill records on 2026-08-17 (#1048). The Linux renderer was
fixed the next day. The Windows renderer kept passing every key through, so the
first `setup-windows.ps1` run after that date (2026-08-28 13:51) rewrote 43
deployed skills with a key that switches each of them off. Nothing failed. The
deploy reported success, the files were well-formed, `dotf doctor` was green,
and the only symptom was a slash command that answered `Unknown skill: docker`.

Three properties made it survive eleven days:

1. **The correlation was perfect and invisible.** Availability tracked exactly
   one frontmatter key. Every other difference between a working and a broken
   skill — size, age, `allowed-tools`, `targets`, the generated provenance
   block — was identical across both groups.
2. **The fixed side cannot observe the broken one.** The bats suite that pinned
   #1080 runs the bash renderer. No assertion in the repository read the
   PowerShell function's output, so the twin could drift arbitrarily far while
   CI stayed green on both OSes.
3. **A renderer is not a script.** The rule lives in a function extracted and
   executed by `tests/setup-windows.bats`; a source-text grep would have proved
   only that a regex exists somewhere, not that a record survives it.

## The rule

When one behaviour has two implementations, the fix is not landed until both
carry it, and the guard is not a guard until it fails on the *other* one. Two
assertions, added together, are what closes this class:

* run the twin and assert on its **output** (a fixture record with a wrapped
  `paths:` value proves the continuation lines follow their key, which is the
  half a per-line filter gets wrong); and
* compare the two implementations' shared rule **textually** across files, so
  the next edit to one of them fails loudly instead of silently deploying.

## Corollary — `! grep` in the middle of a `@test` cannot fail it

The first version of the behavioural guard used `! grep -qE ... "$out"`, and it
passed against the unfixed renderer. `bash` exempts a `!`-prefixed command from
`set -e`, so every such line before the last one in a `@test` is discarded.
`tests/lib/refute.bash` exists for exactly this and documents the trap; loading
it and *not* using it is the same vacuous pass. Mutation-verify every new guard:
remove the fix, watch the guard go red. This one was green.
