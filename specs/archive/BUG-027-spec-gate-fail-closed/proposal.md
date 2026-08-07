---
id: "BUG-027-spec-gate-fail-closed"
type: spec
status: archived
created: "2026-07-09"
tags: [spec, proposal, ci, spec-gate, security, sdd-enforcement]
template_version: "1.0"
---

# BUG-027-spec-gate-fail-closed

Make the SDD Tier-4 Discipline Gate fail closed and close its three bypass
routes, so it cannot silently pass an arbitrary PR.

## Why

`scripts/check-spec-gate.sh` is the enforcement gate behind
`.github/workflows/spec-gate.yml`. An audit (issue #686, C3/C25) found it could
pass silently on any PR:

- **C3 — fail-open on a diff error.** The file list came from
  `done < <(git diff --numstat "${BASE_REF}...${HEAD_REF}" 2>/dev/null || true)`.
  If `BASE_REF` did not resolve (shallow/fresh clone, a typo'd ref, a detached
  worktree), git errored, stderr was discarded, `|| true` swallowed the exit,
  the loop read nothing, `TOTAL_LOC` stayed 0, and the gate printed "below
  threshold" and exited 0. CI usually fetches the ref first, but the opt-in local
  pre-push hook and any manual run inherited the fail-open. An enforcement gate
  must fail **closed**.
- **C25 — three bypass routes.**
  1. `*generated*` in `_excluded()` dropped any path merely containing
     "generated" (e.g. a hand-written `internal/generated_names.go`) from the LOC
     count.
  2. The `dependencies` label skipped the gate with no actor check;
     `spec-gate.yml` re-runs on `labeled`, so any collaborator could add the
     label and neutralise the gate.
  3. Any touch to any active `specs/<id>/` satisfied the gate. With ~35
     shipped-but-unarchived specs in-tree, a one-line edit to an unrelated stale
     spec legitimised a large PR.

Source: audit issue #686 (C3, C25); pairs with #670 (archive-on-merge) and
#589 (conventional-commit gate).

## What

After this PR, in `scripts/check-spec-gate.sh` (+ `spec-gate.yml`):

- **C3 fail-closed.** The diff is computed before the loop; a non-zero git exit
  prints an actionable error and exits 2 (setup error), never 0. `if !` keeps
  `set -e` from aborting before the message. Empty diff (legitimately no change)
  still passes — only a diff *error* fails closed.
- **C25.1.** The `*generated*` glob is removed. Zero generated files exist
  in-tree; a real one gets an explicit allowlist entry (e.g. `*.pb.go`), never a
  substring match.
- **C25.2.** The `dependencies` skip is gated on the PR author being a known
  dependency bot (`dependabot[bot]`, `dependabot-preview[bot]`, `renovate[bot]`)
  via a new `SDD_PR_AUTHOR` env (wired in `spec-gate.yml` from
  `github.event.pull_request.user.login`). A non-bot `dependencies` label no
  longer skips — it falls through to the normal gate.
- **C25.3.** A spec touch counts only if the added+removed LOC within active
  `specs/<id>/` folders reaches `SPEC_FLOOR` (10). A real new/updated spec
  clears it easily; a one-line alibi does not. `--explain` prints the active-spec
  LOC and floor.

## Out of scope

- Full spec<->PR linkage (deriving "this PR's spec" from a branch/PR convention).
  The `SPEC_FLOOR` heuristic defeats the trivial alibi; the complete linkage
  pairs with #670, which archives shipped specs and shrinks the alibi surface.
- The `dependencies` bot allowlist is a fixed set; if the repo adopts another
  update bot, add it to `_is_dependency_bot`.
- Enabling dependabot itself (separate change; the C25.2 gate is what makes the
  `dependencies` skip safe once it is enabled).

## Risks / open questions

- **`SPEC_FLOOR` false-negative.** A multi-PR spec sequence whose follow-up PR
  only checks off one or two tasks (< 10 LOC in the spec folder, no other spec
  change) would no longer satisfy the gate on spec grounds; it would need a
  slightly more substantive spec update or the `skip-sdd` label + rationale.
  Accepted: such PRs are rare and the escape hatch is auditable.
- **`SDD_PR_AUTHOR` unset locally.** Local/manual runs have no PR author, so the
  `dependencies` skip never applies there — correct (fail closed).

## Acceptance criteria

- [x] A diff error (unresolvable base ref) exits 2, not 0.
- [x] A hand-written `*generated*`-named path is counted, not excluded.
- [x] `dependencies` label skips only for a bot author; a non-bot author falls
      through to the normal gate.
- [x] A sub-`SPEC_FLOOR` active-spec touch does not satisfy the gate; a
      substantive one does.
- [x] `spec-gate.yml` passes `SDD_PR_AUTHOR`.
- [x] bats coverage: 25 tests green in `tests/check-spec-gate.bats`, incl. the 5
      new #686 cases; `bash -n` clean; existing #397 rename tests still pass.

## References

- GH issue: [#686](https://github.com/mlorentedev/dotfiles/issues/686)
- Gate: `scripts/check-spec-gate.sh`, `.github/workflows/spec-gate.yml`
- Tests: `tests/check-spec-gate.bats`
- Related: #670 (archive-on-merge), #589 (conventional-commit gate), #126
  (PSScriptAnalyzer coverage — separate)
