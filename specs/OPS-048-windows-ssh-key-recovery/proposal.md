---
id: "OPS-048-windows-ssh-key-recovery"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-09-24"
issue: "mlorentedev/dotfiles#1658"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-048-windows-ssh-key-recovery

> **Naming**: file lives at `<repo>/specs/OPS-048-windows-ssh-key-recovery/proposal.md`. `OPS-048-windows-ssh-key-recovery` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1658: OPS-048: Persist and reconcile dedicated Windows SSH keys -->

A dedicated SSH key for a personal Windows host currently exists only on one workstation, so workstation loss would also lose the credential and force manual trust re-enrolment. The server-side OpenSSH setup is clipboard-driven, difficult to audit, and not safely repeatable. This change makes Bitwarden the recoverable source of truth and turns both client restoration and host authorization into idempotent, verified operations.

## What

- Register the dedicated private key as a Bitwarden-backed file secret that materializes to a stable OpenSSH path.
- Provide Windows scripts that verify/protect a restored client key and idempotently install/authorize its public key on an OpenSSH host.
- Provide stable SSH aliases and an operator runbook for first enrolment, reconciliation, recovery, rotation, and disaster recovery.

## Out of scope

- Automating the one-time trust bootstrap without an already authenticated administrative channel.
- Changing Tailscale routing, RDP configuration, or the target host's dual-homed network setup.
- Replacing the repository's existing Bitwarden/age governance or adding a new secret backend.

## Risks / open questions

- A Bitwarden-backed private key is intentionally materialized on disk for OpenSSH; Windows ACL reconciliation must compensate for the lack of meaningful POSIX mode bits on Windows.
- Windows OpenSSH's default `administrators_authorized_keys` store authorizes the key for administrator accounts as a group. The runbook must make this blast radius explicit and require a dedicated key.
- The first authorization cannot be circularly automated: it requires an existing trusted administrative session. The automation starts at that trust boundary and must be idempotent afterward.

## Acceptance criteria

- [ ] AC1: A replacement workstation can materialize the dedicated private key from Bitwarden at the declared path, enforce owner/SYSTEM-only ACLs, and verify its fingerprint against the committed public key without printing private material.
- [ ] AC2: Re-running Windows OpenSSH host reconciliation installs/enables the service and firewall rule, authorizes the public key exactly once, preserves unrelated keys, and enforces the required key-file ACL.
- [ ] AC3: Stable SSH aliases select the dedicated key and cover both mesh-name and dock-LAN access without requiring the operator to remember user, address, or identity-file details.
- [ ] AC4: The runbook defines the one-time trust bootstrap, normal reconciliation, clean-machine recovery, rotation/revocation, DR escrow, and verification signals, with no plaintext private key in git or command arguments.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`
- Related pattern: `00_meta/patterns/pattern-knowledge-placement.md`
