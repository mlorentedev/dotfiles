---
tags: [spec, tasks, memory, claude-mem]
created: "2026-06-22"
---

# Tasks - MEM-002-retire-claude-mem

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [ ] Branch created from main: `chore/retire-claude-mem`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions blocking

## Implementation

### Guard first (red)

- [ ] Add the claude-mem ban guard-grep (production paths only; excludes docs/specs-archive/historical-ADRs). It is RED now (refs exist) → drives the removal.

### Unwire the hot path

- [ ] Delete the claude-mem heal block in `scripts/claude-session-start.sh`
- [ ] Delete the claude-mem heal block in `scripts/claude-session-start.ps1`
- [ ] Remove the `claude_mem_heal` injector from `session-start-config.json`
- [ ] `git rm scripts/claude-mem-heal.sh scripts/claude-mem-heal.ps1`
- [ ] `git rm tests/claude-mem-heal.bats tests/claude-mem-heal-ps1.bats`

### Go doctor checks (test-guided)

- [ ] Remove `checkClaudeMem` / `resolveClaudeMemHook` / `newestVersionDirs` from `checks_deploy.go` + the call in `checkSymlinks`; update the line-16 comment
- [ ] Remove `runHeals` from `fix.go` + its call site
- [ ] Update `cmd/doctor.go` `--fix` flag help (drop claude-mem)
- [ ] Drop claude-mem test cases/comments in `checks_test.go`
- [ ] `go build ./... && go test ./...` green

### Stop install + automate uninstall

- [ ] `setup-linux.sh`: remove marketplace add + plugin install; add idempotent uninstall/cleanup block
- [ ] `setup-windows.ps1`: remove marketplace add + plugin install; add idempotent uninstall/cleanup block
- [ ] Update `tests/setup-*.bats` assertions (no claude-mem install; cleanup present)

### Config + docs

- [ ] `env-contract.json`: strip claude-mem from `CLAUDE_CONFIG_DIR` desc; drop `npm` entry (verify no other prod caller first)
- [ ] `ai/claude/CLAUDE.md`: remove the "Claude-Only MCP: claude-mem" section
- [ ] `cli/README.md`: drop `(claude-mem)` from the `--fix` line
- [ ] Move `docs/troubleshooting/claude-mem-*.md` → `docs/troubleshooting/archive/`

### Machine

- [ ] Uninstall claude-mem from this machine (after session) + verify absent from `~/.claude/plugins`

## Closing

- [ ] Guard-grep now GREEN (no production refs)
- [ ] `go build`/`go test` green; bats green
- [ ] SessionStart hook emits valid `additionalContext` (smoke)
- [ ] `verification.md` filled
- [ ] PR opened referencing this spec folder
