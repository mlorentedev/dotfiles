---
tags: [spec, verification, templates]
created: "2026-06-21"
---

# Verification - CLI-026-dotf-harness-engine

## Evidence

- [x] AC1 (skills block, slice 1) -> `tests/harness-skills-parity.bats` (5 tests): every output but the prompts byte-identical to `compile-harness.sh --deploy`; prompts differ only by the body rules the shell dropped; a prompt keeps every body rule; executable bits match; a second Go run reports `0 deployed`
- [x] AC2 (skills) -> `TestRenderSkillRecord_CRLFRendersTheLFBytes`
- [ ] AC3 copilot catalog -> slice 2
- [ ] AC4 agents, doctrine, deploy -> slices 4–5
- [ ] AC5 check -> slice 6
- [ ] AC6 `--against-vault` -> slice 6
- [ ] AC7 refresh in Go -> slice 7
- [ ] AC8 callers rewired -> slices 3 and 5
- [ ] AC9 bash engine deleted -> slice 7

## Test status

- Slice 1, 2026-10-20, macOS arm64: `go test ./internal/harness/ -run 'Skill|LoadSkills'` ok; `DOTF_BIN=… bats tests/harness-skills-parity.bats` 5/5 ok.
- Mutation check, slice 1: restoring the shell's "drop every `---`" in the prompt render fails `TestRenderSkillRecord`; restoring substring `targets:` matching fails `TestSkillTargetsAgent/inline_list_names_copilot,_not_pi_(#1733)`.
- Manual oracle run, slice 1: 35 records, 240 files on each side with copilot and pi on PATH; `diff -r -x prompts` empty, file modes identical, rerun `0 deployed, 204 current, 0 pruned`.

## Decisions made during implementation

- Slice 1: the copilot catalog moved to its own slice to keep slice 1 under the executable-line cap (about 293 lines).
- Slice 1: a record whose frontmatter is not valid YAML is now an error for that record (reported after the rest deploy); the shell never parsed it. Every committed record parses (`TestSkillTargetsAgent_EveryCommittedRecordParses`).
- Slice 1: an explicit empty or null `targets:` deploys to nobody, as the shell's substring match did.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? yes: `docs/lessons/lesson-502-a-fence-counter-that-never-stops-counting-eats-the-body.md`
- [ ] ADR-worthy decision? <to answer at archive>
- [ ] New pattern candidate for `00_meta/patterns/`? <to answer at archive>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-026-dotf-harness-engine/` -> `specs/archive/CLI-026-dotf-harness-engine/`
- [ ] Promotions above executed (if any)
