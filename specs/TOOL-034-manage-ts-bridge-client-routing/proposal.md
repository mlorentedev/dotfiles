---
id: "TOOL-034-manage-ts-bridge-client-routing"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-05"
issue: "mlorentedev/dotfiles#3219"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# TOOL-034-manage-ts-bridge-client-routing

> **Naming**: file lives at `<repo>/specs/TOOL-034-manage-ts-bridge-client-routing/proposal.md`. `TOOL-034-manage-ts-bridge-client-routing` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3219: TOOL-034: Manage ts-bridge client version and Gitea proxy routing -->

The corporate Windows client depends on a userspace ts-bridge because native
Tailscale requires administrator privileges. The portable binary drifted at
v1.19.0, which reached `READY` but could not carry real Gitea traffic; v1.20.0
fixed the bootstrap protocol boundary. Dotfiles currently owns neither the
binary version nor the URL-specific Git proxy, so a fresh or drifted machine
cannot converge itself.

## What

Dotf gains a generic checksummed `github-archive` catalog source, then uses it
for a role-gated ts-bridge client. Machines declaring `ts-bridge-client`
install the pinned binary and receive an entirely dotf-owned Git include file
that routes only Gitea through the loopback SOCKS listener. Doctor reports
binary drift, listener readiness and destination reachability without reading
or printing profile credentials.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Installing or configuring native Tailscale, a TUN adapter, DNS, hosts, or a system proxy.
- Copying, rendering or rotating ts-bridge profiles, managed credentials, auth keys, SSH identities or tsnet state.
- Installing an unattended bridge service; connection lifecycle remains operator-controlled.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- Rollout: pre-PR-1 binaries ignore the new `github-archive` source type; PR 1 must enforce catalog roles before PR 2 publishes a role-bound archive entry.
- Archive safety: extraction selects one exact regular-file member and never expands arbitrary archive paths to disk.
- Role safety: no role means no bridge proxy; role removal deletes only the dotf-owned drop-in.
- Runtime proof: listener `READY` is distinct from a TLS-valid Gitea destination probe.
- Atomicity: implementation is split into three PRs per ADR-058.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] `github-archive` verifies the release checksum before extracting one exact declared member from zip or tar.gz into the staged binary destination.
- [x] Missing, duplicate, directory, symlink, unsupported or unsafe archive members fail non-zero without replacing the installed binary.
- [x] An older dotf treats the new source type as an explicit skip; unrelated catalog tools continue converging.
- [x] A `ts-bridge-client` machine role selects the ts-bridge catalog entry; no role means the entry is not planned or installed.
- [x] Role convergence owns `~/.config/git/ts-bridge.gitconfig` with only the Gitea URL proxy, and role removal deletes that file.
- [x] GitHub stays direct; no hosts, DNS, environment-wide proxy or system proxy is changed.
- [x] Doctor reports pinned-version drift, bridge-listener readiness and TLS-valid Gitea destination reachability without credential values.
- [x] A second install/converge reports zero changes, and bridge-down behavior is explicit rather than healthy.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-058-role-gated-userspace-bridge-client.md`
- Related patterns: `pattern-setup-script-idempotence`, `pattern-contract-defaults-per-machine-override`
