---
id: lesson-145-a-three-dot-origin-base-head-diff-needs-the-merge-
type: lesson
status: active
created: "2026-07-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 145: A three-dot `origin/BASE...HEAD` diff needs the merge-base — `--depth=1` starves it, and a fail-closed gate makes that loud

**Context**: The C3 fail-closed hardening (#686/#716) made `check-spec-gate.sh` exit 2 whenever `git diff "origin/BASE...HEAD"` cannot resolve its refs, instead of silently passing with `TOTAL_LOC=0`. The next PR to run the gate — #728, whose branch lagged `main` by 6 commits — failed `spec-gate` with `base/head ref could not be resolved. The Discipline Gate fails closed (exit 2).`

**Problem**: `spec-gate.yml` fetched the base ref with `git fetch --no-tags --prune --depth=1 origin "$BASE_REF"`. The gate diffs `origin/BASE...HEAD` — the **three-dot** form, which git resolves as `git diff $(git merge-base origin/BASE HEAD) HEAD`. A depth-1 base tip carries no history, so for any PR branch that lags `main` by more than one commit the merge-base is not reachable from the shallow base ref → `git diff` errors → the gate (correctly) fails closed. The C3 change did **not** introduce the bug: the shallow fetch was always inadequate for a three-dot diff; C3 only stopped it from passing silently as a zero-LOC no-op. This is exactly the "shallow/fresh clone" case #686/C3 warned about, now made loud instead of latent.

**Solution**: Drop `--depth=1` from the base-ref fetch — one line: `git fetch --no-tags --prune origin "$BASE_REF"`. `actions/checkout` already fetches full head history (`fetch-depth: 0`); the full base fetch makes the merge-base always reachable. Verified end-to-end: #728 (rebased onto current `main`) and #729's own `spec-gate` run both pass under the fixed workflow. Validated on #730, a branch deliberately behind `main`, which now passes.

**Rule**: A three-dot diff (`A...B`) is defined relative to the **merge-base** of A and B, so both sides need enough history to *find* that merge-base — a `--depth=1` fetch of either ref starves it the moment the branch is more than one commit behind its base. In CI, whenever a gate uses a three-dot diff or `git merge-base`, pair `fetch-depth: 0` on `actions/checkout` with a **full (non-shallow) fetch of the base ref** — never `--depth=1`. And remember the interaction that surfaced this: a fail-closed gate does not create shallow-fetch bugs, it *exposes* pre-existing ones as hard failures — so when hardening a check to fail closed, audit every ref-resolution the check depends on in the same change, because faults that used to pass silently will now block every PR.
