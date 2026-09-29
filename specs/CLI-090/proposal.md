---
id: "CLI-090"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-09-28"
issue: "mlorentedev/dotfiles#1803"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-090: Cross-platform bootstrap and recovery updates

## Why

<!-- from issue #1803: CLI-090: Define and verify a cross-platform dotf bootstrap and recovery update flow -->

Updating a checkout with `git pull` does not update the `dotf` binary installed in
the user's local bin directory. A machine can therefore keep an obsolete CLI even
after its checkout advances. The raw POSIX release installer also fails when piped
to Bash because it depends on checkout-local helpers, so stale-binary recovery
cannot use the documented curl-style bootstrap path.

## What

The release installers become self-contained when fetched as raw scripts and can
resolve the latest published release when no checkout pin is available. POSIX and
PowerShell expose the same release-install contract through their native
interpreters: download a platform artifact, verify it against release checksums,
atomically replace the user-local binary, and report the installed version.
Checkout bootstrap remains a separate, explicit operation rather than a hidden
precondition of binary recovery.

## Out of scope

Things this PR explicitly does NOT include:

- A literal Bash payload for Windows hosts without Bash. Windows uses the
  PowerShell launcher with the same installer contract.
- Replacing ADR-019's opt-in, fast-forward-only self-update service.
- Full setup, secrets deployment, or migration of an arbitrary checkout.
- Solving the unrelated checkout-seeding defect tracked by #1660.

## Risks / open questions

Failure modes and constraints:

- GitHub's release API must yield a semver tag before an artifact URL is formed;
  malformed or unavailable metadata must fail without replacing an existing
  binary.
- Both launchers must retain explicit-version and checkout-pin behavior for
  offline test fixtures and reproducible setup runs.
- The raw script must not depend on checkout-local helpers.
- The expected final state is a release binary. A source-built `dotf` remains
  protected from automatic replacement.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1:** Piping the POSIX installer into Bash outside a checkout installs
  a checksum-verified explicitly selected release into an isolated `$HOME`.
- [ ] **AC2:** When no explicit version or checkout pin exists, both POSIX and
  PowerShell installers resolve the latest release before downloading its
  matching platform artifact; malformed release metadata fails loudly.
- [ ] **AC3:** A failed metadata, download, checksum, or extraction step leaves
  an already installed binary runnable and exits non-zero.
- [ ] **AC4:** Documentation names the native one-line recovery command for
  POSIX and Windows, distinguishes it from checkout bootstrap, and states the
  final version assertion.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-019-self-deploy-fast-forward-only.md`
- Related ADR: `docs/adr/adr-036-install-channels.md`
- Related patterns: `00_meta/patterns/pattern-setup-script-idempotence.md`
- Related patterns: `00_meta/patterns/pattern-version-single-source.md`
