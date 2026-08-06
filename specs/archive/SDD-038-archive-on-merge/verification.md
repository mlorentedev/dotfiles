---
tags: [spec, verification, templates]
created: "2026-08-06"
---

# Verification - SDD-038-archive-on-merge

## Evidence

19 tests in `tests/spec-gate-archive.bats`. 9 were red before the implementation; the other 10 were green throughout and pin the behaviour that must **not** change — the "must not fire" cases, which are the dangerous direction for an enforcement gate.

- [x] AC1 — closing without archiving fails, names the spec, prints the command -> `AC1: closing an issue whose active spec is not archived fails the gate`, `AC1: a spec created and closed in the same PR must still be archived`, `AC1: multiple closing references are all checked`
- [x] AC2 — passes once archived -> `AC2: the same PR passes once the spec folder is archived`
- [x] AC3 — non-closing references ignored -> three tests (`Refs #N`, prose, `Part of #N`)
- [x] AC4 — all frontmatter shapes + URL form + case-insensitivity -> four tests
- [x] AC5 — cross-repo reference ignored -> `AC5: a closing reference to another repo is ignored`
- [x] AC6 — fires below threshold, not skipped by `skip-sdd` -> two tests
- [x] AC7 — `skip-archive` + rationale passes, label alone fails -> two tests
- [x] AC8 — empty body / no match / no frontmatter / already archived -> four tests
- [x] AC9 — existing gate unchanged -> `tests/check-spec-gate.bats` 25/25 green

## Test status

- `bats tests/spec-gate-archive.bats tests/check-spec-gate.bats` -> **45/45 pass, 0 failures** (19 new + 25 pre-existing + 1 added mid-implementation).
- `shellcheck -x scripts/check-spec-gate.sh` -> clean.

### Dogfood against real repository data

Run on this very branch, where `specs/SDD-038-archive-on-merge/` is active and declares `issue: "mlorentedev/dotfiles#670"`:

```console
$ SDD_PR_BODY='Closes #670' ./scripts/check-spec-gate.sh --base-ref origin/main --head-ref HEAD
[FAIL] SDD archive-on-merge violation:
       This PR closes an issue whose spec is still active:
         SDD-038-archive-on-merge (#670)
       ...
         dotf spec archive SDD-038-archive-on-merge

$ SDD_PR_BODY='Refs #670' ./scripts/check-spec-gate.sh --base-ref origin/main --head-ref HEAD
[OK] Production diff 317 LOC >= threshold 50 AND spec folder touched in diff
```

This is the strongest evidence available: the gate fires on data it was not written against, and the spec it names exists only on this branch — which is exactly what the base∪head union was added for.

All four open PRs were replayed through the new gate: #762 and #763 pass below threshold, #765 passes (`Refs #748`, spec present, correctly not archived because the issue's remaining work moved to #766).

## Decisions made during implementation

- **Keyed on closing keywords, not on "a spec was touched".** PR #765 is the live proof this matters: it says `Refs #748` because the doctor work split to #766, so its spec must stay active. A gate keyed on spec presence would have forced a premature archive.
- **Presence in the head tree, not a rename in the diff.** `git diff --numstat` only reports `specs/{ => archive}/…` when rename detection fires; the same move can surface as delete+add. Asking the tree cannot be fooled by how git renders the change.
- **Union of base and head for the active-spec map.** Added after realising base alone misses a PR that *creates* a spec and closes its issue in one change — the exact "created, shipped, never archived" pattern this gate exists to stop. An archived spec is no longer under `specs/<id>/`, so the union cannot produce a false positive.
- **Runs before the LOC logic and is not skipped by `skip-sdd`.** A three-line PR can end an issue's life, and `skip-sdd` asserts "this change needs no spec", which says nothing about whether an existing spec's work is finished.
- **Two `set -e` traps found by the tests, both of which made the gate fail closed for the *wrong reason*.** `grep` exits 1 when no closing reference is present — the common case — and under `pipefail` that aborted the whole gate. Separately, `[[ -n "$num" ]] && printf` as a loop body's last command made the enclosing `while` exit 1 whenever a spec had no `issue:` field, which the caller then read as an unreadable tree. Both would have blocked ordinary PRs. Caught only because the suite asserts the negative cases.
- **Enforcement is prospective by design.** Only 16 of 44 active specs carry an `issue:` field. A spec without one cannot be linked and is not enforced; failing on it would block PRs for a data problem the author did not create.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **Yes** — an enforcement gate has two failure directions, and the expensive one is the false positive. Under `set -euo pipefail`, the idioms that abort a script are exactly those on the "nothing matched" path, which is the *normal* path for a gate.
- [ ] ADR-worthy decision? No — this implements the existing Discipline Gate's step 7 rather than changing a decision.
- [ ] New pattern candidate for `00_meta/patterns/`? Not yet; `pattern-spec-driven-development` already carries the lifecycle. Revisit if a second repo adopts the gate.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/SDD-038-archive-on-merge/` -> `specs/archive/SDD-038-archive-on-merge/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

> Archived by the sweep PR, which is what closes #670 — so it had to satisfy the
> gate introduced here, on this very spec. Verified in both directions: with the
> archive in place the gate passes, and un-archiving this folder alone makes it
> fail and name the spec.
