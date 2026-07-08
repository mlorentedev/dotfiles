---
id: "MEMORY-002-agnostic-memory-symlink"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-16"
issue: "dotfiles#402"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, memory, single-sink, cross-agent, symlink]
template_version: "1.0"
---

# MEMORY-002-agnostic-memory-symlink

> **Naming**: file lives at `<repo>/specs/MEMORY-002-agnostic-memory-symlink/proposal.md`. Sibling of `GUARD-001` (guards the sink) and `MEMORY-001-mirror` (per-agent capture). This one provides the **agnostic plumbing** that links each agent's memory dir to the sink.

## Why

<!-- from issue #402: Extract agent-agnostic memory-symlink setup + wire into dotf init -->

The vault→agent-memory symlink is created only by `scripts/claude-session-start.sh` (`ensure_memory_symlink`), so only **Claude Code** gets its `memory/` dir linked into the vault sink. The resolver that finds the vault source — three conventions: `10_projects/<name>/memory`, `CWD/memory` (in-vault sessions), and `50_work/45-development/<family>/<component>/memory` (nested work SDK) — is fully agent-agnostic, but it is trapped inside a Claude-only hook. No other agent and no scaffolder can reuse it. This is the agnostic gap in the single-sink convention: the sink exists, but only one agent is plumbed to it automatically.

## What

- **Standalone `scripts/ensure-memory-symlink.sh`**: given a project CWD and a target memory path, resolve the vault memory source (the three conventions, behaviour-preserved) and create the symlink idempotently. Agent-agnostic: the target path is an **argument**; the resolver knows nothing about Claude's path encoding.
- **`claude-session-start.sh` delegates** to it, passing Claude's `~/.claude/projects/<encoded>/memory` as the target. Claude behaviour is unchanged (parity).
- **`dotf init` wiring** (see R3): best-effort call so a scaffolded repo that already has a vault memory source gets linked immediately; no-op otherwise.
- **bats coverage** for the resolver (all three conventions), the symlink creation, and idempotency.

After this PR: `ensure-memory-symlink.sh --cwd <repo> --target <path>` creates the symlink when a vault source exists, no-ops idempotently otherwise; Claude sessions behave identically; the resolution logic is reusable by any agent's hook (the MEMORY-001-mirror follow-up just calls it with each agent's target).

## Out of scope

- **Per-agent SessionStart/SessionEnd hooks** for opencode / agy / copilot — this provides the reusable plumbing they will call; wiring each agent's hook is `MEMORY-001-mirror`.
- **Changing the three-convention resolution** — extracted as-is, behaviour parity is an explicit acceptance criterion.
- **The memory journal / capture** (`MEMORY-001-cross-agent-session-bridge`) — different mechanism.

## Risks / open questions

- **R1 — script contract.** Recommendation: `--cwd <dir>` + `--target <path>` flags (explicit, testable), `--project <name>` optional (defaults to `basename CWD`). One-line message on stdout when a link is created (the caller appends it to its own context output); silent no-op otherwise; **exit 0 always** — best-effort, never fail a session start. The current Claude function returns early on every miss; preserve that.
- **R2 — keep `encode_project_path` out.** That helper is Claude-specific (its hash encoding). The standalone script must NOT depend on it — the **caller** computes the encoded target and passes it via `--target`. This is what keeps the script agnostic and is the crux of the extraction.
- **R3 — what does `dotf init` pass? ✅ DECIDED (2026-06-16): descope, compose with #395.** Found during implementation: `dotf init` **already** eagerly links memory at scaffold time via `initrepo.linkMemory` (Go), which creates `~/.claude/projects/<encoded>/memory` → the vault entry it just wrote, mirroring Claude's path encoding. So there is no functional gap to fill — `linkMemory` is the *scaffold-time* link (single known convention), while `ensure-memory-symlink.sh` is the *session-start* resolver (three conventions for an existing repo). "Wire into `dotf init`" therefore means *unifying* two implementations, not adding a missing call. Since **#395** is actively moving `linkMemory`/`WriteVaultEntry` from `initrepo` to `cli/internal/vault` (strangler), touching it here collides. **Decision:** this PR does NOT touch `dotf init`; the unification composes with #395 — when `linkMemory` moves, evaluate delegating to `ensure-memory-symlink.sh` there.
- **R4 — cross-OS.** The script is POSIX shell (symlinks via `ln -s`); Windows uses a different memory layout and `ensure_memory_symlink` is not invoked there today. Scope the standalone script to the Linux/macOS path that exists now; note the Windows port as a follow-up rather than inventing untested behaviour.

## Acceptance criteria

- [ ] **AC1** — `ensure-memory-symlink.sh --cwd <repo> --target <path>` creates a symlink `target → <vault source>` when a vault source exists under any of the three conventions. *Verify:* bats fixture per convention.
- [ ] **AC2** — Idempotent + safe: already-linked → no-op; a non-empty real target dir → no-op (never clobbers data); no vault source → no-op exit 0. *Verify:* bats.
- [ ] **AC3** — Behaviour parity: `claude-session-start.sh` delegates to the script and a Claude-shaped fixture produces the same symlink the inline function did. *Verify:* bats against the Claude target convention.
- [ ] **AC4** — *(descoped — composes with #395, see R3).* `dotf init` already links memory at scaffold time (`initrepo.linkMemory`); unifying it with this runtime resolver is deferred to #395's move of that code into `cli/internal/vault`. Not touched in this PR. *Verify:* n/a here; tracked in #402 + #395.
- [ ] **AC5** — No behaviour regression in `claude-session-start.sh` (the inline function is removed, not duplicated). *Verify:* grep + existing session-start tests.

## References

- Issue: dotfiles#402 (work-gate)
- Source of the extracted logic: `scripts/claude-session-start.sh` → `ensure_memory_symlink` (lines ~300–363)
- Sibling specs: `GUARD-001-memory-sink-precommit` (#398), `MEMORY-001-mirror` (per-agent hooks, draft), `MEMORY-001-cross-agent-session-bridge`
- Single-sink decisions D1–D4 (handoff 2026-06-16)
