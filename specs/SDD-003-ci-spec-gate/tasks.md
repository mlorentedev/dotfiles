---
tags: [spec, tasks, sdd, ci]
created: "2026-05-19"
---

# Tasks - SDD-003-ci-spec-gate

> TDD order. One task = one focused commit. Tick as you go. Freeze once status moves to `implementing`.

## Setup

- [ ] Branch created from main: `feat/SDD-003-ci-spec-gate`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation (TDD)

### Phase 1 — Helper script with full local-only coverage

- [ ] Create `tests/check-spec-gate.bats` with 4 red tests:
  - Case A: diff <50 LOC (excluding tests/) and no specs folder → exit 0
  - Case B: diff ≥50 LOC and specs/<feature-id>/ folder touched → exit 0
  - Case C: diff ≥50 LOC and no specs folder → exit 1, message mentions Discipline Gate
  - Case D: env vars `SDD_LABELS=skip-sdd` and `SDD_PR_BODY` contains non-empty `## SDD skip rationale` → exit 0
- [ ] Implement `scripts/check-spec-gate.sh`:
  - Flags: `--base-ref REF`, `--head-ref REF`, `--threshold N` (default 50), `--explain`
  - Reads `SDD_LABELS` (csv) and `SDD_PR_BODY` env vars (CI sets them; locally empty = no skip)
  - LOC computation: `git diff --numstat $base...$head` summed, excluding paths matching the exclusion list (tests/, specs/archive/, *.lock, *.lockb, .gitignore, CHANGELOG.md, generated patterns)
  - Spec-folder presence: `git diff --name-only $base...$head | grep -E '^specs/[A-Z]+-[0-9]+...|^specs/[0-9]{4}-...'`
  - `set -euo pipefail`; `shellcheck` clean
- [ ] All 4 bats cases green; run full `tests/*.bats` to confirm no regression

### Phase 2 — CI workflow

- [ ] Add `.github/workflows/spec-gate.yml`:
  - Triggers: `pull_request` (opened, synchronize, reopened, labeled, unlabeled, edited)
  - Steps: checkout (fetch-depth 0), set env from PR labels + body, run `scripts/check-spec-gate.sh --base-ref origin/${{ base }} --head-ref HEAD --explain`
  - Skip via dependabot label exemption inline (avoids gate firing on dependency bumps)
- [ ] Locally simulate the workflow logic by exporting env vars and running the script; confirm all 4 acceptance cases

### Phase 3 — PR template

- [ ] Add `.github/pull_request_template.md`:
  - Sections: `## Summary`, `## SDD checklist` (vault entry / spec folder / proposal.md filled), `## SDD skip rationale` (header empty by default — autor lo rellena solo si va a labelar `skip-sdd`), `## Test plan`
  - Skill: the template's `## SDD skip rationale` header existence is what the gate's body-check matches against; the body must be non-empty under it for skip to be valid

### Phase 4 — Pre-push hook (opt-in)

- [ ] Extend `scripts/install-precommit.sh` with a `--with-sdd-gate` flag (default off so existing users unaffected)
- [ ] When enabled, installs `.git/hooks/pre-push` that runs `scripts/check-spec-gate.sh --base-ref origin/main --head-ref HEAD`
- [ ] Add bats coverage for the new flag path (1 case: flag installs hook; script absence does not break the existing flow)

### Phase 5 — Wire-up + docs

- [ ] Update `README.md` Requirements/Workflow section with one line on the gate (where to look when it fails)
- [ ] No CLAUDE.md / AGENTS.md changes — the prose rule is already there; gate is its enforcement, not a new rule
- [ ] `features.json` filled (one feature per acceptance criterion, all `state: pending`)

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test in `tests/check-spec-gate.bats` or `tests/install-precommit.bats`
- [ ] Every acceptance criterion has a matching entry in `features.json` with non-vacuous `verification`
- [ ] `shellcheck scripts/check-spec-gate.sh` clean
- [ ] Full bats suite green (target: 396 + 5 new = 401 passing)
- [ ] No unrelated changes in the diff (no scope creep, no opportunistic edits)
- [ ] `verification.md` filled with commit hashes + test output excerpts + simulated PR scenarios
- [ ] PR opened referencing `specs/SDD-003-ci-spec-gate/`
- [ ] **Self-test**: the PR opening this gate must itself pass the gate (the spec folder is in the diff → green case B)

## Machine-readable features

See sibling `features.json`. Pass-state gating: agent only writes `"state": "pending"`; harness flips to `passing` after capturing exit 0.
