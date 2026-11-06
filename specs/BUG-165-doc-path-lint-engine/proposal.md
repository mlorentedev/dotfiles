---
id: "BUG-165-doc-path-lint-engine"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-05"
issue: "mlorentedev/dotfiles#3199"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-165-doc-path-lint-engine

> **Naming**: file lives at `<repo>/specs/BUG-165-doc-path-lint-engine/proposal.md`. `BUG-165-doc-path-lint-engine` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3199: BUG-165: doc-path guard process explosion blocks Windows commits -->

The doc-path pre-commit guard launches thousands of shell subprocesses while scanning instruction files, so commits can block for minutes or hours on Windows. If it remains shell-based, every growth in instruction content increases commit latency and encourages unsafe hook bypasses; the cross-OS guard must become a bounded single-process `dotf` command.

## What

`dotf lint doc-paths` reads checkout-owned policy, discovers governed tracked instruction files, parses backticked path claims in one process, and returns deterministic findings. The rollout first ships the command while the shell guard remains active, then a later retirement PR switches pre-commit and CI to the released pinned command and deletes the shell implementation and its BATS tests.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Changing which path/token shapes the existing guard accepts or rejects, except where current behavior is contradictory and covered by an explicit test.
- Adding a timeout that exits success, a permanent SKIP, or a shell wrapper around the Go engine.
- Implementing PLAT-008 worktree reconciliation or changing unrelated documentation guards.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- Deployment-order risk: PR 1 must leave the shell hook wired; pre-commit switches only after a released/pinned dotf contains `lint doc-paths`.
- Policy-drift risk: governed names, excluded prefixes, known extensions, and vault behavior live in `harness/doc-paths.json`, not compiled constants.
- Coverage risk: PR 1 ports every behavior case plus the governed-pattern discovery floor and zsh bare-source invariant; `cli.yml` is expanded to run the current-repository Go guard when governed docs or policy change. PR 2 then removes the old CI step, script, and both BATS files. No unresolved blocker remains.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] `dotf lint doc-paths` reproduces every accepted/rejected token, discovery, glob, escape, Windows-worktree, vault, governed-pattern-floor, and zsh bare-source outcome currently covered by BATS.
- [ ] The full current repository check completes in <=2 seconds on the affected Windows workstation and produces deterministic sorted output on Windows and POSIX.
- [ ] PR 1 ships the Go engine, checkout-owned `harness/doc-paths.json`, command tests, and current-repository CI coverage while the shell hook remains operational.
- [ ] After a release/pin containing the command, PR 2 switches pre-commit to `dotf lint doc-paths`, removes the old CI step, and deletes the shell script plus both BATS files with no dangling references.
- [ ] A change to any governed instruction file or `harness/doc-paths.json` triggers the Go guard before merge, including docs-only changes.
- [ ] Malformed policy, unreadable Git state, missing explicit files, missing paths, empty globs, and repo escapes exit non-zero; no timeout or inability-to-run path reports success.
- [ ] `harness/doc-paths.json` is versioned, decoded with unknown fields rejected, and an unreadable or unsupported version fails before scanning.
- [ ] Behavioral tests use synthetic temporary repositories/vaults; the only real-repository test is read-only and records the <=2 second performance measurement.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md`
- Related patterns: `pattern-setup-script-idempotence`, `pattern-verification-fails-toward-unproven`
