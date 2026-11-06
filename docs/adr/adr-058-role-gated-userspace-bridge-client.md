---
id: "ADR-058-role-gated-userspace-bridge-client"
type: adr
status: accepted
owner: manu
date: "2026-11-05"
issue: "mlorentedev/dotfiles#3219"
tags: [architecture, decision, networking, tools, git]
created: "2026-11-05"
---

# ADR-058: Role-Gated Userspace Bridge Client

## Status

Accepted

## Date

2026-11-05

## Context

The corporate Windows client cannot install native Tailscale. It reaches the
private Gitea forge through the portable `ts-bridge` userspace node and an SSH
bootstrap to the Headscale control plane.

The binary was an unmanaged copy at v1.19.0. It reached `READY`, but real Gitea
traffic timed out because that version lacked ts-bridge#444's HTTP-to-SOCKS
control/DERP adapter. A checksum-verified replacement with v1.20.0 restored
TLS-valid Gitea health and command-scoped Git access. Git still had no durable
URL-specific proxy, and dotfiles had no install/version owner for the binary.

The solution must preserve native/direct networking on other machines, keep
profiles and managed credentials under ts-bridge ownership, and remain safe
while older dotf binaries read the updated checkout.

## Options Considered

### A. Generic archive catalog source + machine role + Git drop-in

Extend the tool catalog with a checksummed `github-archive` source that extracts
one declared member. Add a `ts-bridge-client` machine role. That role selects the
tool and owns a separate Git include file containing only the Gitea URL proxy.

### B. Dedicated ts-bridge downloader and reconciler

Implement a one-off downloader, checksum verifier, version probe and retry loop
inside a ts-bridge-specific converge step.

Rejected: duplicates the catalog installer and violates ADR-036's one-channel
rule. Reconsider only if safe generic archive extraction cannot fit an atomic
change.

### C. mise GitHub backend

Install ts-bridge through mise's GitHub backend.

Rejected: the Windows archive does not publish an artifact attestation. This
repo already excludes the same class of Windows archive for herdr. Reconsider
when every Windows release archive is attested.

### D. Manual portable binary plus doctor warning

Keep the current copy and only detect version drift.

Rejected: detection without convergence fails fresh-machine bootstrap and
second-run idempotence. Reconsider only if ts-bridge gains a secure self-update
command that owns placement and checksums.

## Decision

1. Add a generic `github-archive` package source.
   - The catalog declares per-platform archive asset, release checksum manifest,
     and exact member path.
   - Extraction streams only the exact member into the staged destination; it
     never expands arbitrary archive paths onto disk.
   - Zip and tar.gz are supported. Missing, duplicate, directory, symlink or
     unsafe members fail closed.
   - Existing version-floor, checksum, retry, atomic placement and never-
     downgrade rules remain unchanged.

2. Add optional machine roles to catalog tools.
   - A no-argument install and converge select entries whose role set is empty
     or intersects `machine.json` roles.
   - Naming a tool explicitly remains an operator override.
   - The ts-bridge entry uses both the new source type and role. An older dotf
     ignores the unknown source type and safely skips it; it never installs the
     archive as a binary.

3. Add the known role `ts-bridge-client`.
   - Machines opt in with `dotf env role add ts-bridge-client`.
   - No role means no bridge proxy and no bridge-specific health requirement.

4. Own Git routing through a separate machine file.
   - The shared `git/dotfiles.gitconfig` includes
     `~/.config/git/ts-bridge.gitconfig`.
   - The role reconciler creates that file with only:
     `http.https://gitea.kubelab.live.proxy =
     socks5h://127.0.0.1:1081`.
   - Role removal deletes only this entirely dotf-owned file.
   - GitHub, DNS, hosts and the system proxy remain direct and untouched.

5. Keep ts-bridge runtime state outside dotfiles ownership.
   - Profiles, credential references, auth keys, tsnet state and SSH identities
     remain in ts-bridge and OpenSSH stores.
   - Doctor may report credential presence and bridge/destination health only
     by consequence; it never prints or copies credential values.

6. Deliver in atomic slices.
   - PR 1: generic `github-archive` source and tests.
   - PR 2: tool role filtering, `ts-bridge-client`, catalog entry and Git
     drop-in convergence/retirement.
   - PR 3: doctor version/readiness/destination diagnostics and end-to-end
     evidence.

## Rationale

This follows ADR-036/044/045: one declared install channel, dotf-owned
orchestration and one cross-OS converge path. The role follows the existing
self-deploy and ssh-target opt-in model. A separate Git include file gives the
proxy exactly one writer and makes role removal deterministic without replacing
the user/gh-owned `~/.gitconfig`.

The new source type is the safe rollout boundary. Old binaries already skip an
unknown source type with an upgrade message, so main can carry the entry before
the next release without blocking unrelated tools.

## Consequences

### Positive

- Fresh bridge clients install a pinned, checksum-verified binary.
- The second install/converge reports zero changes.
- Gitea Git routing is transparent only on opted-in clients.
- Native-Tailscale machines and GitHub remotes remain unaffected.
- Archive support becomes reusable for other checksum-manifested tools.

### Negative

- The catalog and installer gain a new source type and archive parsers.
- The change needs multiple atomic PRs before TOOL-034 can close.
- A bridge-client role cannot be enabled on an installed binary that predates
  the role; local activation waits for a release containing the reader.

### Neutral

- The running bridge remains a foreground/detached ts-bridge process; this ADR
  does not add an unattended service.
- Existing local profiles and credentials are neither migrated nor rewritten.

## References

- #3219
- ADR-012, ADR-020, ADR-025, ADR-036, ADR-041, ADR-044, ADR-045
- `pattern-setup-script-idempotence`
- `pattern-contract-defaults-per-machine-override`
- `10_projects/dotfiles/prestudy/tool034-reference-audit-matrix.md`
- ts-bridge#444
