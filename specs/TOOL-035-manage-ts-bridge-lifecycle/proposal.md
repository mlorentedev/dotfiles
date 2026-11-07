---
id: "TOOL-035-manage-ts-bridge-lifecycle"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-06"
issue: "mlorentedev/dotfiles#3244"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# TOOL-035-manage-ts-bridge-lifecycle

> **Naming**: file lives at `<repo>/specs/TOOL-035-manage-ts-bridge-lifecycle/proposal.md`. `TOOL-035-manage-ts-bridge-lifecycle` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3244: TOOL-035: Manage ts-bridge client lifecycle for desktop agents -->

Machines using `ts-bridge-client` can install, route and diagnose the bridge,
but no dotfiles command owns its process lifecycle. When the detached SSH
bootstrap exited, the listener disappeared and every Copilot App private-origin
operation failed until the full launch command was reconstructed manually.

## What

Dotf gains operator-controlled `bridge start`, `bridge status` and `bridge
stop` commands. They persist only a safe launch descriptor and owned process
identity, reuse it idempotently, separate process/listener/destination states,
and never print or own profile credential content.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Installing an unattended service, Scheduled Task or login trigger.
- Creating, copying or rotating ts-bridge profiles, credentials, auth keys,
  SSH identities or tsnet state.
- Changing system DNS, browser proxy, hosts or the system proxy.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- PID reuse or a foreign listener must never let `stop` kill an unowned
  process.
- Windows and POSIX detach/termination semantics differ and require typed,
  synthetic process seams.
- The first start needs a profile/config reference; later starts may reuse the
  stored non-secret descriptor.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dotf bridge status` reports stopped, starting, listener-ready,
  destination-failed, healthy and unowned states with machine-readable
  `error`/`why`/`fix` fields and no credential values.
- [ ] A first `bridge start --profile ...` records a non-secret descriptor,
  returns only after `READY` plus TLS-valid destination health, and a second
  start is a no-op that spawns no duplicate.
- [ ] `bridge stop` terminates only the exact owned PID/process identity,
  becomes a no-op when absent and refuses stale/reused PID state.
- [ ] State and bounded logs live below the per-user dotf state directory and
  work on Windows, Linux and macOS through platform-specific seams.
- [x] Copilot App recovery is documented: after bridge health is restored, a
  project with stale origin identity is removed/re-added rather than repaired
  through internal database writes.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-059-operator-controlled-userspace-bridge-lifecycle.md`
- Related patterns: `pattern-setup-script-idempotence`,
  `pattern-agent-oriented-errors`
