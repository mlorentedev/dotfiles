---
tags: [spec, verification]
created: "2026-06-10"
---

# Verification - AI-022-pi-harness-slot

## Evidence

- [x] pi in `skills.deploy[]` -> `jq` check in features.json (AI-022-f1)
- [x] /spec renders to `.pi/agent/skills/spec/SKILL.md` as a regular copy -> bats "AI-022: pi gets /spec as a native skill"
- [x] `targets:`-restricted skill excluded from pi -> bats "AI-022: a Claude-only skill is NOT exposed to pi"
- [x] pi-installed sibling symlinks untouched by deploy -> bats "AI-022: deploy leaves pi-installed sibling symlinks alone"

## Test status

- `bats tests/skills-pipeline.bats` -> 10/10 ok (3 new pi tests)
- `bats tests/skills-pipeline.bats tests/compile-harness.bats tests/healthcheck.bats` -> 59/59 ok
- Full suite: only pre-existing/environmental failures (3 shell-profile, identical on main; session-start byte-equivalence flaky under full suite, passes standalone, untouched by this diff)
- shellcheck clean on `healthcheck.sh` + `compile-harness.sh`; manifest valid JSON

## Decisions made during implementation

- Single manifest entry instead of engine changes: both `compile-harness.sh --deploy` and the `setup-windows.ps1` port iterate `skills.deploy[]` generically, so pi inherits render/targets/prune behavior with zero engine code.
- `~/.pi/agent/skills` deliberately excluded from healthcheck's strict symlink sweep: pi's own installer manages sibling skills as symlinks into `~/.agents/skills` (agent-owned state). Guarded instead by a bats test asserting harness deploys land as regular copies and foreign links survive.
- Manifest edited textually (not via JSON round-trip) to keep the diff to 2 lines.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? no — pattern already captured (lesson 13, stop fighting agent filesystem expectations)
- [ ] ADR-worthy decision? no — ADR-010/ADR-012 already cover the parity + copy-deploy model
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-022-pi-harness-slot/` -> `specs/archive/...`
- [ ] Issue #161 closed by the PR
- [ ] Promotions above executed (none)
