---
id: "WIN-016-sysupgrade-partial-result"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-17"
issue: "mlorentedev/dotfiles#3573"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "four reviewed findings of an already merged spec, one small PR (14 active, limit 10, 2026-11-17)"
---

# WIN-016-sysupgrade-partial-result

## Why

<!-- from issue #3573: WIN-016: system-upgrade keeps its partial result when the confirming listing fails -->

The independent review of `specs/2026-11-16-sysupgrade-windows-elevation` (#3553) passed with four Minor findings in the Windows path of the `system-upgrade` step. None is a defect today; each is a place where the step can report something untrue. Without elevation the step lists a second time to confirm what winget said it upgraded, and an error from that listing returned an empty result: the line lost the ids already upgraded and the ones left waiting. The probe listed a third time and discarded the answer, so that listing could only fail a step that had succeeded. And when every outdated id waited for elevation the line still read `catalog entries up to date`.

## What

The step's line says what happened in each of those cases: a failed confirmation keeps what winget reported and what waits, the probe makes no listing whose answer it ignores, and a catalog with entries left waiting is not called up to date.

## Out of scope

- Any change to which ids are attempted, or to the elevation rule (#3553).
- Splitting `systemUpgrade.windows` for its length: declined in the review's dispositions.
- Linux and apt. The wording change does reach a macOS cask that needs sudo, which shares `upgradeDetail`.

## Risks / open questions

- An id winget reported upgraded is counted as a change although unconfirmed. Resolved: `runStep` keeps a failed step's line only when `Changes > 0` (BUG-136), and the line names the ids as not confirmed, so nothing is claimed as upgraded.
- `dotf update` reads the detail for `converge.NeedsElevationNote`. That constant and the note's wording are unchanged.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1: when the confirming listing fails, the step fails naming the confirmation, and its result counts the ids winget reported upgraded, names them as not confirmed and keeps the ids left waiting.
- [x] AC2: without elevation the probe makes no winget listing, and a listing that would fail does not fail it.
- [x] AC3: a run or a plan in which every outdated id waits does not say the catalog is up to date; a catalog with nothing outdated still does.
- [x] AC4: a `winget list --scope machine` that exits with `NO_APPLICATIONS_FOUND` reads as no machine-wide install, and the outdated ids are attempted.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- `specs/2026-11-16-sysupgrade-windows-elevation/review.md`: the four findings
- #3553, #3562: the step this corrects
