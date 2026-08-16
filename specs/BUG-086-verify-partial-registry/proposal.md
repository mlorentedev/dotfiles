---
id: "BUG-086-verify-partial-registry"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-08-16"
issue: "mlorentedev/dotfiles#1004"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# BUG-086-verify-partial-registry

> **Naming**: file lives at `<repo>/specs/BUG-086-verify-partial-registry/proposal.md`. `BUG-086-verify-partial-registry` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #1004: BUG-086: dotf secrets verify aborts on registry validation, so it
can never report per-var health -->

`Registry.validate()` returns on the **first** bad secret and `ParseRegistry` propagates
that, so one malformed entry makes the whole registry unloadable. For `dotf secrets
verify` that is self-defeating: a health check whose job is to tell you what is broken
becomes the first thing to break, and it is taken down by an entry the caller never asked
about. Scoping does not help either, because validation runs at load time, before
resolution.

## What

`verify` reports a per-var status for every well-formed entry even when another entry is
malformed, showing the malformed one as a `FAILED` row naming the defect, and exiting
non-zero.

- `dotf secrets verify` — every healthy entry reports; each defect is one `FAILED` row.
- `dotf secrets verify <id>` — only the named ids are validated and resolved. A defect
  elsewhere in the file is neither reported nor fatal.
- `dotf secrets verify <malformed-id>` — reported as `FAILED`, not as "unknown id".
- Every other command is unchanged: a half-valid registry still fails loudly.

## Out of scope

- **Relaxing validation anywhere else.** `set`, `migrate` and `render` write, and a
  half-valid registry is exactly the state in which they must not run. Only `verify`
  takes the partial door.
- **Repairing malformed entries.** `verify` reports; it does not edit.
- **The original trigger.** The issue cites `GITHUB_PERSONAL_ACCESS_TOKEN` carrying
  `bw.folder: apps` instead of `Dotfiles/apps`. #982 inverted the taxonomy, so that value
  is now correct and the entry is well-formed. The issue says it itself — *"Fix 2 is the
  actual ticket; 1 is the trigger"* — and the trigger vanishing is an argument for
  fixing this, not against: the check is one typo away from being unusable again, with
  nothing to warn until it happens.

## Risks / open questions

- **Defective secrets must not reach `Entries()`.** The package treats validation as
  having happened — `parseFileMode` documents that a bad mode "is impossible in practice"
  because `validate()` rejected it. So the partial parser *excludes* defective secrets
  rather than flagging them in place; anything else trades a clear failure for an
  undefined one. Guarded by a test.
- **Two validation paths would be the real hazard.** A separate lenient validator would
  drift from the strict one exactly as the read and write paths drifted in BUG-084 (#993)
  — silently, and only under the case nobody exercises. Avoided structurally: the strict
  parser is implemented ON TOP of the partial one, so there is one set of per-secret
  checks and two policies over the result.
- **A rejected secret must not reserve its id or vars**, or the *next*, valid secret
  would be blamed as the duplicate. Registration happens only after a secret validates.

## Acceptance criteria

- [ ] AC1 — `verify` reports a per-var status for every well-formed entry when another
      entry is malformed; the malformed one is `FAILED`, exit is non-zero.
- [ ] AC2 — `verify <id>` validates and resolves only the named ids; an out-of-scope
      defect is neither reported nor fatal.
- [ ] AC3 — `verify <malformed-id>` reports it as `FAILED`, never "unknown id".
- [ ] AC4 — a duplicate id blames the *later* definition; the first still verifies.
- [ ] AC5 — no defective secret reaches `Entries()`.
- [ ] AC6 — the strict door is unchanged: `ParseRegistry` still fails on the first bad
      secret, so every write path keeps failing loudly.
- [ ] AC7 — a structural failure (unparseable YAML, unsupported version) still aborts
      rather than degrading into a report.
- [ ] AC8 — a duplicate id reports BOTH halves: the definition that validated resolves
      normally, and every defect sharing that id is reported. Added after review (#1020):
      because a rejected secret does not reserve its id (AC4), one id can name a valid
      entry AND one or more defects, and reporting only the defect hides the half that
      actually resolves.

## References

- Bitácora: `mlorentedev/dotfiles#1004`
- #992 — same family: an AC met by circumstance while its guard stays unbuilt
- #997, #906 — a real gap invisible because nothing checked
- #985 (BUG-080) — registry entries pointing at nonexistent Bitwarden items
