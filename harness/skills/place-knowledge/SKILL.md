---
name: place-knowledge
description: Onboard a repository to the knowledge-placement model — move its build/operate docs (ADRs, runbooks, troubleshooting, lessons) out of a personal knowledge store into the repo's `docs/`, leaving the store with only the cross-project brain. Triggers on "/place-knowledge", "onboard <repo> to the placement model", "migrate knowledge to repo docs", "move ADRs/lessons/runbooks out of the vault", "make this repo's docs self-contained / share-ready". Implements pattern-knowledge-placement; runs the onboard-project-to-placement-model runbook. Validated on 11 repos (KPM-001, 2026-05-28).
---

# Skill: place-knowledge

> Operationalizes [[pattern-knowledge-placement]] via the runbook [[onboard-project-to-placement-model]]. One repo per invocation.

## When to use

- Onboarding an existing repo whose ADRs/lessons/runbooks live in a personal knowledge store (vault) and should move into the repo `docs/`.
- A new org/project where you want repos self-contained and agent-agnostic from day 1.

## When NOT to use

- A repo that has no remote / no local clone (no migration target — leave knowledge in the store, document it).
- Pure decide/position artifacts (feasibility studies, brainstorms, strategy) — those STAY in the store.

## The model (one-line cut)

**decide/position -> store · build/operate -> repo `docs/` · collaborate -> forge (issues/milestones).** Directionality invariant: the store links OUT to repos, never the reverse.

## Procedure

Follow [[onboard-project-to-placement-model]] exactly. Summary:

1. **Branch from fresh `origin/<default>`** (never current HEAD — avoids sweeping another session's WIP). Verify `git log origin/<default>..HEAD` is empty before starting.
2. Create `docs/{adr,architecture,runbooks,troubleshooting}` (only buckets with content).
3. Copy per the placement map: ADRs (`30-architecture/**/adr-*.md` or numbered `NNN-*.md`) -> `docs/adr/`; non-ADR architecture -> `docs/architecture/` (preserve subdirs); `40-runbooks/` -> `docs/runbooks/`; `50-*` -> `docs/troubleshooting/`; `60-resources/` -> `docs/`; `90-lessons.md` -> `docs/lessons.md`; images -> `docs/architecture/<subdir>/`. Skip `_index.md`.
4. **Rewrite cross-refs** (the crux): to an artifact that ALSO moved -> relative repo path; to one that STAYS in the store -> plain-text provenance, **NO live link** (directionality invariant). Skip code fences/inline code (don't corrupt bash `[[ ]]` / TOML `[[x]]`).
5. Add `docs/README.md` index. Re-point the repo README + agent file (`AGENTS.md`/`CLAUDE.md`) to `docs/`; delete repo->store paths.
6. **Stub the store side**: overwrite each moved `.md` in place with a pointer stub (same basename) so inbound `[[wikilinks]]` resolve. Update the project `00-context` to state docs moved to the repo.
7. Verify: `grep -rn '\[\[' docs/` clean; no live store paths in `docs/`; store-side `vault-validate` no new issues.
8. Ship: commit (**NEVER a `Co-Authored-By` trailer** — see [[feedback_pr_no_attribution]]), push, open PR (`skip-sdd` rationale; label only if it exists; **no attribution in the PR body**; do NOT auto-merge — verify the commit is clean first).

## Automation

A deterministic reference implementation exists: the copy + cross-ref-rewrite script used for KPM-001 (`/tmp/kpm_migrate.py` during that epic). For a fleet migration, prefer running it from the **main agent** session — subagents are often sandboxed out of sibling repos (per [[feedback_agent_git_hygiene]]).

## Generalization (donable to any org)

Tool-agnostic: store = vault/Confluence/Notion/private-docs-repo; forge = GitHub/GitLab/Jira; repo `docs/` = universal docs-as-code. The pattern describes the model; this skill + runbook describe the steps.

## References

- [[pattern-knowledge-placement]] — the model
- [[onboard-project-to-placement-model]] — the full runbook
- [[pattern-three-layer-proposal-lifecycle]] — process view
- [[feedback_pr_no_attribution]], [[feedback_agent_git_hygiene]] — hard rules learned during KPM-001
- Epic: `10_projects/knowledge/specs/KPM-001-knowledge-placement-migration/`
