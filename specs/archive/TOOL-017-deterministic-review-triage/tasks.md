---
tags: [spec, tasks, templates]
created: "2026-08-18"
---

# Tasks - TOOL-017-deterministic-review-triage

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch: one per slice instead of one long-lived branch; the first is `feat/review-finding-class-marker`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] **Open question resolved before implementation starts** (owner, #1062, 2026-08-19: the marker is emitted): the classification
      marker does not exist yet. Either `.pr_agent.toml` is asked to emit a stable
      one (TOOL-013's surface, coordinate with whoever holds it) or every finding
      is `unclassified` and every PR escalates — fail-closed but useless. This
      gates the value of everything below.

## Implementation

> TDD order. The reproduction harness comes first, because every later task
> depends on it and it is the piece most able to be silently vacuous.
>
> **Decided by the owner (2026-10-21, #1062).**
> - AC4-AC6, what counts as a reproduction: **(A)**. Only a committed failing
>   test, or a command whose non-zero exit is recorded, reproduces a finding.
>   Anything else is "not reproduced" and goes to a human; a model's judgement
>   never counts.
> - AC7, per-comment dispositions: **(B)**. The PR-level gate stays (`dotf pr
>   land` refuses a PR the triage queue lists), and the unclassified-findings
>   report is the per-finding signal until the marker's adoption is measured.
>   A per-row check and a close-without-merge guard are not built.

- [x] `[AC5]` committed-test path: `dotf pr reproduce -- <cmd>` refuses a dirty tree (untracked included), records the HEAD it ran on, and classifies the exit as `reproduced`, `did-not-reproduce` or `could-not-run` (start failure, 126/127, signal, timeout: never a reproduction, owner #1062). It prints one row a human pastes into `## Review triage`; it applies nothing (`cli/internal/reproduce`)
- [x] `[AC5]` red-direction for that path: six mutations of `reproduce.go` (timeout, 127, signal, dirty, untracked, recorded head), each confirmed landed by checksum, each killed
- [x] `[AC5]` reproduction harness: apply a mutation, **verify by checksum that it landed**, report `did-not-land` as a state distinct from `did-not-reproduce`. `dotf pr reproduce --probe <patch>`: landing is decided by the target files' bytes, not by `git apply`'s exit; the probe is reverted whatever the check said, and a revert that does not restore the bytes is an error
- [x] `[AC5]` red-direction: an invalid mutation must report `did-not-land`, never a clean result — the exact failure two sessions hit on 2026-08-17. `TestRunWithAProbeThatDoesNotLand` (stale context, empty patch, mode-only patch); five mutations of the probe path, each landed by checksum, each killed; a sixth (empty target list) was equivalent and its guard was deleted
- [x] `[AC2]` `[P]` read reviewer identity + notice markers from `harness/review-attestation.json`; no second copy of the distinction
- [x] `[AC2]` red-direction: adding a reviewer to the registry changes triage behaviour with no code edit
- [x] `[AC3]` extract a finding's class from a declared marker; absent marker → `unclassified`
- [x] `[AC3]` red-direction: a finding whose class appears only in prose is `unclassified`, not parsed
- [x] `[AC3]` consume `Classify` where findings are read: `dotf pr triage-queue` names a pending PR's inline findings without a declared class (report only, the verdict is unchanged)
- [x] `[AC3]` declare the classifying reviewers in the registry: `classification.reviewers: ["github-actions"]`, pinned by `TestTheRealRegistryDeclaresTheClassificationMarker`
- [x] `[AC4]` gate `apply` on a successful reproduction for REAL findings. The gate refuses (owner, 2026-10-22, #1062): `dotf pr triage-queue` holds a triaged PR whose record applies a REAL finding without a `Reproduction: reproduced` row, so `dotf pr land` refuses it; findings that cannot be read hold it too (fails closed). `prtriage.UnreproducedFindings` today
- [x] `[AC6]` record a refuted REAL finding with its evidence, distinct from a skip: the disposition `refuted` needs a `Reproduction: did-not-reproduce` row, and a REAL finding that is skipped or declined holds the PR. `pr-review-triage` documents the rule
- [x] `[AC1]` ~~trigger from the reviewer-finished signal (GUARD-003's `workflow_run`)~~ superseded: the pull-based `dotf pr triage-queue` (#1057, #1101), accepted by the owner on #1062 (2026-10-10)
- [x] `[AC7]` closing guard, re-scoped to PR level by the owner (option B): `dotf pr land` refuses a PR the triage queue lists (`cli/internal/prland/prland.go:118`)
- [x] `[AC7]` red-direction: a PR whose reviewer spoke after the last triage is pending (`TestEvaluate`), and `land` refuses it
- [x] `[AC8]` every check above has a red-direction test — asserted, not assumed: each slice ran a mutation pass, each mutation confirmed landed by checksum (AC3 #2845/#2863, AC5 15 killed #2903, AC4/AC6 16 killed here); every equivalent survivor had its guard deleted

## Verification

- [x] `go build ./... && go vet ./... && go test ./...`
  - Evidence (2026-10-30, at `37ecb8db`): exit 0, and `GOOS=windows go vet ./...` exit 0. Each criterion's own command in `features.json` exits 0 and runs every test it names.
- [x] `bats tests/*.bats`
  - Evidence (2026-10-30, at `37ecb8db`): 2001 ok, 1 not ok. The failure is `tests/harness-doctrine-records.bats` "--refresh against the real vault…", which reads the machine's vault checkout: that checkout was 6 commits behind origin, so its `crystallize` source predates the record on main. It is not TOOL-017's, and the record matches origin's source (#2907, #2913, both closed).
- [x] `shellcheck` clean
  - Evidence (2026-10-30): `shellcheck -S warning scripts/*.sh setup-linux.sh` exit 0, the severity CI runs at.
- [x] Red-teamed: removing each fix turns its guard red, **and each mutation is confirmed to have landed**
  - Evidence: the mutation passes recorded on the AC8 line above, each mutation confirmed landed by checksum: AC3 #2845/#2863, AC5 15/15 #2903 and #2935, AC4/AC6 16/16 #2924.
- [x] Replayed against #1059's real payload: the REAL finding must come back **refuted**, and the multi-line hole must not be claimed as the reviewer's finding
  - Evidence (2026-10-22, at `1719024f`, the commit #1059 was reviewed on): the reviewer's flagless evasion as a probe gives `Reproduction: did-not-reproduce (exit 1, defect when the check passes) … with probe flagless.diff: \`bats tests/ci-age-pin.bats\``, since bats catches it at line 98: refuted. The multi-line `apt-get` form, which the reviewer did not name, gives `reproduced (exit 0, …) with probe multiline.diff`: a separate finding, not the reviewer's.
  - The first replay wrapped the check as `sh -c '! bats …'`, which would read a missing `bats` (127) as a refutation; `--defect-when-passes` replaces that wrapper (#2926).

## Machine-readable features

`features.json` sits beside this file, one feature per acceptance criterion, each
with a shell command whose exit 0 is the pass condition.

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the
harness, after running `verification` and capturing exit code 0, may set that
terminal state. Every entry starts `pending` with empty `evidence`.
