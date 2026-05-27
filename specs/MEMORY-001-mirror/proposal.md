---
id: "MEMORY-001-mirror"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# MEMORY-001-mirror

> **Naming**: file lives at `<repo>/specs/MEMORY-001-mirror/proposal.md`. `MEMORY-001-mirror` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: ([#117](https://github.com/mlorentedev/dotfiles/issues/117)) — Cross-Agent Session Memory Bridge dotfiles-side. Vault tracker: `vault/10_projects/knowledge/11-tasks.md § MEMORY-001`. **Dotfiles deliverables:** (a) implement `session-end` hook for opencode (`opencode.jsonc`) + AGY (`~/.gemini/settings.json`) + copilot (wrapper or CLI flag TBD); (b) `dotfiles/scripts/session-handoff.sh` reads hook payload, writes to `~/Projects/knowledge/00_meta/sessions/`; (c) cross-OS test parity (bats + Pester) reproducing session-end. **Spec:** promote to `specs/MEMORY-001-cross-agent-session-bridge/` after ADR-011 draft converges. **Esfuerzo:** ~3-5h. **Tier:** P0.5. **Depends_on:** ADR-011 (vault MEMORY-001 sub-task), SDD-008. -->

Each agent (Claude Code, OpenCode, Antigravity, Copilot) maintains its own session-memory format and lifecycle. Switching between agents loses context: a Claude Code session-handoff is invisible to OpenCode, OpenCode's recent thread doesn't surface in Antigravity, etc. The user pays a re-priming tax on every agent switch. ADR-011 (vault-side) proposes a cross-agent memory bridge: agents emit a standard `session-end` payload → a vault-side runner ingests → next agent's session-start reads the synthesized brief. This spec covers the dotfiles-side deliverable: the hooks + ingestion script + cross-OS test parity.

## What

Three dotfiles-side artifacts:

1. **Session-end hooks** — implemented per-provider:
   - OpenCode: `session-end` hook in `ai/opencode/opencode.jsonc`.
   - Antigravity (`agy`): equivalent in `~/.gemini/settings.json`.
   - Copilot: wrapper script or CLI flag (TBD; spec resolves).
   - Claude Code: already has SessionStart/SessionEnd hook capability; wire it.
2. **Ingestion script** — `scripts/session-handoff.{sh,ps1}` reads the hook payload (stdin or env var), normalizes to a canonical YAML shape, writes to `~/Projects/knowledge/00_meta/sessions/<timestamp>-<agent>.md`.
3. **Cross-OS parity tests** — bats (Linux) + Pester (Windows) reproduce a session-end → ingestion → vault write end-to-end with a mock payload.

## Out of scope

- **ADR-011 design** — that lives in the vault and predates this spec; this PR consumes it.
- **Cross-session memory injection** — i.e., next session's prompt receiving the prior session's brief. That's a separate vault-side render step (vault MEMORY-001 sub-task).
- **Migrating existing per-agent memory** — Claude Code's `MEMORY.md` stays; OpenCode's history stays. Bridge is additive.
- **Multi-machine sync of session memory** — orthogonal; vault sync handles it.

## Risks / open questions

- **R1**: Hook payload schema. Each agent emits a different shape (Claude's JSON vs OpenCode's tool format vs Antigravity's TBD). Normalize at ingestion-script boundary, not at hook boundary — keeps each agent's native format intact.
- **R2**: Copilot has no native session-end hook. Wrapper script may need to detect Copilot session end via process-exit signal. **Open question for proposal phase.**
- **R3**: Windows hook firing reliability. PowerShell process exits don't always run cleanup hooks (kill -9 equivalent skips them). Document the limitation.
- **R4**: Ingestion script runs on each session end → write contention if multiple agents close concurrently. Use atomic file write (write to tmpfile, rename).

## Acceptance criteria

- [ ] OpenCode `session-end` hook configured in `opencode.jsonc`.
- [ ] Antigravity `session-end` hook configured in `~/.gemini/settings.json`.
- [ ] Claude Code SessionEnd hook wired (extends existing SessionStart pattern).
- [ ] `scripts/session-handoff.{sh,ps1}` ingests payload + writes to vault.
- [ ] Bats + Pester tests pass end-to-end with mock payloads.
- [ ] `40-runbooks/guide-cross-agent-memory.md` documents the user-facing flow.
- [ ] Copilot integration: design decision documented (wrapper OR explicit "Copilot excluded for now").

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → MEMORY-001 (GH #117).
- GH: <https://github.com/mlorentedev/dotfiles/issues/117>.
- Upstream: vault ADR-011 (Cross-Agent Memory Bridge design).
- Dependency: SDD-008 (uses skill-pipeline render-all for hook deployment).
