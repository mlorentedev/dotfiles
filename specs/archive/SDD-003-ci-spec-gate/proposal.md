---
id: "SDD-003-ci-spec-gate"
type: spec
status: archived
created: "2026-05-19"
tags: [spec, proposal, sdd, ci]
template_version: "1.0"
---

# SDD-003-ci-spec-gate

## Why

SDD-001 (PR #49) and SDD-002 (PR #51) shipped Tiers 1-3 of the five-layer Spec-Driven Development enforcement: prose rule in `AGENTS.md`, per-session `[sdd]` reminder injected via SessionStart hooks, and settings.json portability. All three are **soft** layers — they nudge agents and the user, but a PR that ignores them still merges. The Discipline Gate documented in `AGENTS.md` is non-negotiable on paper and unenforced in practice. Tiers 4-5 close the loop: a CI check that fails any PR ≥50 LOC without a matching `specs/<feature-id>/` folder, and a PR template that surfaces the SDD checklist at draft time so violations are caught before push, not after.

## What

After this PR merges, every pull request opened against `main` runs a new `spec-gate` job that:

1. Computes net diff LOC against `main`, excluding `tests/`, `specs/archive/**`, `*.lock`, `*.lockb`, `.gitignore`, `CHANGELOG.md`, and files matching `**/*generated*`.
2. If LOC ≥50, requires that the PR diff contains at least one file under `specs/<feature-id>/` (active spec folder), where `<feature-id>` matches `^[A-Z]+-\d+(-[a-z0-9-]+)?$` or `^\d{4}-\d{2}-\d{2}-[a-z0-9-]+$`.
3. **Escape hatch**: PR carries label `skip-sdd` AND PR body contains a non-empty `## SDD skip rationale` section. The label is visible in PR history; the rationale is auditable. Both required — neither alone is enough.
4. Fails loud with a link to `AGENTS.md` Discipline Gate when violated.

A new helper `scripts/check-spec-gate.sh` encapsulates the LOC computation + spec-folder presence check, taking `--base-ref` and `--head-ref` flags. The CI workflow calls it; `scripts/install-precommit.sh` gains an opt-in hook that calls the same script with `--base-ref origin/main` so contributors can catch violations before push.

A new `.github/pull_request_template.md` lists the SDD checklist (vault entry, spec folder, proposal.md filled, label+rationale if skipping) so the gate's requirements are visible at PR draft time, not discovered post-push.

## Out of scope

- Touching tiers 1-3 (`AGENTS.md` prose, SessionStart `[sdd]` reminder, settings.json merge). They are already in main.
- Branch protection rule changes (requires admin GitHub UI; tracked separately, not part of this PR).
- Auto-generating spec folders from PR titles — out of scope; spec scaffolding stays a deliberate `init-spec.sh` step.
- Validating proposal.md content quality. Gate checks **presence** of the folder, not whether the proposal is well-written. The Socratic `/spec fill` flow handles quality.
- Public-contract path detection beyond LOC threshold (e.g. flagging changes to `env-contract.json` regardless of LOC). User selected the simpler LOC-only trigger; can revisit in a follow-up if drift appears.

## Risks / open questions

- **Risk: dependabot/renovate PRs trigger the gate.** Dependency bumps are usually small (<50 LOC) so unlikely, but if a multi-package update lands ≥50 LOC, the gate fires. **Mitigation**: dependabot PRs auto-label `dependencies`; CI workflow exempts that label (same pattern as `skip-sdd`).
- **Risk: bats test files counted toward LOC.** Mitigation already in scope: `tests/` excluded from the LOC computation.
- **Risk: PR with both spec folder AND skip label**. Treat label as override (skip wins). Logged in CI output.
- **Risk: rename refactor with 100 LOC of moves but no real change**. Gate cannot distinguish. **Accepted**: this is exactly the kind of change SDD-001's Discipline Gate says should have a spec ("first step of a multi-PR sequence" criterion likely applies anyway). If not, use `skip-sdd` label with `## SDD skip rationale: mechanical rename, no logic change`.
- **Open: where exactly does the gate compare against?** `${{ github.event.pull_request.base.ref }}` (usually `main`). Documented; no ambiguity at runtime.

## Acceptance criteria

- [ ] `scripts/check-spec-gate.sh --base-ref REF --head-ref REF` exits 0 when PR diff <50 LOC (excluded paths applied) regardless of `specs/` folder presence.
- [ ] Same exits 0 when PR diff ≥50 LOC and at least one file under `specs/<feature-id>/` is in the diff.
- [ ] Same exits 1 with explanatory message when PR diff ≥50 LOC and no `specs/<feature-id>/` folder is touched.
- [ ] Same exits 0 with informational message when the GitHub PR has label `skip-sdd` AND a non-empty `## SDD skip rationale` section in the body (label/body sourced via env vars for testability).
- [ ] `.github/workflows/spec-gate.yml` invokes the script on `pull_request` (opened, synchronize, reopened, labeled, unlabeled, edited). Job status visible in PR checks.
- [ ] `.github/pull_request_template.md` exists with SDD checklist + skip-rationale section header.
- [ ] `scripts/install-precommit.sh` installs a pre-push hook (not pre-commit — needs the full branch diff) that runs `check-spec-gate.sh --base-ref origin/main --head-ref HEAD`.
- [ ] New bats file `tests/check-spec-gate.bats` covers the 4 outcome rows above (≥4 test cases, all green).
- [ ] Existing 396-test bats suite remains green (no regression).
- [ ] `shellcheck` clean on the new script.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` (SDD-003 backlog entry, line ~28).
- Related ADR: none yet — this is operational tooling for an already-decided pattern.
- Related patterns: `00_meta/patterns/pattern-spec-driven-development.md` (the pattern being enforced).
- Prior tiers: PR [#49](https://github.com/mlorentedev/dotfiles/pull/49) (Tier 1+2), PR [#51](https://github.com/mlorentedev/dotfiles/pull/51) (Tier 3).
- Convention precedent: `scripts/diff-check.sh` and `scripts/doctor.sh` follow the same reusable-helper-plus-CI-caller pattern.

<!-- archived 2026-05-20 — PR: https://github.com/mlorentedev/dotfiles/pull/60 -->
