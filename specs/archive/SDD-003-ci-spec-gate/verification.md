---
tags: [spec, verification, sdd, ci]
created: "2026-05-19"
---

# Verification - SDD-003-ci-spec-gate

## Evidence

Mapping of acceptance criteria from `proposal.md` to test cases and observed behaviour. Commit hashes filled at PR-open time (user controls commits).

- [x] AC1 exit 0 when diff <50 LOC -> `tests/check-spec-gate.bats:exits 0 when diff is below threshold (no spec needed)` (test 5)
- [x] AC2 exit 0 when diff >=50 LOC AND active specs/<feature-id>/ touched -> `tests/check-spec-gate.bats:exits 0 when diff >= threshold AND specs folder is touched` (test 6)
- [x] AC3 exit 1 when diff >=50 LOC AND no specs folder, with AGENTS.md reference -> `tests/check-spec-gate.bats:exits 1 when diff >= threshold AND no specs folder` (test 7)
- [x] AC4 skip-sdd label + non-empty rationale -> exit 0 -> `tests/check-spec-gate.bats:exits 0 with skip-sdd label AND non-empty rationale` (test 8); empty rationale variant -> `tests/check-spec-gate.bats:exits 1 when skip-sdd label present but rationale empty` (test 9)
- [x] AC5 `.github/workflows/spec-gate.yml` invokes the script on `pull_request` -> file present + ran `python3 -c 'import yaml; yaml.safe_load(...)'` clean; `grep` confirms `check-spec-gate.sh` reference
- [x] AC6 `.github/pull_request_template.md` exists with SDD checklist + skip-rationale section -> file present, headings verified
- [x] AC7 `scripts/install-precommit.sh` pre-push hook installed via `--with-sdd-gate` flag -> `tests/install-precommit.bats` cases 4-6 (flag in --help, unknown-flag reject, pre-push hook-type in conditional)
- [x] AC8 `tests/check-spec-gate.bats` covers the 4 outcome rows -> 16 tests total, all green
- [x] AC9 Existing bats suite remains green -> 645/645 pass
- [x] AC10 shellcheck clean on new script -> `shellcheck --severity=error` (CI severity) passes on `check-spec-gate.sh` and `install-precommit.sh`

## Test status

- New bats suite: `bats tests/check-spec-gate.bats` -> 16/16 pass.
- Augmented bats: `bats tests/install-precommit.bats` -> 7/7 pass (3 original + 4 new).
- Full regression: `bats tests/*.bats` -> **645/645 pass, 0 fail**.
- Shellcheck (CI severity error): `shellcheck --severity=error scripts/check-spec-gate.sh scripts/install-precommit.sh` -> clean.
- Manual smoke (local): the very PR opening this spec is the canonical self-test -> diff includes `specs/SDD-003-ci-spec-gate/` so AC2 path applies; expected: gate passes.

## Simulated PR scenarios

Outcomes captured against the local feature branch fixture in `tests/check-spec-gate.bats` (16 cases). Mapping to real-world PR shapes:

| Real-world PR shape | Test case | Expected verdict |
|---|---|---|
| Typo fix in README (5 LOC) | test 5 | OK, no spec required |
| New feature with `specs/AI-019-foo/proposal.md` + 200 LOC of code | test 6 | OK |
| Refactor 80 LOC of `scripts/utils.sh` without a spec | test 7 | FAIL with AGENTS.md link |
| Same as above but with `skip-sdd` label + 3-line rationale in PR body | test 8 | OK with skip log |
| Same as above with `skip-sdd` label but empty rationale | test 9 | FAIL with rationale-required message |
| dependabot bump of 6 lockfile entries | test 10 + test 13 | OK (label exempt + lock excluded) |
| 200-LOC bats test additions, no code change | test 11 | OK (tests/ excluded) |
| Moving an old spec to `specs/archive/` | test 12 + test 16 | OK (archive excluded from LOC, archive does NOT satisfy gate alone) |

## Decisions made during implementation

- **LOC formula: added + removed (total churn), not max() or just added.** Rationale: a 100-line refactor swap should trigger the gate. SDD-001's "~50-300 LOC of production diff" wording supports this interpretation. Captured in script comments implicitly via the `(added + removed)` arithmetic.
- **Pre-push, not pre-commit.** Spec-gate needs branch diff against `origin/main`; pre-commit fires on a single commit and would falsely fail intermediate work-in-progress commits. Pre-push fires once per push, matches CI semantics, and avoids friction during local TDD red-green-refactor cycles.
- **`pre-commit` framework (existing) over raw `.git/hooks/`.** Repo already uses `.pre-commit-config.yaml` for gitleaks + commit-msg validation; adding a sibling entry is zero-friction. Raw hooks would have duplicated install logic.
- **Basename match for lockfiles instead of full-path glob.** Caught by failing test 13: `*.lock` glob does not match `package-lock.json` (ends in `.json`). Refactor used `${path##*/}` to extract basename and matched npm/pnpm/go conventions. Vault lesson candidate (see below).
- **Workflow uses `env:` for all `${{ github.event... }}` interpolations** following security-reminder hook flag. Pattern enforced consistently across BASE_REF, SDD_LABELS, SDD_PR_BODY.
- **No `--no-verify` escape.** Even pre-push has `--no-verify` as a per-push escape. Acceptable: CI is the hard enforcement; local hook is just early warning. The label + rationale path is the auditable escape.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? Yes — "Filename glob `*.lock` does not match `package-lock.json` because the file does not end in `.lock`. Use basename extraction (`${path##*/}`) + explicit literal patterns for npm/pnpm/go conventions when filtering lockfiles." Caught by test 13; would be re-hit any time a glob filter is added in the future.
- [ ] ADR-worthy decision for `30-architecture/adr-XXX.md`? No — this is operational tooling for an already-decided pattern (`pattern-spec-driven-development`). SDD-001/002/003 are tiers within the same decision, not separate ADRs.
- [ ] New pattern candidate for `00_meta/patterns/`? Potentially — a CI spec-gate is a generic pattern any spec-driven repo could reuse. Defer until a second project would adopt it (currently only dotfiles). If `kubelab` or another repo adopts SDD, promote then.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-003-ci-spec-gate/` -> `specs/archive/SDD-003-ci-spec-gate/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Lesson promotion executed (see above)
