---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - MEMORY-001-mirror

## Evidence

- [x] Criterion 1 (Claude Code SessionEnd hook) -> Wired in `ai/claude/settings.json` lines 18-24 invoking `dotf mem session-end`.
- [x] Criterion 2 (OpenCode continuity) -> Delivered via `/handoff` command deployed to `~/.config/opencode/commands/handoff.md`.
- [x] Criterion 3 (Antigravity continuity) -> Delivered via `/handoff` skill in harness.
- [x] Criterion 4 (Copilot CLI decision) -> Documented in `docs/runbooks/guide-cross-agent-memory.md` (daemon hooks excluded; manual /handoff per AGENTS.md).
- [x] Criterion 5 (Documentation) -> `docs/runbooks/guide-cross-agent-memory.md` created with architectural diagrams and matrix.
- [x] Criterion 6 (Cross-OS parity) -> `dotf mem` Go implementation covered by unit tests.

## Test status

- Go test suite: `cd cli && go test ./internal/mem/...` -> 100% pass
- Bats test suite: `bats tests/knowledge-crystallize-go-parity.bats` -> pass
- No regressions: yes

## Decisions made during implementation

- Formally documented the cross-agent memory lifecycle and agent integration matrix in `docs/runbooks/guide-cross-agent-memory.md`.
- Clarified that OpenCode and Copilot CLI lack background daemon process lifecycle hooks, using `/handoff` command and manual execution respectively to preserve memory single-sink architecture.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for `<area>/90-lessons.md`? <yes / no - one line of what>
- [ ] ADR-worthy decision for `<area>/30-architecture/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEMORY-001-mirror/` -> `specs/archive/MEMORY-001-mirror/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
