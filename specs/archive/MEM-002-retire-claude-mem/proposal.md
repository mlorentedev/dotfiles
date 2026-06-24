---
id: "MEM-002-retire-claude-mem"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-22"
issue: "mlorentedev/dotfiles#541"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, memory, claude-mem, retirement]
template_version: "1.0"
---

# MEM-002-retire-claude-mem

> Retire the `claude-mem` Claude Code plugin from this repo. Decision: hive
> `docs/adr/adr-016-cross-agent-memory-consolidation.md` Q2, ratified (minimal
> reading) in the 2026-06-22 architecture session (#540 / substrate epic #469).

## Why

claude-mem is the **L0** layer (auto-capture of conversation flow) and the **only
Claude-only** memory store. The architecture session resolved ADR-016 Q2 (L0
ownership): **drop L0**, lean on **L1** (session handoffs) + **L2** (lessons), both
already cross-agent in the vault. Accept the loss of `/mem-search` over raw
conversations (lesson search is unaffected).

Retiring claude-mem *removes a store* rather than moving write load onto hive's
currently-fragile write path, and stops further investment in the claude-mem
band-aid (ADR-016 "continuous simplification"). It also unblocks CLI-025 (#494):
`dotf mem heal` was the only claude-mem-coupled piece — now dropped.

## What

Remove every production wiring of claude-mem from this repo, and automate the
machine-level uninstall so existing installs converge to "no claude-mem" on the
next `setup`.

1. **Stop installing.** Remove the plugin + marketplace registration from
   `setup-linux.sh` and `setup-windows.ps1`.
2. **Automate uninstall (machine).** Add an **idempotent** cleanup to setup (shell —
   bootstrap layer, ADR-020 C7): uninstall `claude-mem@thedotmack` if present and
   remove leftover plugin/marketplace dirs. No-op on clean machines. Marked as a
   one-cycle cleanup (prune follow-up after rollout).
3. **Unwire SessionStart.** Delete the claude-mem heal block in
   `scripts/claude-session-start.{sh,ps1}` and the `claude_mem_heal` injector in
   `session-start-config.json`. All other injectors (SDD, doctor, hive, specs)
   stay; `additionalContext` output must remain valid.
4. **Delete heal scripts + tests.** `git rm scripts/claude-mem-heal.{sh,ps1}` and
   `tests/claude-mem-heal.bats` + `tests/claude-mem-heal-ps1.bats`.
5. **Strip Go doctor checks.** Remove `checkClaudeMem` / `resolveClaudeMemHook` /
   `newestVersionDirs` from `cli/internal/doctor/checks_deploy.go`, `runHeals` from
   `fix.go`, and the `--fix … claude-mem` references in `cli/internal/cmd/doctor.go`;
   drop the matching test cases/comments in `checks_test.go`.
6. **Config schema.** `env-contract.json`: remove claude-mem from the
   `CLAUDE_CONFIG_DIR` description; drop the `npm` optional-binary entry (only used
   for `claude-mem-heal` zod install — verify no other production caller).
7. **Docs.** Remove the "Claude-Only MCP: claude-mem" section from
   `ai/claude/CLAUDE.md`; drop the `(claude-mem)` mention in `cli/README.md`; move
   `docs/troubleshooting/claude-mem-*.md` to `docs/troubleshooting/archive/`
   (historical records — not deleted). `docs/lessons.md` BUG-01x entries stay (audit
   trail).
8. **Guard.** Add a guard-grep pinning that no production caller (`scripts/`,
   `setup-*`, `cli/`, hooks, CI, `session-start-config.json`) references
   `claude.?mem`, excluding `docs/`, `specs/archive/`, and historical ADRs.

## Out of scope

- The `dotf mem session-start/end` port — that is CLI-025 (#494), now heal-free.
- The deferred L2 retrieval index (FTS5/vector) — hive#263.
- Historical ADRs + `docs/lessons.md` entries — immutable records, untouched.
- Rewriting the vault `pattern-dual-memory.md` substrate model beyond noting the
  claude-mem axis is retired (cross-project knowledge — vault edit, separate).

## Risks / open questions

- **Mid-session live plugin.** claude-mem self-heals at SessionStart and is active in
  the current session. The repo change takes effect on next deploy/session; the
  machine uninstall should run **after** this session (or via next `setup`) to avoid
  yanking the running MCP. Document the one command.
- **`installed_plugins.json` dangling entry.** Removing dirs without the official
  `claude plugin uninstall` can leave a dangling registry entry. Prefer the official
  uninstall in the cleanup, dir-removal as fallback.
- **`npm` entry reuse.** Confirm `npm` in `env-contract.json` is not relied on by any
  other production path before deleting it.
- **Output equivalence.** SessionStart `additionalContext` must stay valid JSON after
  removing the heal block — smoke the hook post-edit.

## Acceptance criteria

- [ ] No production caller (`scripts/`, `setup-*`, `cli/`, hooks, CI,
  `session-start-config.json`) references `claude-mem`; guard-grep enforces it.
- [ ] `setup-*` no longer installs claude-mem and contains an idempotent uninstall/cleanup.
- [ ] `scripts/claude-mem-heal.{sh,ps1}` + their bats are deleted.
- [ ] `go build ./...` and `go test ./...` green after the doctor-check removal.
- [ ] SessionStart hook still emits valid `additionalContext` (smoke).
- [ ] claude-mem troubleshooting docs archived (not deleted); CLAUDE.md section removed.
- [ ] claude-mem uninstalled from this machine (verified absent from `~/.claude/plugins`).

## References

- Decision: hive `docs/adr/adr-016-cross-agent-memory-consolidation.md` (Q2)
- Issue: #541 · unblocks #494 · closes #439 (declined) + #232 (obsolete) · deferred index hive#263
- Removal map: 2026-06-22 architecture session footprint audit

<!-- archived 2026-06-24 — PR: https://github.com/mlorentedev/dotfiles/pull/544 -->
