---
tags: [spec, verification, templates]
created: "2026-08-09"
---

# Verification - BUG-062-crystallize-refuse-yaml

## Evidence

- [x] **Criterion 1** (refused, non-zero, at both indents) → cases *"refuses a
      block-scalar MEMORY.md indented four spaces (hive)"* and *"…six spaces
      (pollex)"*.
- [x] **Criterion 2** (byte-identical, still parses) → cases *"leaves the refused
      file byte-identical"* and *"the refused file still parses as YAML"*.
- [x] **Criterion 3** (plain markdown unaffected) → case *"a plain-markdown
      MEMORY.md is untouched by the guard"*, which re-asserts the HARNESS-029
      invariant #851 established.
- [x] **Criterion 4** (refusal counts as skipped) → case *"a refusal is counted as
      skipped, not processed, in both twins"*.
- [x] **Criterion 5** (both twins guarded) → case *"the PowerShell twin carries the
      same guard"*.

## Test status

```console
$ bats tests/knowledge-crystallize-yaml-guard.bats \
       tests/knowledge-crystallize.bats tests/knowledge-crystallize-ps1.bats
42 passed, 0 failed (2 skipped: pwsh unavailable)

$ shellcheck scripts/knowledge-crystallize.sh
CLEAN
$ bash -n scripts/knowledge-crystallize.sh && zsh -n scripts/knowledge-crystallize.sh
OK
```

Red-first: the guard suite was written before the implementation and run against
the unmodified script — 5 failures, 1 pass. Case 4's failure output reproduced
#857's `yaml.scanner.ScannerError` verbatim, so the suite independently
re-derived the reported defect rather than trusting the report.

**Verified against the reporter's own artifact.** #857 ships a non-destructive
repro against `hive`'s real `MEMORY.md`; it was re-run against the guarded script
rather than only against fixtures of our own choosing:

```console
$ bash repro-857.sh <fake-home> ./scripts/knowledge-crystallize.sh
[ERROR] Refusing to stamp …/hive/memory/MEMORY.md
[ERROR] Its body sits inside a YAML block scalar, which this script cannot edit
[ERROR] without corrupting the file (#857). Stamp it by hand until the
[ERROR] YAML-aware 'dotf vault crystallize' lands (#490).
exit=1

file byte-identical:          YES
still parses as YAML:         YES
duplicate stamps at column 0: 0
```

Against `dbe91db` the same input produced two `[SUCCESS]` lines, a duplicate stamp
at column 0, and a `ScannerError`.

**A vacuous test was written and deleted.** A grep-assertion *"neither twin keys
detection on a literal indent width"* passed. Mutation showed it also passed after
a literal `^    ## Session Handoff` anchor was added to the script — it could not
fail, so it was decoration reading as coverage. Deleted, with a comment in its
place: the claim is already carried behaviourally by the four- and six-space
cases, either of which turns red if the guard keys on a width. Worth recording
because it is the same defect class as the issue this branch serves.

- No regressions: full `bats tests/*.bats` — see the run recorded at the bottom of
  this file.

## Decisions made during implementation

- **Refuse, do not fix.** The robust fix is parse → mutate → re-dump, which shell
  does badly and which #490 owns. A shell YAML mutator would be built to be
  deleted, and until #490 flips the callers the twins are still the running code.
  Between now and then the only choice is corrupt or refuse.
- **Structural detection, never an indent probe.** `pollex` indents six spaces and
  `hive` four; a literal width is correct on one machine and wrong on the other.
  The guard keys on `---` plus a `<key>: |` opener.
- **Fixed the `--all` counter in the same change.** `processed = found - skipped`
  reported "5 / 5 (0 skipped)" while declining one. The guard is the first real
  failure path, so it would have made that lie visible on every wrapped project —
  and "prints success while doing nothing" is precisely what #857 is about. The
  PowerShell twin carried the same defect plus a dead `$processed++`.
- **Single-project mode exits 1 explicitly** rather than relying on `set -e` to
  abort before `exit 0`.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? **no** — the generalisable part (a check that
      answers a weaker question than the one you need) is already promoted to
      `pattern-verify-state-before-acting`, per #857's own note.
- [ ] ADR-worthy? **no**.
- [ ] New pattern? **no**.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/BUG-062-crystallize-refuse-yaml/`
- [ ] Bitácora ticket closed with PR link (ADR-018)

> **Not archived by this PR.** #857 asks for crystallize to stop corrupting *and*
> to handle the shape. This delivers the first half. The spec stays active until
> #490's port lands the second, so the PR carries `Refs #857`, not a closing
> keyword.
