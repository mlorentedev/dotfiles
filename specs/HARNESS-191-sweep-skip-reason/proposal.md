---
id: "HARNESS-191-sweep-skip-reason"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3263"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "owner chose the spec route on 2026-11-11 for a fix already written and parked; the PR archives this spec, so the slot frees when it lands (10 active, limit 10, 2026-11-11)"
---

# HARNESS-191-sweep-skip-reason

> **Naming**: file lives at `<repo>/specs/HARNESS-191-sweep-skip-reason/proposal.md`. `HARNESS-191-sweep-skip-reason` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3263: sweep: park detaches without --force and falls back to an ancestry test that cannot see squash merges -->

`dotf worktree sweep` parks a named lane in place once its work has landed, and when it cannot (a dirty tree, a landing the forge could not confirm, a lease it could not write, a detach git refused) the lane is one more in `skipped N` and nothing says which lane or why. #3310 pinned the behaviours #3263 asked for (no forced detach, the forge asked where ancestry cannot see a squash merge); what it left open is the reason. An operator reading `reaped 0, 2 skipped` after a merge cannot tell a lane that is still dirty from one whose landing the forge could not answer, and the second is the one that silently keeps a lane busy for days.

## What

- `SweepReport` carries `LaneSkips` (`lane_skips`, omitted when empty): one entry per named lane that reached the park path and was not parked, with its path and the reason.
- The reason is one of: `has uncommitted changes or untracked files`; the landing reason `branchLanding` already produces (`could not check for a merged pull request: …`, `not landed: neither an ancestor of the base nor the head of a merged pull request`, `its tip could not be read`) or `its landing could not be decided` when it produces none; `its lease could not be written: <err>`; `git refused to detach it: <git's first stderr line>`.
- `dotf worktree sweep` prints each entry as `  - skipped: <path>: <reason>`, the line the session and merge-unknown skips already use, before the totals.
- The landing decision lives in `laneLanded`, apart from the parking it gates; `parkSingleNamedLane` returns the verdict and the reason together. The parking itself is unchanged: on a refused detach the lane's lease is restored and its branch kept.

## Out of scope

- Any change to what sweep parks or reaps, or to the gates that decide it (#3310 pinned those).
- The extraction of `Park`, `parkSingleNamedLane` and `switchLaneLocked` under 40 lines with characterisation tests (#3265): `laneLanded` is its first seam, nothing more is moved here.
- Worktree identity and the squash-merge half of #3293 beyond what the reason already says.
- The `--json` shape of `dotf worktree list`; only the sweep report gains a field.

## Risks / open questions

- A reason read from git's stderr is a free-form string: the report keeps only the first line, so one lane is one printed line, and the tests assert the stable prefix (`git refused to detach it: `) and the token that names the cause (`index.lock`), never git's wording.
- `branchLanding` returns an empty reason for a worktree on `main` or one whose branch cannot be read; the park path never reaches it for `main`, and the fallback names the undecided landing rather than printing an empty reason.
- Adding a field to the sweep JSON is additive and `omitempty`; a consumer of the previous shape reads the same document when no lane was skipped.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] AC1: a lane with uncommitted changes or untracked files is not parked, its state stays ACTIVE, and the reason names the dirty tree.
- [x] AC2: a lane whose landing the forge could not confirm is not parked and the reason carries the forge's error (`could not check for a merged pull request: …`); a lane whose tip the landed heads name is parked.
- [x] AC3: a lane git refuses to detach keeps its state and branch, and the reason starts `git refused to detach it: ` followed by git's own first line.
- [x] AC4: `SweepReport.LaneSkips` lists each such lane with its path and reason, and `dotf worktree sweep` prints one `  - skipped: <path>: <reason>` line per entry.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related: #3310 (the behaviours this completes), #3265 (the extraction that follows), ADR-061 (lane identity)
- Related patterns: `00_meta/patterns/pattern-agent-oriented-errors.md` (a refusal names its cause)
