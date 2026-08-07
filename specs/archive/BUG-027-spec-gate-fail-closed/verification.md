---
id: "BUG-027-spec-gate-fail-closed"
type: spec
status: implementing
created: "2026-07-09"
tags: [spec, verification]
template_version: "1.0"
---

# Verification — BUG-027-spec-gate-fail-closed

## Automated evidence (this branch)

`bats tests/check-spec-gate.bats` — 25/25 pass, including the 5 new #686 cases:

| # | Test | Covers |
|---|---|---|
| 6 | exits 0 when diff >= threshold AND a substantive spec folder is touched | C25.3 (positive) |
| 7 | a trivial (sub-floor) active-spec touch does NOT satisfy the gate | C25.3 (alibi) |
| 11 | exits 0 when dependencies label present AND author is a bot | C25.2 (positive) |
| 12 | dependencies label from a NON-bot author does not bypass the gate | C25.2 (bypass) |
| 13 | fails closed (exit 2) when the base ref does not resolve | C3 |
| 14 | a hand-written *generated* path is counted, not excluded | C25.1 |

Regression: the #397 spec-archive rename tests (24, 25) still pass — the
fail-closed rewrite and the SPEC_LOC accounting leave rename normalization
intact. `bash -n scripts/check-spec-gate.sh` clean.

## Manual spot checks

- C3: `check-spec-gate.sh --base-ref origin/does-not-exist --head-ref HEAD`
  in a repo with no such ref -> exit 2 with "could not be resolved" (was exit 0).
- C25.2: `SDD_LABELS=dependencies SDD_PR_AUTHOR=some-human` on a 60-LOC diff ->
  exit 1 (was exit 0). With `SDD_PR_AUTHOR=dependabot[bot]` -> exit 0.

## CI evidence (post-push, T7)

- [ ] `lint` (shellcheck over scripts/) green.
- [ ] `test` (bats incl. check-spec-gate.bats) green.
- [ ] `spec-gate` self-run green — this PR touches `specs/BUG-027-.../` with a
      substantive proposal, so it satisfies its own hardened gate (dogfood).

## Note

The gate now enforcing itself on this PR is the proof of the C25.3 fix: the
change is >= threshold and passes because a *substantive* spec folder is present,
not because any stale spec was grazed.
