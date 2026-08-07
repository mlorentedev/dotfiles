---
id: "SDD-038-archive-on-merge"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-06"
issue: "mlorentedev/dotfiles#670"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ci, sdd]
template_version: "1.0"
---

# SDD-038-archive-on-merge

## Why

The SDD lifecycle's terminal step — archive on merge, Discipline Gate step 7 — is systematically dropped under delivery speed. Nothing enforces it, so `specs/` no longer describes what is in flight: it accumulates shipped work. Issue #670 was written at "57 active vs 75 archived"; today it is **45 active vs 96 archived**, so the ratio has not improved, it has only rotated. That is the argument for fixing the gate before sweeping: a one-time sweep with no gate starts rotting the day it merges.

The cost is not tidiness. `check-spec-gate.sh` already carries a `SPEC_FLOOR` heuristic whose own comment names this dependency — a stale active spec is an *alibi*: any large PR can touch ten lines of an unrelated shipped spec and satisfy the gate. Every un-archived spec widens that bypass surface. Archiving discipline is also this repo's differential advantage over spec-kit/Kiro, which have no lifecycle at all — and it only pays if enforced.

## What

`check-spec-gate.sh` gains a second, independent check: **a PR that closes an issue must archive the active spec that tracks it.**

Concretely: parse GitHub's closing keywords from the PR body; for each issue closed, find the active spec whose `issue:` frontmatter declares it; require that `specs/archive/<id>/` exists at the head ref. Failure is exit 1 with the exact `dotf spec archive` command to run.

Observable: a PR whose body says `Closes #123`, where `specs/FOO-001/proposal.md` declares `issue: "mlorentedev/dotfiles#123"` and the folder is still active, fails the gate. The same PR with the folder moved to `specs/archive/FOO-001/` passes.

## Design decisions

**Keyed on closing keywords, not on "a spec was touched."** Only a PR that *ends* the issue ends the spec's life. This distinction is load-bearing: PR #765 says `Refs #748` precisely because #748's remaining work was split to #766, and its spec must legitimately stay active. A gate keyed on mere spec presence would have forced a premature archive.

**Presence at the head ref, not a rename in the diff.** `git diff --numstat` only reports `specs/{ => archive}/<id>/…` when rename detection fires; the same move can surface as delete+add (heavy edits in the same commit, `diff.renames=false`, or a config-driven similarity threshold). Asking the head tree directly — does `specs/archive/<id>/` exist? — cannot be fooled by how git chose to render the change.

**Active specs are enumerated from base ∪ head, not base alone.** Base alone misses a PR that *creates* a spec and closes its issue in the same change — exactly the "created, shipped, never archived" pattern this gate exists to stop. An archived spec is no longer under `specs/<id>/` at head, so the union cannot produce a false positive.

**Runs before, and independently of, the LOC logic.** A three-line PR can close a spec'd issue, so the archive check must not sit behind the `TOTAL_LOC < THRESHOLD` early exit. It is also deliberately *not* skipped by `skip-sdd`: that label asserts "this change needs no spec", which says nothing about whether an existing spec's work is finished.

**Its own exemption**, mirroring the existing one exactly: the `skip-archive` label plus a non-empty `## Archive skip rationale` section in the PR body. Same shape as `skip-sdd` + `## SDD skip rationale`, so there is one convention to learn.

## Enforcement is prospective, and the spec says so

Only **16 of 44** active specs carry an `issue:` field, and in three different shapes — `"mlorentedev/dotfiles#426"`, `"dotfiles#161"`, and a bare unquoted `479`. The parser accepts all three (plus the full `https://github.com/owner/repo/issues/N` URL form), but a spec with no `issue:` field simply cannot be matched and is not enforced.

This is deliberate. `dotf spec init --issue N` has written the field since REFACTOR-012, so every new spec is covered; retrofitting the other 28 is sweep work, not gate work. A gate that fails on specs it cannot link would block PRs for a data problem the author did not create.

## Out of scope

- **The one-time sweep** of the ~21 specs whose issue is already closed. Mechanical, high file count, and it *depends* on this gate landing first; splitting keeps both diffs reviewable. It is also the gate's first real dogfood: the sweep PR closes #670, so it must archive this very spec to pass.
- Backfilling `issue:` frontmatter into the 28 specs that lack it (sweep work).
- Renaming stragglers such as `specs/CLI-019` (no slug) — noted in #670, belongs with the sweep.
- The duplicate `MEMORY-001-*` pair and the `AI-022-*` id collision: data defects, not gate defects.

## Risks / open questions

- **False positives block merges.** The check must not fire on `Refs #N`, `Part of #N`, `See #N`, or an issue number appearing in prose. Resolved: only GitHub's own closing verbs (`close[sd]`, `fix(e[sd])`, `resolve[sd]`) immediately preceding the reference count, matched case-insensitively.
- **Cross-repo references.** `Closes owner/other#5` must not be matched against this repo's specs. Resolved: a reference carrying a repo qualifier is only considered when the qualifier names this repo (bare `#N` is this repo by definition).
- **An empty PR body** (local pre-push runs, where `SDD_PR_BODY` is unset) must be a clean pass, not a failure — the same reason the existing skips tolerate an unset body.
- **Fail closed on git errors**, matching the existing gate's `#686/C3` lesson: if the base or head tree cannot be read, exit non-zero rather than silently passing.
- **The gate cannot verify the archive is *correct*** — only that the folder moved. Whether `verification.md` was actually filled in is a review concern, not a CI one.

## Acceptance criteria

- [ ] A PR closing an issue tracked by an active spec, without archiving it, fails with exit 1 and names the `dotf spec archive` command.
- [ ] The same PR passes once `specs/archive/<id>/` exists at head.
- [ ] Non-closing references (`Refs #N`, `Part of #N`, a bare `#N` in prose) do not trigger the check.
- [ ] All three `issue:` frontmatter shapes are matched, plus the full-URL form.
- [ ] A cross-repo closing reference is ignored.
- [ ] The check runs irrespective of the LOC threshold, and is not skipped by `skip-sdd`.
- [ ] `skip-archive` + a non-empty `## Archive skip rationale` passes; the label without the rationale fails.
- [ ] An empty PR body, or a closed issue with no matching active spec, is a clean pass.
- [ ] The existing gate's behaviour is unchanged (its full test suite stays green).

## References

- Bitácora board: `mlorentedev/dotfiles#670`
- `scripts/check-spec-gate.sh`, `.github/workflows/spec-gate.yml`, `tests/check-spec-gate.bats`
- Prior hardening this builds on: #686/C3 (fail closed on diff error), #686/C25 (`SPEC_FLOOR`, the alibi bypass this shrinks), #397 (rename normalisation)
- `AGENTS.md` — "Discipline Gate (NON-NEGOTIABLE)", step 7
- `dotf spec archive` (`cli/internal/spec`)

<!-- archived 2026-08-06 — PR: https://github.com/mlorentedev/dotfiles/pull/767 -->
