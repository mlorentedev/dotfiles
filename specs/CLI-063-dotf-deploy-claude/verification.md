---
tags: [spec, verification, templates]
created: "2026-09-04"
---

# Verification - CLI-063-dotf-deploy-claude

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **The command home moved from `dotf agent claude sync` to `dotf deploy` before any code was
  written.** #1339's framing rested on three premises that measurement falsified: ADR-032:122-124
  reserves `dotf agent` for the quota-spending run side; `strategy: merge` already exists (AI-039,
  in production on three configs); and `ai/deploy.json`'s own `$comment` names MCP registration as
  slice 3 of CLI-039. Owner decision 2026-09-05.
- **The scope shrank as a result.** The "largest and subtlest" increment is not a 100-line jq port:
  `mergeInto` already implements 3 of Claude's 6 key policies. What is missing is three per-key
  behaviours (`env`, `enabledPlugins` nested merge; `permissions.allow` union) the manifest cannot
  yet express.
- **Golden characterization capture was ruled out deliberately**, not skipped. Four capabilities
  running at four points of a setup, two of them shelling out to `claude`, have no single
  capturable stream. See `tasks.md` for the per-capability oracle chosen instead.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-063-dotf-deploy-claude/` -> `specs/archive/CLI-063-dotf-deploy-claude/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
