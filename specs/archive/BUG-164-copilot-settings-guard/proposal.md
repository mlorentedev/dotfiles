---
id: "BUG-164-copilot-settings-guard"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-05"
issue: "mlorentedev/dotfiles#3193"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# BUG-164-copilot-settings-guard

> **Naming**: file lives at `<repo>/specs/BUG-164-copilot-settings-guard/proposal.md`. `BUG-164-copilot-settings-guard` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3193: BUG-164: unsupported local Copilot setting blocks every Desktop session -->

GitHub Copilot Desktop rejected the locally configured managed-policy-only key
`telemetry.disabled` before it could create any session, so every repository and plain
folder failed identically. The active file is repaired, but the current deploy merge
preserves unknown user keys forever and `copilot config --list --json` silently omits
this invalid key with exit 0, so neither setup nor doctor prevents recurrence.

## What

`dotf` reads a versioned checkout-owned list of retired Copilot local setting paths.
Doctor reports an installed retired path by file and key only; `doctor --fix`,
`dotf deploy`, and the configs converge step remove only those paths through the
existing staged JSON edit, one-time backup, and atomic rename. The same repair seam
runs on Windows, macOS, and Linux without invoking the Copilot CLI or reading live
authentication data.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

- Changing GitHub-managed telemetry policy or Copilot Desktop's internal database.
- Replacing `settings.json` wholesale or removing undeclared user settings.
- Changing the `ai/deploy.json` schema before the current released binary supports it.

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

- Rollout: the policy must be additive data ignored by older binaries; this PR must not bump the deploy manifest version or make `dotf 0.79.0` reject main.
- Safety: malformed/unreadable settings or policy files fail loud; no path may be reported with its value.
- Ownership: repair runs only for the existing `copilot-settings` merge entry and uses its declared destination/mode, backup, and atomic write behavior.
- Idempotence: pruning an empty parent object is allowed only after its declared retired leaf is removed; unrelated siblings remain untouched.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] A synthetic Copilot home reproduces `telemetry.disabled` without live credentials or user data.
- [x] A versioned checkout policy declares retired local setting paths and rejects malformed, unknown, unsupported, empty, duplicate, or unsafe paths.
- [x] Doctor reports each retired key by settings file and dotted key only; no setting value appears in output.
- [x] `doctor --fix` removes only declared retired leaves, prunes only newly empty parents, preserves every other setting, and keeps the standard `.pre-dotf` backup.
- [x] Bare/named `dotf deploy` and configs converge apply the same repair before reporting `copilot-settings` healthy.
- [x] A second repair/deploy/converge run reports zero changes.
- [x] Missing settings are left for normal deploy creation; unreadable, malformed, or non-object settings fail loud and are never treated as empty.
- [x] No validation path invokes or trusts `copilot config --list`.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md`
- Related patterns: `pattern-verification-fails-toward-unproven`, `pattern-setup-script-idempotence`

<!-- archived 2026-11-05 — PR: https://github.com/mlorentedev/dotfiles/pull/3218 -->
