---
id: "CHORE-002-changelog-policy-reconcile"
type: spec
status: archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# CHORE-002-changelog-policy-reconcile

> **Naming**: file lives at `<repo>/specs/CHORE-002-changelog-policy-reconcile/proposal.md`. `CHORE-002-changelog-policy-reconcile` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: AGENTS.md:231 says "NEVER create CHANGELOG.md in-repo" but `CHANGELOG.md` exists at root (added by PR #11). Decide: delete + migrate to vault, OR amend AGENTS.md. Effort: S. Anti-scope: policy decision only, not migration. -->

`AGENTS.md:231` is unambiguous: "**NEVER** create `docs/`, `TODO.md` or `CHANGELOG.md` inside the repo." Yet `CHANGELOG.md` exists at repo root (added by PR #11). Either the rule is wrong, the file is wrong, or the rule has an unwritten exception. Agents that read the rule but observe the file will be confused about future similar artifacts (when can I create a docs/?), and the SSOT discipline AGENTS.md enforces erodes from the contradiction.

## What

A **decision ticket** that ends with one of two states:

- **(a) Delete** — Remove `CHANGELOG.md`; migrate any content of lasting value to `10_projects/dotfiles/changelog.md` in the vault (or fold into `90-lessons.md` + git tags). AGENTS.md rule preserved as written.
- **(b) Amend** — Keep `CHANGELOG.md`; edit `AGENTS.md:231` to allow it with stated rationale (e.g., "CHANGELOG.md acceptable when generated from git tags via `changelog-gen.sh`").

The PR commits to one path. Both options are S effort.

## Out of scope

- **Adding release-please / semantic-release tooling** — separate ticket if "amend" wins and CHANGELOG becomes machine-generated.
- **Adjusting the `docs/` or `TODO.md` clauses of the same rule** — surgical to CHANGELOG.
- **Auditing other AGENTS.md rules for similar drift** — pattern-matching audits are a separate task.

## Risks / open questions

- **R1**: External tooling may EXPECT a root `CHANGELOG.md` (release-please, dependabot, GitHub release auto-attach). If (a) wins, audit `.github/` for any reference and either delete those references too or carve a `release/CHANGELOG.md` exception.
- **R2**: The existing CHANGELOG content has historical value (PR list, breaking changes, dates). Migration to vault should preserve it as a single archive note, not lose it.
- **R3**: `scripts/changelog-gen.sh` exists — was it ever intended to regenerate CHANGELOG.md? Audit its purpose. If yes, (b) may be the intended state and (a) regresses functionality.

## Acceptance criteria

- [ ] PR body's `## Decision` section picks (a) or (b) with rationale + cites the R3 audit outcome.
- [ ] If (a): `CHANGELOG.md` removed from repo root; vault has equivalent archive content; AGENTS.md unchanged.
- [ ] If (b): `CHANGELOG.md` retained; `AGENTS.md:231` amended with the stated exception clause; commit message explains the carve-out.
- [ ] No other AGENTS.md rules are touched.

## References

- Vault: `10_projects/dotfiles/11-tasks.md` § "Session 2026-05-27 — fresh-eyes audit" → CHORE-002.
- Conflict site: `AGENTS.md:231` vs. `CHANGELOG.md` (root).
- Related: `scripts/changelog-gen.sh` (purpose to be audited in R3).
