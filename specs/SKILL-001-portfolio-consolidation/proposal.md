---
id: "SKILL-001-portfolio-consolidation"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1692"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# SKILL-001-portfolio-consolidation

## Why

<!-- from issue #1692: SKILL-001: Consolidate skill portfolio, retire redundancies and decouple persona forced rosters -->

The vault carried 39 skills, and the persona rosters told every agent to consume skills that did not fit its task. A compliant builder loaded Go, Python async, hardware and MCP guidance on every build (9 skills, 53,311 B). Review and planning each had near-duplicates, and three skills belonged to no persona. The audit `10_projects/ai-strategy/research/audit-00-meta-harness-gemini-3-8.md` section 6 (`SKILL-01`, `SKILL-02`) measured the cost; this change removes it without losing a rule.

## What

- Eight skills move to vault `90_archive/skills/` with `status: archived` and a banner naming their successor, leaving 31 active: `dispatching-parallel-agents`, `project-maturation` and `research-prompt` (no persona used them), `verification-before-completion` and `audit`, `writing-plans` and `executing-plans`, and `enrich-us`.
- Their rules move to the skills that remain:
  - `adversarial-review` gains "Evidence before claims", "Closing pass (Definition of Done)" and a code-level checklist.
  - `spec` gains "Planning the tasks" and "Executing the tasks", so the plan is `tasks.md` and never a `docs/plans/` file.
  - `new-ticket` gains "Enrich a thin item".
- Every existing severity is kept:
  - reviewer: `adversarial-review`, `cyclomatic-complexity` and `pr-review-triage`, all at `enforce: warn`;
  - planner: `spec` (warn), `new-ticket`, `prd-to-issues`;
  - builder: `test-driven-development` (warn), `test` (warn), `systematic-debugging`, `cyclomatic-complexity`.
  The builder's domain skills stay in the catalog, on demand.
- Prompts in those domains still route to builder. A prompt's persona is derived from `trigger.skills ∩ persona.skills` (HARNESS-110), so the four domain rules (Go, Python, hardware, MCP) also name a skill on the builder's roster (`test`, or `systematic-debugging` for hardware). The hook still names the domain skill first.
- Nothing that routes skills names a retired one. That covers the trigger rules, `DefaultSkillDependencies`, the `requires:` frontmatter, the Definition of Done sentence ("executed by the closing pass of the `adversarial-review` skill"), the persona bodies and `ROSTER.md`. A new test fails if a trigger or the dependency map names a skill with no record.
- `compile-harness.sh --refresh` drops the eight records, and `--deploy` prunes their rendered copies from every harness. No file in a deploy target is removed by hand.

## Out of scope

- Retiring `docker`, `helm` and `terraform`, consolidating the curator cluster, and fixing `pattern-loader` (`SKILL-03` in the audit).
- Replacing `cyclomatic-complexity` with a deterministic linter, and a `dotf verify` command for the closing pass.
- Raising any severity: a skill's `enforce` level is unchanged by this spec.
- The long-standing drift between `DefaultSkillDependencies` and the skills' `requires:` frontmatter. Only the retired names are removed here; the drift is HARNESS-147 (#1693).

## Risks / open questions

- **A flat roster entry disarms the gate.** A skill written without `enforce` is `EnforceUnset`, which the gate neither enforces nor warns about. The rosters in the request were written flat, which would have switched the reviewer's and planner's warnings off. Every severity is kept, and `TestEveryDeclaredSkillHasARecord` plus the persona migration tests still pass.
- **A forced roster is also a routing table.** Taking the domain skills off the builder's roster unrouted four trigger rules, and `TestRoleJoinDrift` caught it (12 of 18 resolving, floor 16). The data fix above restores 16 of 18. The schema-level alternative, a persona key for associated skills that the join reads but the gate does not, is left for the review of this PR to choose.
- **The rules must survive the move.** Each fold is a condensed rewrite, not a copy, so a rule could be lost in the rewrite. The archived originals stay in the vault with a banner pointing at the new section, and the review compares them.
- **Vault and repo land separately.** The vault commit lands when this PR opens; until it merges, a `--refresh` from another branch drops the same eight records. Peers were told, and the drop is the same change this PR carries.
- **The deploy targets are shared, and the last deploy wins.** After the merge, a `--deploy` from a worktree whose branch predates it renders the eight records back into `$HOME` for every session. The live peers were told to rebase before deploying. The general hazard, a deploy from a branch behind main undoing what main retired, is its own ticket.
- **A retired name is not ours alone.** Other tools install skills into the same directories, and one named `audit` turned the first form of the AC6 check red during the independent review. Deploy only removes what it rendered, which carries `generated_from: 00_meta/skills/<name>/SKILL.md` (a `from:` comment in gemini prompts). So the check reads that mark rather than the name.
- **`pr-review-triage` on the reviewer** is a skill for dispositioning reviewer output, and the reviewer never edits. Its persona body scopes it to judging other reviewers' findings, never applying them.

## Acceptance criteria

- [ ] **AC1** — `00_meta/skills/` holds 31 skills, and the eight retired ones live under `90_archive/skills/` with `status: archived`.
- [ ] **AC2** — `harness/skills/` has no record for a retired skill, and no trigger, dependency-map entry, `requires:` field or persona roster names one (`TestEverySkillTheRouterNamesHasARecord`, `TestEveryDeclaredSkillHasARecord`).
- [ ] **AC3** — the reviewer, planner and builder rosters are as listed above, with every existing severity preserved, and at least 16 of the 18 trigger rules still resolve to a persona (`TestRoleJoinDrift`).
- [ ] **AC4** — `adversarial-review`, `spec` and `new-ticket` carry the folded rules.
- [ ] **AC5** — `go test ./...`, the full bats suite, `compile-harness.sh --check` and the 8,000-character doctrine budget test pass.
- [ ] **AC6** — after merge and `compile-harness.sh --deploy`, no copy of a retired skill that this pipeline rendered remains in any deploy target of `harness/manifest.json`, and neither the Copilot catalog nor a forced roster names one. A same-named skill another tool installed is left alone.

## References

- Bitácora: `mlorentedev/dotfiles#1692` (see the `issue:` frontmatter field).
- Audit: vault `10_projects/ai-strategy/research/audit-00-meta-harness-gemini-3-8.md` sections 2 and 6.
- Precedent: HARNESS-021 D1 archived `agent-config-optimization` into `90_archive/skills/` the same way.
- Enforcement semantics: `cli/internal/harness/persona.go` (`EnforceUnset`), `gate.go`.
