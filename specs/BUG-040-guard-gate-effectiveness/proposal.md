---
id: "BUG-040-guard-gate-effectiveness"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-07"
issue: "mlorentedev/dotfiles#766"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-040-guard-gate-effectiveness

> **Naming**: file lives at `<repo>/specs/BUG-040-guard-gate-effectiveness/proposal.md`. `BUG-040-guard-gate-effectiveness` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #766: BUG-040: dotf doctor proposes a pre-commit install it can never apply, and checks file existence instead of gate effectiveness -->

The GUARD-001 wiring check asks "is `core.hooksPath` equal to the deploy mirror?" when the
question that matters is "is a GUARD dispatcher actually running?". The two diverge whenever
`core.hooksPath` points at a *different* directory holding the *same* dispatcher — the normal
state on a machine that develops the hooks from a repo checkout. Observed on the Windows box
2026-08-07: `core.hooksPath` = `C:/Users/mlorente/Projects/Workspace/dotfiles/git-hooks`,
target = `C:\Users\mlorente\.dotfiles\git-hooks`, `pre-commit` byte-identical in both
(SHA256 `CD3A25D3…37F3`) — yet every setup run prints *"the memory-sink guard is INACTIVE"*.
A security warning that cries wolf on every run is a security warning that gets ignored, which
is the failure mode #766 names: the check tests file identity, not gate effectiveness.

## What

The check reports the guard's real state, in three tiers instead of two:

1. `core.hooksPath` resolves to the deploy mirror → PASS, unchanged.
2. `core.hooksPath` resolves elsewhere **but that directory is a GUARD dispatcher** → PASS,
   naming the active path so the divergence stays visible.
3. `core.hooksPath` points at anything else → WARN and preserve, unchanged.

Tier 1 additionally normalizes the path before comparing, so separator direction (`/` vs `\`)
and trailing separators no longer produce a false negative on Windows. The same three tiers
land in all three implementations of this check, which are today independently wrong in the
same way: `cli/internal/doctor/checks_guard.go`, `scripts/install-git-hooks.sh`, and
`scripts/install-git-hooks.ps1`.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- The other half of #766 — doctor proposing a pre-commit install it can never apply. Different
  code path (`checks_vault_hooks.go`), different failure; #766 stays open for it.
- Auto-repointing `core.hooksPath` at the deploy mirror. The no-clobber rule stands: a global
  hooksPath has machine-wide blast radius and is only ever written when unset.
- `chain-local-hook.sh` behaviour inside linked worktrees (#776) — adjacent, separately tracked.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is
unresolved, do not move to `tasks.md` yet.

- **Resolved — what identifies "a GUARD dispatcher":** the presence of both `pre-commit` and
  `lib/memory-sink-guard.sh`. Structural, not a comment grep: `pre-commit` delegates the guard
  to that script, so its absence means the memory-sink guard genuinely cannot run. A marker
  comment would be editable without changing behaviour; this cannot.
- **Resolved — is tier 2 a PASS or a softer WARN?** PASS. The check's contract is "is the guard
  active", and it is. The active path is printed so a divergent setup stays visible rather than
  silent.
- **Accepted risk — tier 2 points at a working tree.** A branch checkout can then change the
  hooks under you. That is a property of the user's chosen setup, not something this check
  should override; naming the path in the PASS line is the mitigation.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] **AC1 — Equivalent dispatcher passes:** `core.hooksPath` at a non-mirror directory
      containing `pre-commit` + `lib/memory-sink-guard.sh` reports PASS naming that path, in all
      three implementations.
- [ ] **AC2 — Foreign hooksPath still warns:** a directory without the dispatcher still produces
      the preserve-and-WARN, and is never clobbered.
- [ ] **AC3 — Separator normalization:** `C:/…/git-hooks` and `C:\…\git-hooks` for the same
      directory compare equal and report the tier-1 PASS.
- [ ] **AC4 — No behaviour change on the settled paths:** unset → wired under `--fix` only;
      already-wired → idempotent, no re-write; dispatcher absent → FAIL.
- [ ] **AC5 — Regression asserted:** a test per implementation fails against the pre-fix code.

## References

- Bitácora board: [#766](https://github.com/mlorentedev/dotfiles/issues/766) (see the `issue:` frontmatter field)
- GUARD-001 origin: #398 (guard), #409 (dispatcher), #415 (verifier), #418 (inert-guard fix)
- Related: #776 (`chain-local-hook.sh` in linked worktrees), BUG-036 (`specs/BUG-036-precommit-under-global-hookspath/`)
- Related patterns: `00_meta/patterns/pattern-git-workflow.md`
