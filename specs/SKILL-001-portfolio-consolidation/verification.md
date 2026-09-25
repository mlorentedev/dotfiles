---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - SKILL-001-portfolio-consolidation

## Evidence

- [x] AC1 -> vault commit: eight skills under `90_archive/skills/` with `status: archived`, 31 left in `00_meta/skills/`
- [x] AC2 -> `TestEverySkillTheRouterNamesHasARecord` (new, failed first on the real triggers and dependency map), `TestEveryDeclaredSkillHasARecord`, no `requires:` naming a retired skill
- [x] AC3 -> `dotf harness resolve-skills` on the three records, and the `enforce: warn` count in each record's frontmatter
- [x] AC4 -> the folded sections in the `adversarial-review`, `spec` and `new-ticket` records
- [x] AC5 -> build, vet, `go test ./...`, lint, `--check`, full bats
- [ ] AC6 -> after merge: `--deploy`, then every deploy target and the Copilot catalog read clean

## Test status

- `go build ./...`, `go vet ./...` (linux and windows), `go test -count=1 ./...`: pass.
- `golangci-lint run ./...` (0 issues), `shellcheck --severity=error scripts/compile-harness.sh`, `compile-harness.sh --check` (no harness drift), `check-bats-names.sh`, `check-doc-paths.sh`, `check-md-escapes.sh`: all pass.
- `bats tests/*.bats`, serial: 1,599 run, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine. The five files this change touches were re-run on the final tree: 192 run, 0 failures (`verify-setup.bats` skips outside its integration container, so CI's container job is where its edited assertions run).
- Vault: `vault-validate.py` reports the same 65 issues on master and on this change; none is new.
- TDD: `TestEverySkillTheRouterNamesHasARecord` was written first and failed, naming `project-maturation`, `writing-plans`, `audit`, `verification-before-completion`, `executing-plans` (all in `DefaultSkillDependencies`) and `enrich-us` (trigger `task-and-ticket-tracking`). It passed after the remap.
- `TestRoleJoinDrift` (HARNESS-110) went red on the slimmed builder roster: 12 of 18 rules resolved, against a floor of 16. See the first decision below; after it, 16 of 18 resolve again. `TestResolveRoles` now expects `pr-review-triage` to resolve to reviewer and shipper.
- Prompt hook, run against this branch's records (`DOTFILES_DIR=<worktree> dotf harness suggest --from-hook`): prompts about Go, firmware, an MCP server and asyncio still print `[persona] builder` and name the domain skill first (`golang-pro`, `debug-hardware`, `mcp-builder`, `async-python-patterns`). Control run with the new rosters and main's triggers: no output at all for the Go prompt.

## Size of what a persona is told to consume

Sum of the roster's `SKILL.md` files; tokens estimated at 4 characters each. Reference files a skill loads on demand are not counted.

| Roster | Before | After |
|---|---|---|
| builder | 9 skills, 53,311 B, about 13.3k tokens | 4 skills, 29,003 B, about 7.2k tokens (-46%) |
| planner | 6 skills, 61,859 B, about 15.4k tokens | 3 skills, 56,782 B, about 14.1k tokens (-8%) |
| reviewer | 4 skills, 36,955 B, about 9.2k tokens | 3 skills, 48,782 B, about 12.1k tokens (+32%: `pr-review-triage` joined, and `adversarial-review` absorbed two skills) |
| all active skills | 39, 304,054 B, about 75.6k tokens | 31, 282,514 B, about 70.2k tokens (-7%) |

## Decisions made during implementation

- **Four domain triggers gained a discipline skill from the builder's roster.** A prompt's persona is derived from `trigger.skills ∩ persona.skills` (HARNESS-110). With the domain skills off the builder's roster, the Go, Python, hardware and MCP rules resolved to no persona, and a rule with no persona prints nothing, not even its domain skill. That contradicts "available when their domain is detected". `golang-engineering`, `python-cli` and `mcp-tool-design` now also name `test`, and `hardware-and-embedded-debug` names `systematic-debugging`. Each is on the builder's roster, and none changes which skill the hook names first. The principled alternative is a persona key for associated skills that the join reads but the gate does not. It is a schema change, offered on the PR rather than made here.
- **Severities kept as they were.** A flat roster entry is `EnforceUnset`, which the gate neither enforces nor warns on, so rewriting the reviewer's and planner's warn-level entries flat would have switched them off.
- **Retired trigger skills remapped, not dropped.** `security-and-quality-audit` now names `adversarial-review`, which holds the checklist. `plan-authoring-and-execution` now names `spec`, which holds planning and execution. `task-and-ticket-tracking` keeps `new-ticket` and `prd-to-issues`.
- **`DefaultSkillDependencies` loses only the retired names.** `handoff` and `pr-review-triage` now depend on `adversarial-review` (the successor of what they depended on). The map was already out of step with the skills' `requires:` frontmatter before this change; that is HARNESS-147 (#1693), not widened here.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons/`? Candidate: a forced roster is also a routing table (HARNESS-110), so slimming it silently unroutes prompts unless a guard counts resolving rules.
- [ ] ADR-worthy decision for the repo's `docs/adr/`? no
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SKILL-001-portfolio-consolidation/` -> `specs/archive/SKILL-001-portfolio-consolidation/`
- [ ] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018)
- [ ] Promotions above executed
