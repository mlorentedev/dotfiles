---
tags: [spec, verification, memory, claude-mem]
created: "2026-06-22"
---

# Verification - MEM-002-retire-claude-mem

## Evidence

- [x] No production caller references claude-mem; guard enforces it
  -> `tests/guard-no-claude-mem.bats` (2/2 pass: ban-grep clean + cleanup-block-present anti-regression);
     direct `grep -rni 'claude.?mem' scripts/ setup-* cli/ session-start-config.json env-contract.json .github/` returns only the deliberate MEM-002 cleanup block.
- [x] `setup-*` no longer installs claude-mem and contains an idempotent uninstall/cleanup
  -> `bats -f MEM-002 tests/setup-linux.bats tests/setup-windows.bats` (5/5 pass): "no longer registers thedotmack marketplace", "ships idempotent cleanup block", "no longer deploys claude-mem-heal.ps1".
- [x] `scripts/claude-mem-heal.{sh,ps1}` + their bats deleted
  -> `git rm` (4 files: 2 scripts + claude-mem-heal.bats + claude-mem-heal-ps1.bats).
- [x] `go build ./...` and `go test ./...` green after doctor-check removal
  -> `go build ./...` OK; `go test ./internal/doctor/...` OK. (3 unrelated FAILs in initrepo/spec/vault are `TestEmbeddedTemplatesMatchVault` = pre-existing vault-template drift #461, not this change.)
- [x] SessionStart hook still emits valid `additionalContext`
  -> `echo '{}' | bash scripts/claude-session-start.sh` -> valid JSON; `contains claude-mem: False`.
- [x] claude-mem troubleshooting docs archived (not deleted); CLAUDE.md section removed
  -> `git mv docs/troubleshooting/claude-mem-{surrogate-400,broken-marketplace}.md docs/troubleshooting/archive/`; "Claude-Only MCP: claude-mem" section removed from `ai/claude/CLAUDE.md`.
- [ ] claude-mem uninstalled from this machine (verified absent from `~/.claude/plugins`)
  -> DEFERRED to after this session: the plugin is live in the current session (self-heals at SessionStart). Run `claude plugin uninstall claude-mem@thedotmack` (or re-run `setup`, which now carries the idempotent cleanup) once this session ends.

## Test status

- Go: `go build ./...` OK; `go test ./internal/doctor/...` OK. Full `go test ./...` has 3 pre-existing template-drift FAILs (#461), unrelated.
- Bats: `tests/guard-no-claude-mem.bats` 2/2; `bats -f MEM-002 tests/setup-*.bats` 5/5. (Full `setup-linux.bats` hangs headless — it launches Obsidian/network; environment limitation, not a regression. CI runs it properly.)
- Manual smoke: SessionStart hook emits valid JSON with no claude-mem block.
- No regressions: doctor package green; setup-bats claude-mem cases intentionally removed (17) and replaced with 5 MEM-002 anti-regression tests.

## Decisions made during implementation

- **`npm` kept, not deleted** from `env-contract.json` — it is used by obsidian-cli / yarn / opencode (setup-linux.sh), not only the retired heal. Only its `purpose` string changed.
- **Machine uninstall = idempotent one-cycle cleanup in `setup-*`** (shell, bootstrap layer, ADR-020 C7), not a `dotf` subcommand — a one-time retirement should not grow permanent Go surface. Marked for pruning after rollout; the guard's second test pins the block present so the guard can't pass by deleting it too.
- **Live-plugin caveat:** repo change takes effect next deploy/session; machine uninstall deferred to post-session to avoid yanking the running MCP.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no (the decision rationale lives in hive ADR-016 + #541).
- [ ] ADR-worthy? Decision already recorded in hive `docs/adr/adr-016` (Q2) — not duplicated here.
- [ ] New pattern for `00_meta/patterns/`? no.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/MEM-002-retire-claude-mem/` -> `specs/archive/MEM-002-retire-claude-mem/`
- [ ] Issue #541 closed (built-in workflow -> Done)
- [ ] Machine uninstall executed post-session
