---
id: "WIN-014"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-28"
issue: "mlorentedev/dotfiles#1751"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# WIN-014: Reliable Windows harness mirroring

## Why

<!-- from issue #1751: WIN-014: dotf harness mirror finds the checkout from the cwd, drops the exec bit, and the Windows call site is silent without dotf -->

Setup must mirror the harness inputs from the checkout it is configuring, even
when launched outside that checkout or from another repository. Today the
command infers its source from the current directory, loses executable modes,
and can be silent on Windows when `dotf` is unavailable. Those failures leave
the deployed agent harness stale without a trustworthy remediation path.

## What

`dotf harness mirror --repo <checkout>` explicitly selects the checkout for a
mirror operation. Both setup twins pass their own checkout directory, mirror
files retain source executable permissions, and both report a clear warning
when the CLI needed for mirroring is unavailable.

## Out of scope

Things this PR explicitly does NOT include:

- Pruning stale harness files; `dotf doctor --fix` owns that behavior.
- Changing manifest target semantics or the deploy directory layout.
- Repairing arbitrary environment-variable drift outside the setup invocation.

## Risks / open questions

The explicit repository path must still fail clearly when it lacks the manifest.
Mode preservation must not rewrite destination files when both bytes and mode
match; it must rewrite byte-identical destinations when their mode differs.
Setup must preserve its current non-fatal behavior for a missing CLI while
making that skip visible.

## Acceptance criteria

- [ ] A mirror invoked outside a checkout or from inside another repository
  uses `--repo` and copies the named checkout's harness and manifest targets.
- [ ] An executable source file remains executable in the mirror.
- [ ] Linux and Windows setup pass their checkout paths to the mirror command
  and both warn when `dotf` is unavailable.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md`
- Related patterns: `00_meta/patterns/pattern-setup-script-idempotence.md`

<!-- archived 2026-09-30 — PR: https://github.com/mlorentedev/dotfiles/pull/1806 -->
